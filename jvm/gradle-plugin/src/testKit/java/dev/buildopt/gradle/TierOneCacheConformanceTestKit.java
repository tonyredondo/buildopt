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
import java.util.Base64;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** Gradle-side HTTP cache conformance for every declared Tier 1 runtime. */
public final class TierOneCacheConformanceTestKit {
    private static final String LOCAL_MODE_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_MODE";
    private static final String LOCAL_GENERATION_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_SECURITY_GENERATION";
    private static final String LOCAL_RETENTION_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_RETENTION_DAYS";
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
    private static final String GATEWAY_URL_ENVIRONMENT =
            "BUILDOPT_GATEWAY_URL";
    private static final String GATEWAY_USERNAME_ENVIRONMENT =
            "BUILDOPT_GATEWAY_USERNAME";
    private static final String GATEWAY_PASSWORD_ENVIRONMENT =
            "BUILDOPT_GATEWAY_PASSWORD";
    private static final String GATEWAY_GENERATION_ENVIRONMENT =
            "BUILDOPT_GATEWAY_CONNECTION_GENERATION";

    private TierOneCacheConformanceTestKit() {}

    /**
     * Runs the native HttpBuildCache client against the normalized gateway
     * response surface for both Tier 1 fixture DSLs.
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
        File gradleHome =
                Path.of(arguments[1]).toAbsolutePath().normalize().toFile();
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
            throw new IllegalArgumentException(
                    "invalid Gradle home: " + gradleHome);
        }
        if (!Files.isRegularFile(pluginJar)) {
            throw new IllegalArgumentException(
                    "missing plugin JAR: " + pluginJar);
        }

        Path testRoot =
                Files.createTempDirectory("buildopt-tier-one-cache-conformance-");
        try {
            copyTree(fixtureRoot, testRoot.resolve("tier1"));
            Path initScript = writeInitScript(testRoot, pluginJar);
            for (String dsl : new String[] {"kotlin", "groovy"}) {
                runConformance(
                        testRoot.resolve("tier1").resolve(dsl),
                        initScript,
                        gradleHome,
                        dsl);
            }
        } finally {
            deleteTree(testRoot);
        }
        System.out.printf(
                "A0-G01 HttpBuildCache OK: Gradle %s / JDK %d / Kotlin+Groovy%n",
                gradleHome.getName(),
                expectedJava);
    }

    private static Path writeInitScript(Path testRoot, Path pluginJar)
            throws IOException {
        Path initScript =
                testRoot.resolve("buildopt-tier-one-cache.init.gradle");
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

    private static void runConformance(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl)
            throws IOException {
        String scenario = dsl + "-http-conformance";
        String gatewayPassword =
                Base64.getUrlEncoder()
                        .withoutPadding()
                        .encodeToString(new byte[32]);
        String basicAuthorization =
                "Basic "
                        + Base64.getEncoder()
                                .encodeToString(
                                        ("buildopt:" + gatewayPassword)
                                                .getBytes(
                                                        StandardCharsets.UTF_8));
        try (CacheServer cache = new CacheServer(basicAuthorization)) {
            Map<String, String> environment =
                    managedEnvironment(
                            cache.endpoint(),
                            gatewayPassword);
            String[] arguments = arguments(initScript);
            Path source =
                    project.resolve(
                            "src/main/java/example/TierOne.java");
            String original =
                    Files.readString(source, StandardCharsets.UTF_8);

            cache.mode(Mode.NORMAL);
            cache.resetCounters();
            BuildResult store =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment,
                            arguments);
            requireOutcome(
                    store,
                    ":compileJava",
                    TaskOutcome.SUCCESS,
                    scenario + "-store");
            if (cache.misses() < 1
                    || cache.puts() < 1
                    || cache.objectCount() < 1) {
                throw new IllegalStateException(
                        scenario
                                + " did not produce a miss and durable PUT\n"
                                + store.getOutput());
            }
            requireNoCredential(store, gatewayPassword, scenario);

            cache.resetCounters();
            BuildResult hit =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment,
                            arguments);
            requireOutcome(
                    hit,
                    ":compileJava",
                    TaskOutcome.FROM_CACHE,
                    scenario + "-hit");
            requireConfigurationCacheReuse(hit, scenario + "-hit");
            if (cache.hits() < 1) {
                throw new IllegalStateException(
                        scenario
                                + " did not consume a remote hit\n"
                                + hit.getOutput());
            }
            requireNoCredential(hit, gatewayPassword, scenario);

            int retainedObjects = cache.objectCount();
            Files.writeString(
                    source,
                    original + "\n// a0-g01-413\n",
                    StandardCharsets.UTF_8);
            cache.mode(Mode.REJECT_PUT);
            cache.resetCounters();
            BuildResult rejected =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment,
                            arguments);
            requireOutcome(
                    rejected,
                    ":compileJava",
                    TaskOutcome.SUCCESS,
                    scenario + "-413");
            if (cache.payloadTooLarge() < 1
                    || cache.objectCount() != retainedObjects) {
                throw new IllegalStateException(
                        scenario
                                + " did not preserve early 413 semantics\n"
                                + rejected.getOutput());
            }
            requireNoCredential(rejected, gatewayPassword, scenario);

            Files.writeString(source, original, StandardCharsets.UTF_8);
            cache.mode(Mode.NORMAL);
            cache.resetCounters();
            BuildResult retained =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment,
                            arguments);
            requireOutcome(
                    retained,
                    ":compileJava",
                    TaskOutcome.FROM_CACHE,
                    scenario + "-retained");
            if (cache.hits() < 1) {
                throw new IllegalStateException(
                        scenario
                                + " lost a prior object after 413\n"
                                + retained.getOutput());
            }

            Files.writeString(
                    source,
                    original + "\n// a0-g01-normalized-fault\n",
                    StandardCharsets.UTF_8);
            cache.mode(Mode.SAFE_MISS_WRITE_FAILURE);
            cache.resetCounters();
            BuildResult normalizedFailure =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment,
                            arguments);
            requireOutcome(
                    normalizedFailure,
                    ":compileJava",
                    TaskOutcome.SUCCESS,
                    scenario + "-normalized-fault");
            if (cache.safeMisses() < 1
                    || cache.writeFailures() < 1
                    || cache.objectCount() != retainedObjects) {
                throw new IllegalStateException(
                        scenario
                                + " did not retain the baseline after normalized faults\n"
                                + normalizedFailure.getOutput());
            }
            requireNoCredential(
                    normalizedFailure,
                    gatewayPassword,
                    scenario);
        }
    }

    private static Map<String, String> managedEnvironment(
            String endpoint,
            String gatewayPassword) {
        Map<String, String> environment = new HashMap<>();
        environment.put(
                LOCAL_MODE_ENVIRONMENT,
                "DISABLED_L2_WRITER");
        environment.put(LOCAL_GENERATION_ENVIRONMENT, "47");
        environment.put(LOCAL_RETENTION_ENVIRONMENT, "7");
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
        environment.put(GATEWAY_URL_ENVIRONMENT, endpoint);
        environment.put(GATEWAY_USERNAME_ENVIRONMENT, "buildopt");
        environment.put(GATEWAY_PASSWORD_ENVIRONMENT, gatewayPassword);
        environment.put(
                GATEWAY_GENERATION_ENVIRONMENT,
                "33333333-3333-4333-8333-333333333333");
        return environment;
    }

    private static String[] arguments(Path initScript) {
        return new String[] {
            "clean",
            "compileJava",
            "--build-cache",
            "--init-script",
            initScript.toString(),
            "--configuration-cache",
            "--stacktrace",
            "--warning-mode=fail",
            "--console=plain",
            "--info",
            "-PbuildoptTierOneRegisterTransform=false"
        };
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
                .withTestKitDir(
                        project.resolve(".test-kit-" + scenario).toFile())
                .withEnvironment(environment)
                .withArguments(arguments)
                .build();
    }

    private static void requireOutcome(
            BuildResult result,
            String taskPath,
            TaskOutcome expected,
            String scenario) {
        if (result.task(taskPath) == null) {
            throw new IllegalStateException(
                    scenario
                            + " task was absent: "
                            + taskPath
                            + "\n"
                            + result.getOutput());
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
        if (!result.getOutput().contains(
                "Configuration cache entry reused.")) {
            throw new IllegalStateException(
                    scenario
                            + " did not reuse Configuration Cache\n"
                            + result.getOutput());
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

    private static void copyTree(Path source, Path target)
            throws IOException {
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
                        Files.createDirectories(
                                target.resolve(source.relativize(directory)));
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

    private enum Mode {
        NORMAL,
        REJECT_PUT,
        SAFE_MISS_WRITE_FAILURE
    }

    private static final class CacheServer implements AutoCloseable {
        private final HttpServer server;
        private final String authorization;
        private final Map<String, byte[]> objects =
                new ConcurrentHashMap<>();
        private final AtomicInteger hits = new AtomicInteger();
        private final AtomicInteger misses = new AtomicInteger();
        private final AtomicInteger puts = new AtomicInteger();
        private final AtomicInteger payloadTooLarge = new AtomicInteger();
        private final AtomicInteger safeMisses = new AtomicInteger();
        private final AtomicInteger writeFailures = new AtomicInteger();
        private volatile Mode mode = Mode.NORMAL;

        CacheServer(String authorization) throws IOException {
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

        void mode(Mode next) {
            mode = next;
        }

        void resetCounters() {
            hits.set(0);
            misses.set(0);
            puts.set(0);
            payloadTooLarge.set(0);
            safeMisses.set(0);
            writeFailures.set(0);
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

        int payloadTooLarge() {
            return payloadTooLarge.get();
        }

        int safeMisses() {
            return safeMisses.get();
        }

        int writeFailures() {
            return writeFailures.get();
        }

        int objectCount() {
            return objects.size();
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
            if (mode == Mode.SAFE_MISS_WRITE_FAILURE) {
                safeMisses.incrementAndGet();
                exchange.sendResponseHeaders(404, -1);
                return;
            }
            byte[] payload = objects.get(key);
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
            if (mode == Mode.REJECT_PUT) {
                payloadTooLarge.incrementAndGet();
                exchange.sendResponseHeaders(413, -1);
                return;
            }
            if (mode == Mode.SAFE_MISS_WRITE_FAILURE) {
                writeFailures.incrementAndGet();
                exchange.sendResponseHeaders(503, -1);
                return;
            }
            puts.incrementAndGet();
            byte[] payload = exchange.getRequestBody().readAllBytes();
            byte[] existing = objects.putIfAbsent(key, payload);
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
}
