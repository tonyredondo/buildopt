package dev.buildopt.fixtures.correlation;

import java.io.IOException;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.TreeMap;
import java.util.regex.Pattern;
import org.gradle.api.internal.tasks.execution.ExecuteTaskBuildOperationType;
import org.gradle.caching.internal.operations.BuildCacheRemoteStoreBuildOperationType;
import org.gradle.internal.operations.trace.BuildOperationRecord;
import org.gradle.internal.operations.trace.BuildOperationTrace;
import org.gradle.internal.operations.trace.BuildOperationTree;

/**
 * Converts Gradle's version-pinned operation trace into a fail-closed
 * task-execution to remote-PUT correlation report.
 */
public final class CorrelationTraceAnalyzer {
    private static final String FIXTURE_TASK_PREFIX =
            CorrelationFixturePlugin.class.getName() + "$";
    private static final Pattern CACHE_KEY = Pattern.compile("[0-9a-f]{32,64}");
    private static final Set<String> SUCCESS_TASK_CLASSES = Set.of(
            FIXTURE_TASK_PREFIX + "CorrelationFixtureTask",
            FIXTURE_TASK_PREFIX + "NoIsolationWorkerFixtureTask",
            FIXTURE_TASK_PREFIX + "ProcessIsolationWorkerFixtureTask",
            FIXTURE_TASK_PREFIX + "ChildProcessFixtureTask");

    private CorrelationTraceAnalyzer() {}

    public static void main(String[] arguments) throws Exception {
        if (arguments.length == 1 && "--self-test".equals(arguments[0])) {
            selfTest();
            System.out.println(
                    "SPK-001 analyzer self-test OK: exact, missing, and ambiguous ancestry");
            return;
        }
        if (arguments.length != 4) {
            throw new IllegalArgumentException(
                    "usage: CorrelationTraceAnalyzer "
                            + "<trace-base> <miss|hit|failure|cancellation> "
                            + "<http-put-log> <report>");
        }

        Path traceBase = Path.of(arguments[0]).toAbsolutePath().normalize();
        String scenario = arguments[1];
        Path putLog = Path.of(arguments[2]).toAbsolutePath().normalize();
        Path report = Path.of(arguments[3]).toAbsolutePath().normalize();
        Analysis analysis = analyze(readTree(traceBase), scenario, readPutLog(putLog));
        writeReport(report, scenario, analysis);

        System.out.printf(
                "SPK-001 trace %s: scenario=%s fixtureTasks=%d remotePuts=%d%n",
                analysis.capability(),
                scenario,
                analysis.fixtureTasks().size(),
                analysis.puts().size());
        if (!analysis.issues().isEmpty()) {
            analysis.issues().forEach(issue -> System.err.println("correlation: " + issue));
            boolean fallbackOnly = analysis.issues().stream()
                    .allMatch(issue -> issue.startsWith("UNATTRIBUTED PUT "));
            System.exit(fallbackOnly ? 2 : 1);
        }
    }

    private static BuildOperationTree readTree(Path traceBase)
            throws ReflectiveOperationException {
        Method reader;
        try {
            reader = BuildOperationTrace.class.getMethod(
                    "readTree",
                    String.class);
        } catch (NoSuchMethodException ignored) {
            reader = BuildOperationTrace.class.getMethod(
                    "read",
                    String.class);
        }
        try {
            return (BuildOperationTree) reader.invoke(null, traceBase.toString());
        } catch (InvocationTargetException exception) {
            Throwable cause = exception.getCause();
            if (cause instanceof RuntimeException runtimeException) {
                throw runtimeException;
            }
            if (cause instanceof Error error) {
                throw error;
            }
            throw exception;
        }
    }

    private static List<String> readPutLog(Path putLog) throws IOException {
        if (!Files.isRegularFile(putLog)) {
            throw new IOException("HTTP PUT log is missing: " + putLog);
        }
        List<String> keys = new ArrayList<>();
        for (String line : Files.readAllLines(putLog, StandardCharsets.UTF_8)) {
            if (line.isBlank()) {
                continue;
            }
            String key = line.trim();
            if (!CACHE_KEY.matcher(key).matches()) {
                throw new IOException("invalid HTTP PUT key: " + key);
            }
            keys.add(key);
        }
        return keys;
    }

    private static Analysis analyze(
            BuildOperationTree tree,
            String scenario,
            List<String> httpPutKeys) {
        if (!Set.of("miss", "hit", "failure", "cancellation").contains(scenario)) {
            throw new IllegalArgumentException("unsupported scenario: " + scenario);
        }

        Map<Long, Long> parents = new HashMap<>();
        Map<Long, TaskExecution> tasks = new HashMap<>();
        List<RawPut> rawPuts = new ArrayList<>();
        List<BuildOperationRecord> records =
                new ArrayList<>(tree.records.values());
        records.sort(Comparator.comparing(record -> record.id));

        for (BuildOperationRecord record : records) {
            parents.put(record.id, record.parentId);
            if (hasDetails(record, ExecuteTaskBuildOperationType.Details.class)) {
                tasks.put(record.id, taskExecution(record));
            }
            if (hasDetails(
                    record,
                    BuildCacheRemoteStoreBuildOperationType.Details.class)) {
                rawPuts.add(new RawPut(
                        record.id,
                        record.parentId,
                        stringValue(record.details, "cacheKey"),
                        booleanValue(record.result, "stored")));
            }
        }

        Attribution attribution = attribute(
                parents,
                tasks.keySet(),
                rawPuts);
        List<String> issues = new ArrayList<>(attribution.issues());
        Map<Long, Integer> putsByTask = new HashMap<>();
        for (CachePut put : attribution.puts()) {
            if (put.taskExecutionId() != null) {
                putsByTask.merge(put.taskExecutionId(), 1, Integer::sum);
                TaskExecution task = tasks.get(put.taskExecutionId());
                if (task == null || task.outcome() == Outcome.UNKNOWN) {
                    issues.add(
                            "PUT "
                                    + put.operationId()
                                    + " lacks a completed task outcome");
                }
            }
            if (!put.stored()) {
                issues.add("remote store operation " + put.operationId() + " was not stored");
            }
        }

        List<String> tracedKeys = attribution.puts().stream()
                .filter(CachePut::stored)
                .map(CachePut::key)
                .sorted()
                .toList();
        List<String> observedKeys = httpPutKeys.stream().sorted().toList();
        if (!tracedKeys.equals(observedKeys)) {
            issues.add(
                    "HTTP PUT multiset does not match remote store operations: "
                            + "trace="
                            + tracedKeys
                            + " http="
                            + observedKeys);
        }

        List<TaskExecution> fixtureTasks = tasks.values().stream()
                .filter(task -> task.taskClass().startsWith(FIXTURE_TASK_PREFIX))
                .sorted(Comparator.comparingLong(TaskExecution::operationId))
                .toList();
        validateScenario(
                scenario,
                fixtureTasks,
                putsByTask,
                attribution.puts(),
                issues);

        String capability = capability(issues);
        return new Analysis(
                capability,
                attemptAborted(issues),
                fixtureTasks,
                attribution.puts(),
                List.copyOf(issues));
    }

    private static TaskExecution taskExecution(BuildOperationRecord record) {
        long gradleTaskId = numberValue(record.details, "taskId");
        String taskPath = stringValue(record.details, "taskPath");
        String taskClass = stringValue(record.details, "taskClass");
        return new TaskExecution(
                record.id,
                gradleTaskId,
                taskPath,
                taskClass,
                outcome(record));
    }

    private static Outcome outcome(BuildOperationRecord record) {
        if (record.failure != null) {
            if (record.failure.contains("BuildCancelledException")) {
                return Outcome.CANCELLED;
            }
            return Outcome.FAILED;
        }
        if (record.result == null) {
            return Outcome.UNKNOWN;
        }
        Object skipMessage = record.result.get("skipMessage");
        if ("FROM-CACHE".equals(skipMessage)) {
            return Outcome.FROM_CACHE;
        }
        if (Boolean.TRUE.equals(record.result.get("actionable"))
                && skipMessage == null) {
            return Outcome.SUCCESS;
        }
        return Outcome.SKIPPED;
    }

    private static boolean hasDetails(
            BuildOperationRecord record,
            Class<?> detailsType) {
        try {
            return record.hasDetailsOfType(detailsType);
        } catch (ClassNotFoundException ignored) {
            return false;
        }
    }

    private static Attribution attribute(
            Map<Long, Long> parents,
            Collection<Long> taskOperationIds,
            List<RawPut> rawPuts) {
        Set<Long> tasks = new HashSet<>(taskOperationIds);
        List<CachePut> puts = new ArrayList<>();
        List<String> issues = new ArrayList<>();

        for (RawPut rawPut : rawPuts) {
            LinkedHashSet<Long> ancestors = new LinkedHashSet<>();
            Long current = rawPut.parentId();
            Set<Long> visited = new HashSet<>();
            while (current != null && visited.add(current)) {
                if (tasks.contains(current)) {
                    ancestors.add(current);
                }
                current = parents.get(current);
            }
            if (current != null) {
                issues.add(
                        "operation ancestry contains a cycle for PUT "
                                + rawPut.operationId());
            }

            Long taskExecutionId = null;
            if (ancestors.size() == 1) {
                taskExecutionId = ancestors.iterator().next();
            } else {
                issues.add(
                        "UNATTRIBUTED PUT "
                                + rawPut.operationId()
                                + " has "
                                + ancestors.size()
                                + " task ancestors");
            }
            puts.add(new CachePut(
                    rawPut.operationId(),
                    rawPut.key(),
                    taskExecutionId,
                    rawPut.stored()));
        }
        return new Attribution(List.copyOf(puts), List.copyOf(issues));
    }

    private static void validateScenario(
            String scenario,
            List<TaskExecution> fixtureTasks,
            Map<Long, Integer> putsByTask,
            List<CachePut> puts,
            List<String> issues) {
        switch (scenario) {
            case "miss" -> validateMiss(
                    fixtureTasks,
                    putsByTask,
                    puts,
                    issues);
            case "hit" -> validateHit(
                    fixtureTasks,
                    putsByTask,
                    issues);
            case "failure" -> validateTerminal(
                    fixtureTasks,
                    putsByTask,
                    FIXTURE_TASK_PREFIX + "FailureFixtureTask",
                    Outcome.FAILED,
                    issues);
            case "cancellation" -> validateCancellation(
                    fixtureTasks,
                    putsByTask,
                    issues);
            default -> throw new IllegalStateException(
                    "unhandled scenario: " + scenario);
        }
    }

    private static void validateMiss(
            List<TaskExecution> fixtureTasks,
            Map<Long, Integer> putsByTask,
            List<CachePut> puts,
            List<String> issues) {
        if (fixtureTasks.size() != 8) {
            issues.add("miss scenario expected 8 fixture tasks, found " + fixtureTasks.size());
        }
        Map<String, List<String>> keysByClass = new TreeMap<>();
        Map<Long, String> keyByTask = new HashMap<>();
        for (CachePut put : puts) {
            if (put.taskExecutionId() != null) {
                keyByTask.put(put.taskExecutionId(), put.key());
            }
        }
        for (TaskExecution task : fixtureTasks) {
            if (!SUCCESS_TASK_CLASSES.contains(task.taskClass())) {
                issues.add("miss scenario executed unexpected fixture task " + task.taskClass());
            }
            if (task.outcome() != Outcome.SUCCESS
                    && task.outcome() != Outcome.FROM_CACHE) {
                issues.add(
                        "miss task "
                                + task.taskPath()
                                + " has outcome "
                                + task.outcome());
            }
            int putCount = putsByTask.getOrDefault(task.operationId(), 0);
            int expectedPutCount = task.outcome() == Outcome.SUCCESS ? 1 : 0;
            if (putCount != expectedPutCount) {
                issues.add(
                        "miss task "
                                + task.taskPath()
                                + " has "
                                + putCount
                                + " PUTs, expected "
                                + expectedPutCount);
            }
            String key = keyByTask.get(task.operationId());
            if (key != null) {
                keysByClass.computeIfAbsent(
                        task.taskClass(),
                        ignored -> new ArrayList<>()).add(key);
            }
        }
        for (String taskClass : SUCCESS_TASK_CLASSES) {
            List<String> keys = keysByClass.getOrDefault(taskClass, List.of());
            int minimumKeys = taskClass.endsWith("$CorrelationFixtureTask") ? 2 : 1;
            if (keys.size() < minimumKeys
                    || keys.stream().distinct().count() != 1) {
                issues.add(
                        "miss-path "
                                + taskClass
                                + " executions did not expose one shared key: "
                                + keys);
            }
        }
    }

    private static void validateHit(
            List<TaskExecution> fixtureTasks,
            Map<Long, Integer> putsByTask,
            List<String> issues) {
        if (fixtureTasks.size() != 8) {
            issues.add("hit scenario expected 8 fixture tasks, found " + fixtureTasks.size());
        }
        for (TaskExecution task : fixtureTasks) {
            if (!SUCCESS_TASK_CLASSES.contains(task.taskClass())) {
                issues.add("hit scenario executed unexpected fixture task " + task.taskClass());
            }
            if (task.outcome() != Outcome.FROM_CACHE) {
                issues.add(
                        "hit task "
                                + task.taskPath()
                                + " has outcome "
                                + task.outcome());
            }
            int putCount = putsByTask.getOrDefault(task.operationId(), 0);
            if (putCount != 0) {
                issues.add(
                        "cache-hit task "
                                + task.taskPath()
                                + " emitted "
                                + putCount
                                + " PUTs");
            }
        }
    }

    private static void validateTerminal(
            List<TaskExecution> fixtureTasks,
            Map<Long, Integer> putsByTask,
            String taskClass,
            Outcome outcome,
            List<String> issues) {
        if (fixtureTasks.size() != 1) {
            issues.add(
                    "terminal scenario expected 1 fixture task, found "
                            + fixtureTasks.size());
            return;
        }
        TaskExecution task = fixtureTasks.get(0);
        if (!taskClass.equals(task.taskClass())) {
            issues.add("terminal scenario executed " + task.taskClass());
        }
        if (task.outcome() != outcome) {
            issues.add(
                    "terminal task "
                            + task.taskPath()
                            + " has outcome "
                            + task.outcome()
                            + ", expected "
                            + outcome);
        }
        int putCount = putsByTask.getOrDefault(task.operationId(), 0);
        if (putCount != 0) {
            issues.add(
                    "terminal task "
                            + task.taskPath()
                            + " emitted "
                            + putCount
                            + " PUTs");
        }
    }

    private static void validateCancellation(
            List<TaskExecution> fixtureTasks,
            Map<Long, Integer> putsByTask,
            List<String> issues) {
        if (fixtureTasks.size() != 1) {
            issues.add(
                    "cancellation scenario expected 1 fixture task, found "
                            + fixtureTasks.size());
            return;
        }
        TaskExecution task = fixtureTasks.get(0);
        if (!(FIXTURE_TASK_PREFIX + "CancellationFixtureTask")
                .equals(task.taskClass())) {
            issues.add("cancellation scenario executed " + task.taskClass());
        }
        if (task.outcome() != Outcome.CANCELLED
                && task.outcome() != Outcome.FAILED) {
            issues.add(
                    "cancellation task "
                            + task.taskPath()
                            + " has outcome "
                            + task.outcome());
        }
        int putCount = putsByTask.getOrDefault(task.operationId(), 0);
        if (putCount != 0) {
            issues.add(
                    "cancellation task "
                            + task.taskPath()
                            + " emitted "
                            + putCount
                            + " PUTs");
        }
    }

    private static long numberValue(Map<String, ?> values, String key) {
        Object value = values.get(key);
        if (value instanceof Number number) {
            return number.longValue();
        }
        throw new IllegalArgumentException("missing numeric trace field: " + key);
    }

    private static String stringValue(Map<String, ?> values, String key) {
        Object value = values.get(key);
        if (value instanceof String string) {
            return string;
        }
        throw new IllegalArgumentException("missing string trace field: " + key);
    }

    private static boolean booleanValue(Map<String, ?> values, String key) {
        if (values == null) {
            return false;
        }
        return Boolean.TRUE.equals(values.get(key));
    }

    private static void selfTest() {
        RawPut put = new RawPut(300, 200L, "0123456789abcdef0123456789abcdef", true);
        Attribution exact = attribute(
                Map.of(200L, 100L, 100L, 1L),
                Set.of(100L),
                List.of(put));
        require(
                exact.issues().isEmpty()
                        && Objects.equals(
                                exact.puts().get(0).taskExecutionId(),
                                100L),
                "exact ancestry did not resolve");

        Attribution missing = attribute(
                Map.of(200L, 1L),
                Set.of(100L),
                List.of(put));
        require(
                !missing.issues().isEmpty()
                        && missing.puts().get(0).taskExecutionId() == null
                        && "UNAVAILABLE".equals(capability(missing.issues()))
                        && attemptAborted(missing.issues()),
                "missing ancestry did not fail closed");

        Attribution ambiguous = attribute(
                Map.of(200L, 150L, 150L, 100L, 100L, 1L),
                Set.of(100L, 150L),
                List.of(put));
        require(
                !ambiguous.issues().isEmpty()
                        && ambiguous.puts().get(0).taskExecutionId() == null
                        && "UNAVAILABLE".equals(capability(ambiguous.issues()))
                        && attemptAborted(ambiguous.issues()),
                "ambiguous ancestry did not fail closed");
    }

    private static String capability(List<String> issues) {
        return issues.isEmpty() ? "EXACT" : "UNAVAILABLE";
    }

    private static boolean attemptAborted(List<String> issues) {
        return !issues.isEmpty();
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalStateException(message);
        }
    }

    private static void writeReport(
            Path report,
            String scenario,
            Analysis analysis) throws IOException {
        StringBuilder json = new StringBuilder();
        json.append("{\n")
                .append("  \"schemaVersion\": \"buildopt.dev/gradle-correlation-spike/v1\",\n")
                .append("  \"scenario\": ")
                .append(quote(scenario))
                .append(",\n")
                .append("  \"capability\": ")
                .append(quote(analysis.capability()))
                .append(",\n")
                .append("  \"attemptAborted\": ")
                .append(analysis.attemptAborted())
                .append(",\n")
                .append("  \"taskExecutions\": [\n");
        for (int index = 0; index < analysis.fixtureTasks().size(); index++) {
            TaskExecution task = analysis.fixtureTasks().get(index);
            json.append("    {\"taskExecutionId\": ")
                    .append(quote(Long.toString(task.operationId())))
                    .append(", \"gradleTaskId\": ")
                    .append(task.gradleTaskId())
                    .append(", \"taskPath\": ")
                    .append(quote(task.taskPath()))
                    .append(", \"taskClass\": ")
                    .append(quote(task.taskClass()))
                    .append(", \"outcome\": ")
                    .append(quote(task.outcome().name()))
                    .append("}");
            json.append(index + 1 == analysis.fixtureTasks().size() ? "\n" : ",\n");
        }
        json.append("  ],\n  \"puts\": [\n");
        for (int index = 0; index < analysis.puts().size(); index++) {
            CachePut put = analysis.puts().get(index);
            json.append("    {\"operationId\": ")
                    .append(quote(Long.toString(put.operationId())))
                    .append(", \"cacheKey\": ")
                    .append(quote(put.key()))
                    .append(", \"taskExecutionId\": ");
            if (put.taskExecutionId() == null) {
                json.append("null");
            } else {
                json.append(quote(Long.toString(put.taskExecutionId())));
            }
            json.append(", \"stored\": ")
                    .append(put.stored())
                    .append("}");
            json.append(index + 1 == analysis.puts().size() ? "\n" : ",\n");
        }
        json.append("  ],\n  \"issues\": [");
        for (int index = 0; index < analysis.issues().size(); index++) {
            if (index != 0) {
                json.append(", ");
            }
            json.append(quote(analysis.issues().get(index)));
        }
        json.append("]\n}\n");

        Files.createDirectories(report.getParent());
        Files.writeString(
                report,
                json.toString(),
                StandardCharsets.UTF_8);
    }

    private static String quote(String value) {
        StringBuilder escaped = new StringBuilder(value.length() + 2);
        escaped.append('"');
        for (int index = 0; index < value.length(); index++) {
            char character = value.charAt(index);
            switch (character) {
                case '"' -> escaped.append("\\\"");
                case '\\' -> escaped.append("\\\\");
                case '\b' -> escaped.append("\\b");
                case '\f' -> escaped.append("\\f");
                case '\n' -> escaped.append("\\n");
                case '\r' -> escaped.append("\\r");
                case '\t' -> escaped.append("\\t");
                default -> {
                    if (character < 0x20) {
                        escaped.append(String.format("\\u%04x", (int) character));
                    } else {
                        escaped.append(character);
                    }
                }
            }
        }
        return escaped.append('"').toString();
    }

    private enum Outcome {
        SUCCESS,
        FROM_CACHE,
        FAILED,
        CANCELLED,
        SKIPPED,
        UNKNOWN
    }

    private record TaskExecution(
            long operationId,
            long gradleTaskId,
            String taskPath,
            String taskClass,
            Outcome outcome) {}

    private record RawPut(
            long operationId,
            Long parentId,
            String key,
            boolean stored) {}

    private record CachePut(
            long operationId,
            String key,
            Long taskExecutionId,
            boolean stored) {}

    private record Attribution(
            List<CachePut> puts,
            List<String> issues) {}

    private record Analysis(
            String capability,
            boolean attemptAborted,
            List<TaskExecution> fixtureTasks,
            List<CachePut> puts,
            List<String> issues) {}
}
