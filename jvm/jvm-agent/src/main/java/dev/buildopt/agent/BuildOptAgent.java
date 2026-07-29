package dev.buildopt.agent;

import java.io.IOException;
import java.lang.instrument.ClassFileTransformer;
import java.lang.instrument.IllegalClassFormatException;
import java.lang.instrument.Instrumentation;
import java.nio.charset.StandardCharsets;
import java.nio.file.AtomicMoveNotSupportedException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.security.ProtectionDomain;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Bounded, fail-closed observation prototype for {@code SPK-002}.
 *
 * <p>This prototype observes only allowlisted class loads. A class load is not evidence that a
 * particular API call occurred, so every report remains incomplete and aborts pending
 * publication. The bounded spike intentionally does not add a bytecode rewriting dependency or
 * claim deep tracing.
 */
public final class BuildOptAgent {
    private static final String SCHEMA = "buildopt.spikes/jvm-agent-report/v1";
    private static final List<CoverageTarget> TARGETS =
            List.of(
                    new CoverageTarget(
                            "IO_NIO",
                            List.of("java/io/", "java/nio/file/")),
                    new CoverageTarget(
                            "ENVIRONMENT_PROPERTIES",
                            List.of("java/lang/System")),
                    new CoverageTarget(
                            "PROCESS",
                            List.of("java/lang/ProcessBuilder", "java/lang/ProcessImpl")),
                    new CoverageTarget(
                            "NETWORK",
                            List.of("java/net/", "java/net/http/")),
                    new CoverageTarget(
                            "CLOCK_LOCALE_TIMEZONE",
                            List.of("java/time/", "java/util/Locale", "java/util/TimeZone")),
                    new CoverageTarget(
                            "RANDOMNESS",
                            List.of(
                                    "java/util/Random",
                                    "java/util/concurrent/ThreadLocalRandom",
                                    "java/security/SecureRandom")));
    private static final String CONFLICT_CLASS =
            "dev/buildopt/fixtures/tierone/TierOneFixturePlugin$AgentProbeTask";

    private BuildOptAgent() {}

    /**
     * Installs the bounded observer when a report argument is supplied.
     *
     * @param agentArguments comma-separated report, capacity, and fault values
     * @param instrumentation JVM instrumentation service
     */
    public static void premain(String agentArguments, Instrumentation instrumentation) {
        Objects.requireNonNull(instrumentation, "instrumentation");
        if (agentArguments == null || agentArguments.isBlank()) {
            return;
        }

        AgentConfiguration configuration;
        try {
            configuration = AgentConfiguration.parse(agentArguments);
        } catch (IllegalArgumentException exception) {
            System.err.println("[buildopt-agent] disabled: " + exception.getMessage());
            return;
        }
        AgentState state = new AgentState(configuration.capacity());
        if (configuration.fault() == Fault.CRASH) {
            state.markCrash();
            writeReport(configuration.report(), state, configuration.fault());
            throw new IllegalStateException("SPK-002 injected premain crash");
        }
        if (configuration.fault() == Fault.CONFLICT) {
            instrumentation.addTransformer(new ConflictTransformer(state), false);
        }
        instrumentation.addTransformer(new ObservingTransformer(state), false);
        Runtime.getRuntime()
                .addShutdownHook(
                        new Thread(
                                () -> writeReport(
                                        configuration.report(),
                                        state,
                                        configuration.fault()),
                                "buildopt-agent-report"));
    }

    private static void writeReport(Path report, AgentState state, Fault configuredFault) {
        try {
            Path parent = report.getParent();
            if (parent == null) {
                throw new IOException("report path has no parent");
            }
            Files.createDirectories(parent);
            Path temporary =
                    Files.createTempFile(parent, report.getFileName().toString(), ".tmp");
            Files.writeString(
                    temporary,
                    renderReport(state, configuredFault),
                    StandardCharsets.UTF_8);
            try {
                Files.move(
                        temporary,
                        report,
                        StandardCopyOption.ATOMIC_MOVE,
                        StandardCopyOption.REPLACE_EXISTING);
            } catch (AtomicMoveNotSupportedException exception) {
                Files.move(
                        temporary,
                        report,
                        StandardCopyOption.REPLACE_EXISTING);
            }
        } catch (IOException | RuntimeException exception) {
            System.err.println(
                    "[buildopt-agent] incomplete report: " + exception.getMessage());
        }
    }

    private static String renderReport(AgentState state, Fault configuredFault) {
        String reason;
        String fallback = "ABORT_PENDING";
        if (state.crashed()) {
            reason = "INJECTED_AGENT_CRASH";
            fallback = "DISABLE_COMPATIBILITY_CLASS_AND_RUN_BASELINE";
        } else if (state.dropped() > 0) {
            reason = "BUFFER_OVERFLOW";
        } else if (state.conflict()) {
            reason = "TRANSFORMER_CONFLICT";
        } else {
            reason = "CLASS_LOAD_ONLY_NO_ACCESS_INSTRUMENTATION";
        }

        StringBuilder json = new StringBuilder();
        json.append("{\n")
                .append("  \"schemaVersion\": \"")
                .append(SCHEMA)
                .append("\",\n")
                .append("  \"mode\": \"TRACE_OBSERVE\",\n")
                .append("  \"traceComplete\": false,\n")
                .append("  \"qualification\": \"UNAVAILABLE\",\n")
                .append("  \"taskQualificationState\": \"OBSERVING\",\n")
                .append("  \"pendingPublication\": \"ABORTED\",\n")
                .append("  \"fallback\": \"")
                .append(fallback)
                .append("\",\n")
                .append("  \"reason\": \"")
                .append(reason)
                .append("\",\n")
                .append("  \"configuredFault\": \"")
                .append(configuredFault)
                .append("\",\n")
                .append("  \"capacity\": ")
                .append(state.capacity())
                .append(",\n")
                .append("  \"recorded\": ")
                .append(state.events().size())
                .append(",\n")
                .append("  \"dropped\": ")
                .append(state.dropped())
                .append(",\n")
                .append("  \"coverage\": {");
        for (int index = 0; index < TARGETS.size(); index++) {
            CoverageTarget target = TARGETS.get(index);
            if (index > 0) {
                json.append(',');
            }
            json.append("\n    \"")
                    .append(target.id())
                    .append("\": \"")
                    .append(state.observed(target) ? "LOAD_ONLY" : "UNOBSERVED")
                    .append('"');
        }
        json.append("\n  },\n  \"events\": [");
        List<String> events = state.events();
        for (int index = 0; index < events.size(); index++) {
            if (index > 0) {
                json.append(',');
            }
            json.append("\n    \"").append(jsonEscape(events.get(index))).append('"');
        }
        if (!events.isEmpty()) {
            json.append('\n');
        }
        return json.append("  ]\n}\n").toString();
    }

    private static String jsonEscape(String value) {
        StringBuilder result = new StringBuilder(value.length());
        for (int index = 0; index < value.length(); index++) {
            char current = value.charAt(index);
            switch (current) {
                case '"' -> result.append("\\\"");
                case '\\' -> result.append("\\\\");
                case '\b' -> result.append("\\b");
                case '\f' -> result.append("\\f");
                case '\n' -> result.append("\\n");
                case '\r' -> result.append("\\r");
                case '\t' -> result.append("\\t");
                default -> {
                    if (current < 0x20) {
                        result.append(
                                String.format(Locale.ROOT, "\\u%04x", (int) current));
                    } else {
                        result.append(current);
                    }
                }
            }
        }
        return result.toString();
    }

    private static final class ObservingTransformer implements ClassFileTransformer {
        private final AgentState state;

        private ObservingTransformer(AgentState state) {
            this.state = state;
        }

        @Override
        public byte[] transform(
                Module module,
                ClassLoader loader,
                String className,
                Class<?> classBeingRedefined,
                ProtectionDomain protectionDomain,
                byte[] classfileBuffer) {
            if (className != null && isAllowlisted(className)) {
                state.record(className);
            }
            return null;
        }
    }

    private static final class ConflictTransformer implements ClassFileTransformer {
        private final AgentState state;
        private final AtomicBoolean injected = new AtomicBoolean();

        private ConflictTransformer(AgentState state) {
            this.state = state;
        }

        @Override
        public byte[] transform(
                Module module,
                ClassLoader loader,
                String className,
                Class<?> classBeingRedefined,
                ProtectionDomain protectionDomain,
                byte[] classfileBuffer)
                throws IllegalClassFormatException {
            if (CONFLICT_CLASS.equals(className) && injected.compareAndSet(false, true)) {
                state.markConflict();
                throw new IllegalClassFormatException(
                        "SPK-002 injected transformation conflict");
            }
            return null;
        }
    }

    private static boolean isAllowlisted(String className) {
        if (className.startsWith("dev/buildopt/fixtures/tierone/")) {
            return true;
        }
        for (CoverageTarget target : TARGETS) {
            if (target.matches(className)) {
                return true;
            }
        }
        return false;
    }

    private record CoverageTarget(String id, List<String> prefixes) {
        private boolean matches(String className) {
            for (String prefix : prefixes) {
                if (className.startsWith(prefix)) {
                    return true;
                }
            }
            return false;
        }
    }

    private enum Fault {
        NONE,
        CONFLICT,
        CRASH
    }

    private record AgentConfiguration(Path report, int capacity, Fault fault) {
        private static AgentConfiguration parse(String arguments) {
            Map<String, String> values = new LinkedHashMap<>();
            for (String item : arguments.split(",", -1)) {
                int separator = item.indexOf('=');
                if (separator <= 0 || separator == item.length() - 1) {
                    throw new IllegalArgumentException("malformed agent argument");
                }
                String key = item.substring(0, separator);
                String value = item.substring(separator + 1);
                if (!List.of("report", "capacity", "fault").contains(key)
                        || values.putIfAbsent(key, value) != null) {
                    throw new IllegalArgumentException("unknown or duplicate argument: " + key);
                }
            }
            if (!values.keySet().equals(
                    java.util.Set.of("report", "capacity", "fault"))) {
                throw new IllegalArgumentException("report, capacity, and fault are required");
            }
            Path report = Path.of(values.get("report")).toAbsolutePath().normalize();
            int capacity;
            try {
                capacity = Integer.parseInt(values.get("capacity"));
            } catch (NumberFormatException exception) {
                throw new IllegalArgumentException("capacity must be an integer", exception);
            }
            if (capacity < 1 || capacity > 65_536) {
                throw new IllegalArgumentException("capacity must be between 1 and 65536");
            }
            Fault fault;
            try {
                fault = Fault.valueOf(values.get("fault").toUpperCase(Locale.ROOT));
            } catch (IllegalArgumentException exception) {
                throw new IllegalArgumentException("fault must be none, conflict, or crash", exception);
            }
            return new AgentConfiguration(report, capacity, fault);
        }
    }

    private static final class AgentState {
        private final int capacity;
        private final List<String> events = new ArrayList<>();
        private final AtomicInteger dropped = new AtomicInteger();
        private final AtomicBoolean conflict = new AtomicBoolean();
        private final AtomicBoolean crashed = new AtomicBoolean();

        private AgentState(int capacity) {
            this.capacity = capacity;
        }

        private synchronized void record(String className) {
            if (events.contains(className)) {
                return;
            }
            if (events.size() >= capacity) {
                dropped.incrementAndGet();
                return;
            }
            events.add(className);
        }

        private synchronized List<String> events() {
            return List.copyOf(events);
        }

        private boolean observed(CoverageTarget target) {
            for (String event : events()) {
                if (target.matches(event)) {
                    return true;
                }
            }
            return false;
        }

        private int capacity() {
            return capacity;
        }

        private int dropped() {
            return dropped.get();
        }

        private void markConflict() {
            conflict.set(true);
        }

        private boolean conflict() {
            return conflict.get();
        }

        private void markCrash() {
            crashed.set(true);
        }

        private boolean crashed() {
            return crashed.get();
        }
    }
}
