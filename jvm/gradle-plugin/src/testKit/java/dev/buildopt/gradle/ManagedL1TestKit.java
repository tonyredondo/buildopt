package dev.buildopt.gradle;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.File;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.nio.file.FileVisitResult;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.StandardCopyOption;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.HashMap;
import java.util.Map;
import java.util.Base64;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
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
    private static final String SHARED_MODE_ENVIRONMENT =
            "BUILDOPT_MANAGED_SHARED_MODE";
    private static final String AUTHORITY_DIGEST_ENVIRONMENT =
            "BUILDOPT_MANAGED_AUTHORITY_DIGEST";
    private static final String POLICY_DIGEST_ENVIRONMENT =
            "BUILDOPT_MANAGED_POLICY_DIGEST";
    private static final String CONFIGURATION_DIGEST_ENVIRONMENT =
            "BUILDOPT_MANAGED_CONFIGURATION_POLICY_DIGEST";
    private static final String AUTHORITY_CONTRACT_ENVIRONMENT =
            "BUILDOPT_MANAGED_AUTHORITY_CONTRACT";
    private static final String GATEWAY_URL_ENVIRONMENT = "BUILDOPT_GATEWAY_URL";
    private static final String GATEWAY_USERNAME_ENVIRONMENT =
            "BUILDOPT_GATEWAY_USERNAME";
    private static final String GATEWAY_PASSWORD_ENVIRONMENT =
            "BUILDOPT_GATEWAY_PASSWORD";
    private static final String GATEWAY_GENERATION_ENVIRONMENT =
            "BUILDOPT_GATEWAY_CONNECTION_GENERATION";
    private static final int RETENTION_DAYS = 7;

    private ManagedL1TestKit() {}

    /**
     * Runs generation rotation on every row and fail-closed modes on the golden
     * runtime.
     *
     * @param arguments fixture root, Gradle home, plugin JAR, Java major, and
     *     optional focused mode
     * @throws IOException when a fixture cannot be copied or inspected
     */
    public static void main(String[] arguments) throws IOException {
        if (arguments.length < 4
                || arguments.length > 5
                || (arguments.length == 5
                        && !arguments[4].equals("shared-only")
                        && !arguments[4].equals("lifecycle-only")
                        && !arguments[4].equals("circuit-only"))) {
            throw new IllegalArgumentException(
                    "expected fixture root, Gradle home, plugin JAR, Java major, and optional shared-only, lifecycle-only, or circuit-only");
        }
        Path fixtureRoot = Path.of(arguments[0]).toAbsolutePath().normalize();
        File gradleHome = Path.of(arguments[1]).toAbsolutePath().normalize().toFile();
        Path pluginJar = Path.of(arguments[2]).toAbsolutePath().normalize();
        int expectedJava = Integer.parseInt(arguments[3]);
        String requestedMode = arguments.length == 5 ? arguments[4] : "";
        boolean sharedOnly = requestedMode.equals("shared-only");
        boolean lifecycleOnly = requestedMode.equals("lifecycle-only");
        boolean circuitOnly = requestedMode.equals("circuit-only");
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
                if (circuitOnly) {
                    runCircuitBreakerFallback(
                            project,
                            testRoot.resolve("circuit-state-" + dsl),
                            initScript,
                            gradleHome,
                            dsl);
                    continue;
                }
                if (lifecycleOnly) {
                    runL1L2Lifecycle(
                            project,
                            testRoot.resolve("l1-l2-state-" + dsl),
                            initScript,
                            gradleHome,
                            dsl);
                    continue;
                }
                if (sharedOnly) {
                    runManagedShared(
                            project,
                            initScript,
                            gradleHome,
                            dsl);
                    continue;
                }
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
                    runManagedShared(
                            project,
                            initScript,
                            gradleHome,
                            dsl);
                }
            }
        } finally {
            deleteTree(testRoot);
        }
        if (circuitOnly) {
            System.out.printf(
                    "A1-G02 circuit fallback TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy / flood+large-object+full-disk%n",
                    gradleHome.getName(),
                    expectedJava);
        } else if (lifecycleOnly) {
            System.out.printf(
                    "A0-G02 TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy L2-to-L1 revocation and abort%n",
                    gradleHome.getName(),
                    expectedJava);
        } else if (sharedOnly) {
            System.out.printf(
                    "A0-006 TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy HttpBuildCache%n",
                    gradleHome.getName(),
                    expectedJava);
        } else {
            System.out.printf(
                    "A0-003 TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy%n",
                    gradleHome.getName(),
                    expectedJava);
        }
    }

    private static void runCircuitBreakerFallback(
            Path project,
            Path stateRoot,
            Path initScript,
            File gradleHome,
            String dsl)
            throws IOException {
        Path source = project.resolve("src/main/java/example/TierOne.java");
        String original = Files.readString(source, StandardCharsets.UTF_8);
        String[] reasons = {"FLOOD", "OBJECT_TOO_LARGE", "DISK_PRESSURE"};
        String[] arguments =
                arguments(
                        initScript,
                        "--build-cache",
                        "-PbuildoptTierOneRegisterTransform=false",
                        "clean",
                        "compileJava");

        try {
            for (int index = 0; index < reasons.length; index++) {
                String reason = reasons[index];
                long generation = 60L + index;
                Path directory = managedDirectory(stateRoot, generation);
                Files.createDirectories(directory);
                Files.writeString(
                        source,
                        original + "\n// a1-g02-" + reason.toLowerCase() + "\n",
                        StandardCharsets.UTF_8);
                String scenario =
                        dsl
                                + "-circuit-"
                                + reason.toLowerCase().replace('_', '-');
                Map<String, String> environment =
                        readWriteEnvironment(directory, generation);

                BuildResult fallback =
                        run(
                                project,
                                gradleHome,
                                scenario,
                                environment,
                                arguments);
                requireOutcome(
                        fallback,
                        ":compileJava",
                        TaskOutcome.SUCCESS,
                        scenario + "-fallback");
                requireCacheContent(directory, scenario + "-fallback");

                BuildResult replay =
                        run(
                                project,
                                gradleHome,
                                scenario,
                                environment,
                                arguments);
                requireOutcome(
                        replay,
                        ":compileJava",
                        TaskOutcome.FROM_CACHE,
                        scenario + "-replay");
                requireConfigurationCacheReuse(replay, scenario + "-replay");
            }
        } finally {
            Files.writeString(source, original, StandardCharsets.UTF_8);
        }
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

    private static void runManagedShared(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl)
            throws IOException {
        String scenario = dsl + "-managed-shared";
        String gatewayPassword =
                Base64.getUrlEncoder()
                        .withoutPadding()
                        .encodeToString(new byte[32]);
        String basicAuthorization =
                "Basic "
                        + Base64.getEncoder()
                                .encodeToString(
                                        ("buildopt:" + gatewayPassword)
                                                .getBytes(StandardCharsets.UTF_8));
        Map<String, byte[]> objects = new ConcurrentHashMap<>();
        AtomicInteger gets = new AtomicInteger();
        AtomicInteger puts = new AtomicInteger();
        HttpServer server =
                HttpServer.create(
                        new InetSocketAddress("127.0.0.1", 0),
                        0);
        server.createContext(
                "/cache/",
                exchange ->
                        serveCache(
                                exchange,
                                basicAuthorization,
                                objects,
                                gets,
                                puts));
        server.start();
        try {
            Map<String, String> environment =
                    managedEnvironment(DISABLED_L2_WRITER_MODE, "", 46);
            environment.put(SHARED_MODE_ENVIRONMENT, "READ_WRITE");
            environment.put(
                    AUTHORITY_DIGEST_ENVIRONMENT,
                    "sha256:" + "a".repeat(64));
            environment.put(
                    POLICY_DIGEST_ENVIRONMENT,
                    "sha256:" + "b".repeat(64));
            environment.put(
                    CONFIGURATION_DIGEST_ENVIRONMENT,
                    "sha256:" + "c".repeat(64));
            environment.put(
                    AUTHORITY_CONTRACT_ENVIRONMENT,
                    "buildopt-local-cache-authority/v1");
            environment.put(
                    GATEWAY_URL_ENVIRONMENT,
                    "http://127.0.0.1:" + server.getAddress().getPort());
            environment.put(GATEWAY_USERNAME_ENVIRONMENT, "buildopt");
            environment.put(GATEWAY_PASSWORD_ENVIRONMENT, gatewayPassword);
            environment.put(
                    GATEWAY_GENERATION_ENVIRONMENT,
                    "11111111-1111-4111-8111-111111111111");
            String[] arguments =
                    arguments(
                            initScript,
                            "--build-cache",
                            "-PbuildoptTierOneRegisterTransform=false",
                            "clean",
                            "compileJava");

            BuildResult first =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment,
                            arguments);
            requireOutcome(first, ":compileJava", TaskOutcome.SUCCESS, scenario);
            if (puts.get() == 0 || objects.isEmpty()) {
                throw new IllegalStateException(
                        scenario
                                + " did not publish through HttpBuildCache\n"
                                + first.getOutput());
            }
            BuildResult second =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment,
                            arguments);
            requireOutcome(
                    second,
                    ":compileJava",
                    TaskOutcome.FROM_CACHE,
                    scenario);
            requireConfigurationCacheReuse(second, scenario);
            if (gets.get() == 0) {
                throw new IllegalStateException(
                        scenario + " did not read through HttpBuildCache");
            }
        } finally {
            server.stop(0);
        }
    }

    private static void runL1L2Lifecycle(
            Path project,
            Path stateRoot,
            Path initScript,
            File gradleHome,
            String dsl)
            throws IOException {
        String scenario = dsl + "-l1-l2-lifecycle";
        String gatewayPassword =
                Base64.getUrlEncoder()
                        .withoutPadding()
                        .encodeToString(new byte[32]);
        String basicAuthorization =
                "Basic "
                        + Base64.getEncoder()
                                .encodeToString(
                                        ("buildopt:" + gatewayPassword)
                                                .getBytes(StandardCharsets.UTF_8));
        Path generation50 = managedDirectory(stateRoot, 50);
        Path generation51 = managedDirectory(stateRoot, 51);
        Path abortedWriterGeneration = managedDirectory(stateRoot, 52);
        Path abortReaderGeneration = managedDirectory(stateRoot, 53);
        Files.createDirectories(generation50);
        Files.createDirectories(generation51);
        Files.createDirectories(abortReaderGeneration);
        String[] buildArguments =
                arguments(
                        initScript,
                        "--build-cache",
                        "-PbuildoptTierOneRegisterTransform=false",
                        "clean",
                        "compileJava");
        Path source =
                project.resolve("src/main/java/example/TierOne.java");
        String original = Files.readString(source, StandardCharsets.UTF_8);

        try (LifecycleCacheServer server =
                new LifecycleCacheServer(basicAuthorization)) {
            Map<String, String> seedWriter =
                    lifecycleEnvironment(
                            DISABLED_L2_WRITER_MODE,
                            "",
                            50,
                            "READ_WRITE",
                            server.endpoint(),
                            gatewayPassword,
                            "a",
                            "b",
                            "c",
                            "44444444-4444-4444-8444-444444444444");
            server.resetCounters();
            BuildResult seeded =
                    run(
                            project,
                            gradleHome,
                            scenario + "-seed-writer",
                            seedWriter,
                            buildArguments);
            requireOutcome(
                    seeded,
                    ":compileJava",
                    TaskOutcome.SUCCESS,
                    scenario + "-seed-writer");
            requireNoCredential(
                    seeded,
                    gatewayPassword,
                    scenario + "-seed-writer");
            if (server.puts() < 1 || server.pendingCount() < 1) {
                throw new IllegalStateException(
                        scenario
                                + " trusted writer produced no pending L2 object\n"
                                + seeded.getOutput());
            }
            server.commitPending();
            if (server.stableCount() < 1 || server.pendingCount() != 0) {
                throw new IllegalStateException(
                        scenario + " did not commit the seeded L2 objects");
            }

            Map<String, String> generation50Reader =
                    lifecycleEnvironment(
                            READ_WRITE_MODE,
                            generation50.toString(),
                            50,
                            "READ_ONLY",
                            server.endpoint(),
                            gatewayPassword,
                            "a",
                            "b",
                            "c",
                            "44444444-4444-4444-8444-444444444444");
            server.resetCounters();
            BuildResult remoteHit =
                    run(
                            project,
                            gradleHome,
                            scenario + "-reader",
                            generation50Reader,
                            buildArguments);
            requireOutcome(
                    remoteHit,
                    ":compileJava",
                    TaskOutcome.FROM_CACHE,
                    scenario + "-remote-hit");
            requireNoCredential(
                    remoteHit,
                    gatewayPassword,
                    scenario + "-remote-hit");
            requireCacheContent(generation50, scenario + "-remote-hit");
            if (server.hits() < 1) {
                throw new IllegalStateException(
                        scenario
                                + " did not restore the committed L2 object\n"
                                + remoteHit.getOutput());
            }

            server.setReadsEnabled(false);
            server.resetCounters();
            BuildResult localHit =
                    run(
                            project,
                            gradleHome,
                            scenario + "-reader",
                            generation50Reader,
                            buildArguments);
            requireOutcome(
                    localHit,
                    ":compileJava",
                    TaskOutcome.FROM_CACHE,
                    scenario + "-local-hit");
            requireConfigurationCacheReuse(
                    localHit,
                    scenario + "-local-hit");
            requireNoCredential(
                    localHit,
                    gatewayPassword,
                    scenario + "-local-hit");

            server.setReadsEnabled(true);
            server.revokeStable();
            Map<String, String> generation51Reader =
                    lifecycleEnvironment(
                            READ_WRITE_MODE,
                            generation51.toString(),
                            51,
                            "READ_ONLY",
                            server.endpoint(),
                            gatewayPassword,
                            "d",
                            "e",
                            "f",
                            "55555555-5555-4555-8555-555555555555");
            server.resetCounters();
            BuildResult revokedMiss =
                    run(
                            project,
                            gradleHome,
                            scenario + "-reader",
                            generation51Reader,
                            buildArguments);
            requireOutcome(
                    revokedMiss,
                    ":compileJava",
                    TaskOutcome.SUCCESS,
                    scenario + "-revoked-miss");
            if (revokedMiss.getOutput().contains(
                    "Configuration cache entry reused.")) {
                throw new IllegalStateException(
                        scenario
                                + " reused Configuration Cache across authenticated revocation\n"
                                + revokedMiss.getOutput());
            }
            if (server.misses() < 1) {
                throw new IllegalStateException(
                        scenario + " did not observe the revoked L2 miss");
            }
            requireNoCredential(
                    revokedMiss,
                    gatewayPassword,
                    scenario + "-revoked-miss");
            requireCacheContent(generation51, scenario + "-revoked-miss");

            server.setReadsEnabled(false);
            server.resetCounters();
            BuildResult rotatedHit =
                    run(
                            project,
                            gradleHome,
                            scenario + "-reader",
                            generation51Reader,
                            buildArguments);
            requireOutcome(
                    rotatedHit,
                    ":compileJava",
                    TaskOutcome.FROM_CACHE,
                    scenario + "-rotated-hit");
            requireConfigurationCacheReuse(
                    rotatedHit,
                    scenario + "-rotated-hit");
            requireNoCredential(
                    rotatedHit,
                    gatewayPassword,
                    scenario + "-rotated-hit");

            Files.writeString(
                    source,
                    original + "\n// a0-g02-aborted-writer\n",
                    StandardCharsets.UTF_8);
            server.setReadsEnabled(true);
            Map<String, String> abortedWriter =
                    lifecycleEnvironment(
                            DISABLED_L2_WRITER_MODE,
                            "",
                            52,
                            "READ_WRITE",
                            server.endpoint(),
                            gatewayPassword,
                            "7",
                            "8",
                            "9",
                            "66666666-6666-4666-8666-666666666666");
            server.resetCounters();
            BuildResult pending =
                    run(
                            project,
                            gradleHome,
                            scenario + "-aborted-writer",
                            abortedWriter,
                            buildArguments);
            requireOutcome(
                    pending,
                    ":compileJava",
                    TaskOutcome.SUCCESS,
                    scenario + "-aborted-writer");
            requireNoCredential(
                    pending,
                    gatewayPassword,
                    scenario + "-aborted-writer");
            if (server.puts() < 1
                    || server.pendingCount() < 1
                    || server.stableCount() != 0
                    || Files.exists(abortedWriterGeneration)) {
                throw new IllegalStateException(
                        scenario
                                + " aborted writer did not remain pending-only\n"
                                + pending.getOutput());
            }
            server.abortPending();

            Map<String, String> abortReader =
                    lifecycleEnvironment(
                            READ_WRITE_MODE,
                            abortReaderGeneration.toString(),
                            53,
                            "READ_ONLY",
                            server.endpoint(),
                            gatewayPassword,
                            "1",
                            "2",
                            "3",
                            "77777777-7777-4777-8777-777777777777");
            server.resetCounters();
            BuildResult abortedMiss =
                    run(
                            project,
                            gradleHome,
                            scenario + "-abort-reader",
                            abortReader,
                            buildArguments);
            requireOutcome(
                    abortedMiss,
                    ":compileJava",
                    TaskOutcome.SUCCESS,
                    scenario + "-abort-reader");
            if (server.hits() != 0
                    || server.misses() < 1
                    || server.stableCount() != 0
                    || server.pendingCount() != 0) {
                throw new IllegalStateException(
                        scenario
                                + " exposed an aborted local or remote hit\n"
                                + abortedMiss.getOutput());
            }
            requireNoCredential(
                    abortedMiss,
                    gatewayPassword,
                    scenario + "-abort-reader");
        }
    }

    private static Map<String, String> lifecycleEnvironment(
            String localMode,
            String localDirectory,
            long generation,
            String sharedMode,
            String endpoint,
            String gatewayPassword,
            String authorityDigit,
            String policyDigit,
            String configurationDigit,
            String gatewayGeneration) {
        Map<String, String> environment =
                managedEnvironment(
                        localMode,
                        localDirectory,
                        generation);
        environment.put(SHARED_MODE_ENVIRONMENT, sharedMode);
        environment.put(
                AUTHORITY_DIGEST_ENVIRONMENT,
                "sha256:" + authorityDigit.repeat(64));
        environment.put(
                POLICY_DIGEST_ENVIRONMENT,
                "sha256:" + policyDigit.repeat(64));
        environment.put(
                CONFIGURATION_DIGEST_ENVIRONMENT,
                "sha256:" + configurationDigit.repeat(64));
        environment.put(
                AUTHORITY_CONTRACT_ENVIRONMENT,
                "buildopt-local-cache-authority/v1");
        environment.put(GATEWAY_URL_ENVIRONMENT, endpoint);
        environment.put(GATEWAY_USERNAME_ENVIRONMENT, "buildopt");
        environment.put(GATEWAY_PASSWORD_ENVIRONMENT, gatewayPassword);
        environment.put(
                GATEWAY_GENERATION_ENVIRONMENT,
                gatewayGeneration);
        return environment;
    }

    private static final class LifecycleCacheServer
            implements AutoCloseable {
        private final HttpServer server;
        private final String authorization;
        private final Map<String, byte[]> stable =
                new ConcurrentHashMap<>();
        private final Map<String, byte[]> pending =
                new ConcurrentHashMap<>();
        private final AtomicInteger hits = new AtomicInteger();
        private final AtomicInteger misses = new AtomicInteger();
        private final AtomicInteger puts = new AtomicInteger();
        private volatile boolean readsEnabled = true;

        LifecycleCacheServer(String authorization) throws IOException {
            this.authorization = authorization;
            server =
                    HttpServer.create(
                            new InetSocketAddress("127.0.0.1", 0),
                            0);
            server.createContext("/cache/", this::serve);
            server.start();
        }

        String endpoint() {
            return "http://127.0.0.1:" + server.getAddress().getPort();
        }

        void setReadsEnabled(boolean enabled) {
            readsEnabled = enabled;
        }

        void resetCounters() {
            hits.set(0);
            misses.set(0);
            puts.set(0);
        }

        int hits() {
            return hits.get();
        }

        int misses() {
            return misses.get();
        }

        int puts() {
            return puts.get();
        }

        int stableCount() {
            return stable.size();
        }

        int pendingCount() {
            return pending.size();
        }

        void commitPending() {
            stable.putAll(pending);
            pending.clear();
        }

        void abortPending() {
            pending.clear();
        }

        void revokeStable() {
            stable.clear();
            pending.clear();
        }

        private void serve(HttpExchange exchange) throws IOException {
            try (exchange) {
                if (!authorization.equals(
                        exchange.getRequestHeaders()
                                .getFirst("Authorization"))) {
                    exchange.sendResponseHeaders(401, -1);
                    return;
                }
                String key = exchange.getRequestURI().getPath();
                if (!key.startsWith("/cache/")
                        || key.length() <= "/cache/".length()) {
                    exchange.sendResponseHeaders(404, -1);
                    return;
                }
                if (exchange.getRequestMethod().equals("GET")) {
                    serveGet(exchange, key);
                    return;
                }
                if (exchange.getRequestMethod().equals("PUT")) {
                    servePut(exchange, key);
                    return;
                }
                exchange.getResponseHeaders().set("Allow", "GET, PUT");
                exchange.sendResponseHeaders(405, -1);
            }
        }

        private void serveGet(HttpExchange exchange, String key)
                throws IOException {
            byte[] payload = readsEnabled ? stable.get(key) : null;
            if (payload == null) {
                misses.incrementAndGet();
                exchange.sendResponseHeaders(404, -1);
                return;
            }
            hits.incrementAndGet();
            exchange.getResponseHeaders()
                    .set("Content-Type", "application/octet-stream");
            exchange.sendResponseHeaders(200, payload.length);
            exchange.getResponseBody().write(payload);
        }

        private void servePut(HttpExchange exchange, String key)
                throws IOException {
            puts.incrementAndGet();
            byte[] payload = exchange.getRequestBody().readAllBytes();
            byte[] existing = pending.putIfAbsent(key, payload);
            if (existing != null
                    && !java.util.Arrays.equals(existing, payload)) {
                exchange.sendResponseHeaders(409, -1);
                return;
            }
            exchange.sendResponseHeaders(
                    existing == null ? 201 : 200,
                    -1);
        }

        @Override
        public void close() {
            server.stop(0);
        }
    }

    private static void serveCache(
            HttpExchange exchange,
            String authorization,
            Map<String, byte[]> objects,
            AtomicInteger gets,
            AtomicInteger puts)
            throws IOException {
        try (exchange) {
            if (!authorization.equals(
                    exchange.getRequestHeaders().getFirst("Authorization"))) {
                exchange.sendResponseHeaders(401, -1);
                return;
            }
            String key = exchange.getRequestURI().getPath();
            if (!key.startsWith("/cache/") || key.length() <= "/cache/".length()) {
                exchange.sendResponseHeaders(404, -1);
                return;
            }
            if (exchange.getRequestMethod().equals("GET")) {
                gets.incrementAndGet();
                byte[] payload = objects.get(key);
                if (payload == null) {
                    exchange.sendResponseHeaders(404, -1);
                    return;
                }
                exchange.getResponseHeaders()
                        .set("Content-Type", "application/octet-stream");
                exchange.sendResponseHeaders(200, payload.length);
                exchange.getResponseBody().write(payload);
                return;
            }
            if (exchange.getRequestMethod().equals("PUT")) {
                puts.incrementAndGet();
                objects.put(key, exchange.getRequestBody().readAllBytes());
                exchange.sendResponseHeaders(201, -1);
                return;
            }
            exchange.sendResponseHeaders(405, -1);
        }
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

    private static void requireNoCredential(
            BuildResult result,
            String gatewayPassword,
            String scenario) {
        if (result.getOutput().contains(gatewayPassword)) {
            throw new IllegalStateException(
                    scenario + " exposed the local gateway credential");
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
