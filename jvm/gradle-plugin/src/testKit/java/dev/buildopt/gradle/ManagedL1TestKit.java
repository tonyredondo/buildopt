package dev.buildopt.gradle;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.FileVisitResult;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.StandardCopyOption;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.HashMap;
import java.util.Map;
import java.util.stream.Stream;
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** Functional conformance for the launcher-owned native managed L1. */
public final class ManagedL1TestKit {
    private static final String SCOPE =
            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
    private static final String TASK_DEFAULT_DENY_REASON =
            "BuildOpt Tier 1 default-deny allowlist rejected this task";
    private static final String DIRECTORY_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_DIRECTORY";
    private static final String MODE_ENVIRONMENT = "BUILDOPT_MANAGED_L1_MODE";
    private static final String GENERATION_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_SECURITY_GENERATION";
    private static final String RETENTION_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_RETENTION_DAYS";
    private static final String READ_WRITE_MODE = "READ_WRITE";
    private static final String DISABLED_L2_WRITER_MODE =
            "DISABLED_L2_WRITER";
    private static final int RETENTION_DAYS = 7;

    private ManagedL1TestKit() {}

    /**
     * Runs generation rotation on every row and fail-closed modes on the golden
     * runtime.
     *
     * @param arguments fixture root, Gradle home, plugin JAR, and Java major
     * @throws IOException when a fixture cannot be copied or inspected
     */
    public static void main(String[] arguments) throws IOException {
        if (arguments.length != 4) {
            throw new IllegalArgumentException(
                    "expected fixture root, Gradle home, plugin JAR, and Java major");
        }
        Path fixtureRoot = Path.of(arguments[0]).toAbsolutePath().normalize();
        File gradleHome = Path.of(arguments[1]).toAbsolutePath().normalize().toFile();
        Path pluginJar = Path.of(arguments[2]).toAbsolutePath().normalize();
        int expectedJava = Integer.parseInt(arguments[3]);
        if (Runtime.version().feature() != expectedJava) {
            throw new IllegalStateException(
                    "TestKit Java "
                            + Runtime.version().feature()
                            + " does not match "
                            + expectedJava);
        }
        if (!gradleHome.toPath().resolve("bin/gradle").toFile().canExecute()) {
            throw new IllegalArgumentException("invalid Gradle home: " + gradleHome);
        }
        if (!Files.isRegularFile(pluginJar)) {
            throw new IllegalArgumentException("missing plugin JAR: " + pluginJar);
        }

        Path testRoot = Files.createTempDirectory("buildopt-managed-l1-");
        try {
            copyTree(fixtureRoot, testRoot.resolve("tier1"));
            Path initScript = writeInitScript(testRoot, pluginJar);
            for (String dsl : new String[] {"kotlin", "groovy"}) {
                Path project = testRoot.resolve("tier1").resolve(dsl);
                runGenerationRotation(
                        project,
                        testRoot.resolve("state-" + dsl),
                        initScript,
                        gradleHome,
                        dsl);
                if (gradleHome.getName().equals("gradle-9.6.1")
                        && expectedJava == 21) {
                    runDisabledL2Writer(
                            project,
                            initScript,
                            gradleHome,
                            dsl);
                    runInvalidContext(
                            project,
                            initScript,
                            gradleHome,
                            dsl);
                }
            }
        } finally {
            deleteTree(testRoot);
        }
        System.out.printf(
                "A0-003 TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy%n",
                gradleHome.getName(),
                expectedJava);
    }

    private static Path writeInitScript(Path testRoot, Path pluginJar)
            throws IOException {
        Path initScript = testRoot.resolve("buildopt-managed-l1.init.gradle");
        Files.writeString(
                initScript,
                "initscript { dependencies { classpath files('"
                        + groovyQuote(pluginJar)
                        + "') } }\n"
                        + "beforeSettings { settings ->\n"
                        + "    settings.pluginManager.apply("
                        + "dev.buildopt.gradle.BuildOptManagedL1Plugin)\n"
                        + "}\n",
                StandardCharsets.UTF_8);
        return initScript;
    }

    private static void runGenerationRotation(
            Path project,
            Path stateRoot,
            Path initScript,
            File gradleHome,
            String dsl)
            throws IOException {
        String scenario = dsl + "-rotation";
        Path generation42 = managedDirectory(stateRoot, 42);
        Path generation43 = managedDirectory(stateRoot, 43);
        Files.createDirectories(generation42);
        Files.createDirectories(generation43);
        String[] arguments =
                arguments(
                        initScript,
                        "--build-cache",
                        "-PbuildoptTierOneRegisterTransform=false",
                        "clean",
                        "compileJava",
                        "compileTestJava",
                        "unknownCacheable");

        BuildResult first =
                run(
                        project,
                        gradleHome,
                        scenario,
                        readWriteEnvironment(generation42, 42),
                        arguments);
        requireOutcome(first, ":compileJava", TaskOutcome.SUCCESS, scenario);
        requireOutcome(first, ":compileTestJava", TaskOutcome.SUCCESS, scenario);
        requireOutcome(first, ":unknownCacheable", TaskOutcome.SUCCESS, scenario);
        requireCacheContent(generation42, scenario);

        BuildResult second =
                run(
                        project,
                        gradleHome,
                        scenario,
                        readWriteEnvironment(generation42, 42),
                        arguments);
        requireOutcome(second, ":compileJava", TaskOutcome.FROM_CACHE, scenario);
        requireOutcome(second, ":compileTestJava", TaskOutcome.FROM_CACHE, scenario);
        requireOutcome(second, ":unknownCacheable", TaskOutcome.SUCCESS, scenario);
        requireConfigurationCacheReuse(second, scenario);
        requireReason(second, TASK_DEFAULT_DENY_REASON, scenario);

        BuildResult rotated =
                run(
                        project,
                        gradleHome,
                        scenario,
                        readWriteEnvironment(generation43, 43),
                        arguments);
        requireOutcome(rotated, ":compileJava", TaskOutcome.SUCCESS, scenario);
        requireOutcome(rotated, ":compileTestJava", TaskOutcome.SUCCESS, scenario);
        requireOutcome(rotated, ":unknownCacheable", TaskOutcome.SUCCESS, scenario);
        if (rotated.getOutput().contains("Configuration cache entry reused.")) {
            throw new IllegalStateException(
                    scenario
                            + " reused Configuration Cache across an L1 generation\n"
                            + rotated.getOutput());
        }
        requireCacheContent(generation43, scenario);

        BuildResult rotatedReplay =
                run(
                        project,
                        gradleHome,
                        scenario,
                        readWriteEnvironment(generation43, 43),
                        arguments);
        requireOutcome(
                rotatedReplay,
                ":compileJava",
                TaskOutcome.FROM_CACHE,
                scenario);
        requireOutcome(
                rotatedReplay,
                ":compileTestJava",
                TaskOutcome.FROM_CACHE,
                scenario);
        requireOutcome(
                rotatedReplay,
                ":unknownCacheable",
                TaskOutcome.SUCCESS,
                scenario);
        requireConfigurationCacheReuse(rotatedReplay, scenario);
    }

    private static void runDisabledL2Writer(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl) {
        String scenario = dsl + "-l2-writer";
        String[] arguments =
                arguments(
                        initScript,
                        "--build-cache",
                        "-PbuildoptTierOneRegisterTransform=false",
                        "clean",
                        "compileJava");
        Map<String, String> environment = managedEnvironment(
                DISABLED_L2_WRITER_MODE,
                "",
                44);
        requireOutcome(
                run(project, gradleHome, scenario, environment, arguments),
                ":compileJava",
                TaskOutcome.SUCCESS,
                scenario);
        BuildResult second =
                run(project, gradleHome, scenario, environment, arguments);
        requireOutcome(second, ":compileJava", TaskOutcome.SUCCESS, scenario);
        requireConfigurationCacheReuse(second, scenario);
    }

    private static void runInvalidContext(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl) {
        String scenario = dsl + "-invalid";
        String[] arguments =
                arguments(
                        initScript,
                        "--build-cache",
                        "-PbuildoptTierOneRegisterTransform=false",
                        "clean",
                        "compileJava");
        Map<String, String> environment = managedEnvironment(
                READ_WRITE_MODE,
                "relative/cache",
                45);
        BuildResult first =
                run(project, gradleHome, scenario, environment, arguments);
        requireOutcome(first, ":compileJava", TaskOutcome.SUCCESS, scenario);
        requireReason(first, "Build cache is disabled", scenario);
        BuildResult second =
                run(project, gradleHome, scenario, environment, arguments);
        requireOutcome(second, ":compileJava", TaskOutcome.SUCCESS, scenario);
    }

    private static Path managedDirectory(Path stateRoot, long generation) {
        return stateRoot
                .resolve("l1")
                .resolve("scopes")
                .resolve(SCOPE)
                .resolve("generation-" + generation)
                .resolve("cache")
                .toAbsolutePath()
                .normalize();
    }

    private static Map<String, String> readWriteEnvironment(
            Path directory,
            long generation) {
        return managedEnvironment(
                READ_WRITE_MODE,
                directory.toString(),
                generation);
    }

    private static Map<String, String> managedEnvironment(
            String mode,
            String directory,
        long generation) {
        Map<String, String> environment = new HashMap<>();
        environment.put(MODE_ENVIRONMENT, mode);
        environment.put(
                GENERATION_ENVIRONMENT,
                Long.toString(generation));
        environment.put(
                RETENTION_ENVIRONMENT,
                Integer.toString(RETENTION_DAYS));
        if (!directory.isEmpty()) {
            environment.put(DIRECTORY_ENVIRONMENT, directory);
        }
        return environment;
    }

    private static String[] arguments(
            Path initScript,
            String... scenarioArguments) {
        String[] arguments = new String[scenarioArguments.length + 7];
        System.arraycopy(scenarioArguments, 0, arguments, 0, scenarioArguments.length);
        int index = scenarioArguments.length;
        arguments[index++] = "--init-script";
        arguments[index++] = initScript.toString();
        arguments[index++] = "--configuration-cache";
        arguments[index++] = "--stacktrace";
        arguments[index++] = "--warning-mode=fail";
        arguments[index++] = "--console=plain";
        arguments[index] = "--info";
        return arguments;
    }

    private static BuildResult run(
            Path project,
            File gradleHome,
            String scenario,
            Map<String, String> managedEnvironment,
            String[] arguments) {
        Map<String, String> environment = new HashMap<>(System.getenv());
        environment.keySet().removeIf(key -> key.startsWith("BUILDOPT_"));
        environment.put(
                "GRADLE_USER_HOME",
                project.resolve(".gradle-user-home-" + scenario).toString());
        environment.putAll(managedEnvironment);
        return GradleRunner.create()
                .withProjectDir(project.toFile())
                .withGradleInstallation(gradleHome)
                .withTestKitDir(project.resolve(".test-kit-" + scenario).toFile())
                .withEnvironment(environment)
                .withArguments(arguments)
                .build();
    }

    private static void requireCacheContent(Path directory, String scenario)
            throws IOException {
        try (Stream<Path> entries = Files.walk(directory)) {
            if (entries.noneMatch(path -> !path.equals(directory))) {
                throw new IllegalStateException(
                        scenario + " did not populate managed L1 " + directory);
            }
        }
    }

    private static void requireOutcome(
            BuildResult result,
            String taskPath,
            TaskOutcome expected,
            String scenario) {
        if (result.task(taskPath) == null) {
            throw new IllegalStateException(
                    scenario + " task was absent: " + taskPath + "\n" + result.getOutput());
        }
        TaskOutcome actual = result.task(taskPath).getOutcome();
        if (actual != expected) {
            throw new IllegalStateException(
                    scenario
                            + " task "
                            + taskPath
                            + " was "
                            + actual
                            + ", expected "
                            + expected
                            + "\n"
                            + result.getOutput());
        }
    }

    private static void requireConfigurationCacheReuse(
            BuildResult result,
            String scenario) {
        if (!result.getOutput().contains("Configuration cache entry reused.")) {
            throw new IllegalStateException(
                    scenario + " did not reuse Configuration Cache\n" + result.getOutput());
        }
    }

    private static void requireReason(
            BuildResult result,
            String reason,
            String scenario) {
        if (!result.getOutput().contains(reason)) {
            throw new IllegalStateException(
                    scenario + " did not report " + reason + "\n" + result.getOutput());
        }
    }

    private static String groovyQuote(Path path) {
        return path.toString().replace("\\", "\\\\").replace("'", "\\'");
    }

    private static void copyTree(Path source, Path target) throws IOException {
        Files.walkFileTree(
                source,
                new SimpleFileVisitor<>() {
                    @Override
                    public FileVisitResult preVisitDirectory(
                            Path directory,
                            BasicFileAttributes attributes)
                            throws IOException {
                        if (!directory.equals(source)
                                && isTransientFixtureDirectory(directory)) {
                            return FileVisitResult.SKIP_SUBTREE;
                        }
                        Files.createDirectories(target.resolve(source.relativize(directory)));
                        return FileVisitResult.CONTINUE;
                    }

                    @Override
                    public FileVisitResult visitFile(
                            Path file,
                            BasicFileAttributes attributes)
                            throws IOException {
                        Files.copy(
                                file,
                                target.resolve(source.relativize(file)),
                                StandardCopyOption.COPY_ATTRIBUTES);
                        return FileVisitResult.CONTINUE;
                    }
                });
    }

    private static boolean isTransientFixtureDirectory(Path directory) {
        Path name = directory.getFileName();
        if (name == null) {
            return false;
        }
        String value = name.toString();
        return value.equals(".gradle")
                || value.equals("build")
                || value.startsWith(".gradle-user-home-")
                || value.startsWith(".test-kit-");
    }

    private static void deleteTree(Path root) throws IOException {
        if (!Files.exists(root)) {
            return;
        }
        Files.walkFileTree(
                root,
                new SimpleFileVisitor<>() {
                    @Override
                    public FileVisitResult visitFile(
                            Path file,
                            BasicFileAttributes attributes)
                            throws IOException {
                        Files.delete(file);
                        return FileVisitResult.CONTINUE;
                    }

                    @Override
                    public FileVisitResult postVisitDirectory(
                            Path directory,
                            IOException failure)
                            throws IOException {
                        if (failure != null) {
                            throw failure;
                        }
                        Files.delete(directory);
                        return FileVisitResult.CONTINUE;
                    }
                });
    }
}
