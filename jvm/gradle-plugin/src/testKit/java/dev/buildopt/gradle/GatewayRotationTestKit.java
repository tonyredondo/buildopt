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

/** Golden-lane conformance for gateway restart and complete identity rotation. */
public final class GatewayRotationTestKit {
    private static final String MODE_ENVIRONMENT = "BUILDOPT_MANAGED_L1_MODE";
    private static final String GENERATION_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_SECURITY_GENERATION";
    private static final String RETENTION_ENVIRONMENT =
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
    private static final String GATEWAY_URL_ENVIRONMENT = "BUILDOPT_GATEWAY_URL";
    private static final String GATEWAY_USERNAME_ENVIRONMENT =
            "BUILDOPT_GATEWAY_USERNAME";
    private static final String GATEWAY_PASSWORD_ENVIRONMENT =
            "BUILDOPT_GATEWAY_PASSWORD";
    private static final String GATEWAY_GENERATION_ENVIRONMENT =
            "BUILDOPT_GATEWAY_CONNECTION_GENERATION";
    private static final String GATEWAY_USERNAME = "buildopt";
    private static final String UPSTREAM_CREDENTIAL_MARKER =
            "upstream-secret-must-never-reach-gradle";

    private GatewayRotationTestKit() {}

    /**
     * Runs stable restart and complete rotation for Kotlin and Groovy.
     *
     * @param arguments fixture root, Gradle home, plugin JAR, and Java major
     * @throws IOException when fixture or gateway state cannot be managed
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

        Path testRoot = Files.createTempDirectory("buildopt-gateway-rotation-");
        try {
            copyTree(fixtureRoot, testRoot.resolve("tier1"));
            Path initScript = writeInitScript(testRoot, pluginJar);
            for (String dsl : new String[] {"kotlin", "groovy"}) {
                runScenario(
                        testRoot.resolve("tier1").resolve(dsl),
                        initScript,
                        gradleHome,
                        dsl);
            }
        } finally {
            deleteTree(testRoot);
        }
        System.out.printf(
                "A0-G03 TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy stable restart and complete gateway rotation%n",
                gradleHome.getName(),
                expectedJava);
    }

    private static void runScenario(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl)
            throws IOException {
        String scenario = dsl + "-gateway-rotation";
        String[] arguments = {
            "clean",
            "compileJava",
            "--build-cache",
            "-PbuildoptTierOneRegisterTransform=false",
            "--init-script",
            initScript.toString(),
            "--configuration-cache",
            "--stacktrace",
            "--warning-mode=fail",
            "--console=plain",
            "--info"
        };
        try (RotatingGateway gateway = new RotatingGateway()) {
            gateway.resetCounters();
            BuildResult first =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment(gateway),
                            arguments);
            requireOutcome(first, TaskOutcome.SUCCESS, scenario + "-cold");
            requireSecretsAbsent(
                    first,
                    scenario + "-cold",
                    gateway.password(),
                    UPSTREAM_CREDENTIAL_MARKER);
            if (gateway.puts() < 1) {
                throw new IllegalStateException(
                        scenario
                                + " cold build did not publish through the gateway\n"
                                + first.getOutput());
            }

            String stableEndpoint = gateway.endpoint();
            String stablePassword = gateway.password();
            String stableGeneration = gateway.generation();
            gateway.restartStable();
            if (!gateway.endpoint().equals(stableEndpoint)
                    || !gateway.password().equals(stablePassword)
                    || !gateway.generation().equals(stableGeneration)) {
                throw new IllegalStateException(
                        scenario + " stable restart changed gateway identity");
            }
            gateway.resetCounters();
            BuildResult restarted =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment(gateway),
                            arguments);
            requireOutcome(
                    restarted,
                    TaskOutcome.FROM_CACHE,
                    scenario + "-stable-restart");
            requireConfigurationCacheReuse(
                    restarted,
                    scenario + "-stable-restart");
            requireSecretsAbsent(
                    restarted,
                    scenario + "-stable-restart",
                    stablePassword,
                    UPSTREAM_CREDENTIAL_MARKER);
            if (gateway.hits() < 1) {
                throw new IllegalStateException(
                        scenario + " stable restart did not serve a cache hit");
            }

            gateway.rotate();
            if (gateway.endpoint().equals(stableEndpoint)
                    || gateway.password().equals(stablePassword)
                    || gateway.generation().equals(stableGeneration)) {
                throw new IllegalStateException(
                        scenario + " rotation retained a gateway identity field");
            }
            gateway.resetCounters();
            BuildResult rotated =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment(gateway),
                            arguments);
            requireOutcome(
                    rotated,
                    TaskOutcome.FROM_CACHE,
                    scenario + "-rotated");
            if (rotated.getOutput().contains(
                    "Configuration cache entry reused.")) {
                throw new IllegalStateException(
                        scenario
                                + " reused Configuration Cache across gateway rotation\n"
                                + rotated.getOutput());
            }
            requireSecretsAbsent(
                    rotated,
                    scenario + "-rotated",
                    stablePassword,
                    gateway.password(),
                    UPSTREAM_CREDENTIAL_MARKER);
            if (gateway.hits() < 1) {
                throw new IllegalStateException(
                        scenario + " rotated gateway did not serve a cache hit");
            }

            gateway.resetCounters();
            BuildResult replay =
                    run(
                            project,
                            gradleHome,
                            scenario,
                            environment(gateway),
                            arguments);
            requireOutcome(
                    replay,
                    TaskOutcome.FROM_CACHE,
                    scenario + "-rotated-replay");
            requireConfigurationCacheReuse(
                    replay,
                    scenario + "-rotated-replay");
            requireSecretsAbsent(
                    replay,
                    scenario + "-rotated-replay",
                    stablePassword,
                    gateway.password(),
                    UPSTREAM_CREDENTIAL_MARKER);
            if (gateway.hits() < 1) {
                throw new IllegalStateException(
                        scenario + " rotated replay did not serve a cache hit");
            }
        }
    }

    private static Map<String, String> environment(
            RotatingGateway gateway) {
        Map<String, String> environment = new HashMap<>();
        environment.put(MODE_ENVIRONMENT, "DISABLED_L2_WRITER");
        environment.put(GENERATION_ENVIRONMENT, "70");
        environment.put(RETENTION_ENVIRONMENT, "7");
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
        environment.put(GATEWAY_URL_ENVIRONMENT, gateway.endpoint());
        environment.put(GATEWAY_USERNAME_ENVIRONMENT, GATEWAY_USERNAME);
        environment.put(
                GATEWAY_PASSWORD_ENVIRONMENT,
                gateway.password());
        environment.put(
                GATEWAY_GENERATION_ENVIRONMENT,
                gateway.generation());
        return environment;
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

    private static void requireOutcome(
            BuildResult result,
            TaskOutcome expected,
            String scenario) {
        if (result.task(":compileJava") == null) {
            throw new IllegalStateException(
                    scenario
                            + " task was absent: :compileJava\n"
                            + result.getOutput());
        }
        TaskOutcome actual = result.task(":compileJava").getOutcome();
        if (actual != expected) {
            throw new IllegalStateException(
                    scenario
                            + " task :compileJava was "
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
                    scenario
                            + " did not reuse Configuration Cache\n"
                            + result.getOutput());
        }
    }

    private static void requireSecretsAbsent(
            BuildResult result,
            String scenario,
            String... secrets) {
        for (String secret : secrets) {
            if (!secret.isEmpty() && result.getOutput().contains(secret)) {
                throw new IllegalStateException(
                        scenario + " exposed a gateway or upstream credential");
            }
        }
    }

    private static Path writeInitScript(Path testRoot, Path pluginJar)
            throws IOException {
        Path initScript = testRoot.resolve("buildopt-gateway-rotation.init.gradle");
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

    private static final class RotatingGateway implements AutoCloseable {
        private final Map<String, byte[]> objects =
                new ConcurrentHashMap<>();
        private final AtomicInteger hits = new AtomicInteger();
        private final AtomicInteger puts = new AtomicInteger();
        private HttpServer server;
        private String password;
        private String generation;

        RotatingGateway() throws IOException {
            password = encodedSecret((byte) 0x21);
            generation = "88888888-8888-4888-8888-888888888888";
            start(new InetSocketAddress("127.0.0.1", 0));
        }

        String endpoint() {
            return "http://127.0.0.1:" + server.getAddress().getPort();
        }

        String password() {
            return password;
        }

        String generation() {
            return generation;
        }

        int hits() {
            return hits.get();
        }

        int puts() {
            return puts.get();
        }

        void resetCounters() {
            hits.set(0);
            puts.set(0);
        }

        void restartStable() throws IOException {
            InetSocketAddress address = server.getAddress();
            server.stop(0);
            start(address);
        }

        void rotate() throws IOException {
            server.stop(0);
            password = encodedSecret((byte) 0x32);
            generation = "99999999-9999-4999-8999-999999999999";
            start(new InetSocketAddress("127.0.0.1", 0));
        }

        private void start(InetSocketAddress address) throws IOException {
            server = HttpServer.create(address, 0);
            server.createContext("/cache/", this::serve);
            server.start();
        }

        private void serve(HttpExchange exchange) throws IOException {
            try (exchange) {
                String expectedAuthorization =
                        "Basic "
                                + Base64.getEncoder()
                                        .encodeToString(
                                                (GATEWAY_USERNAME
                                                                + ":"
                                                                + password)
                                                        .getBytes(
                                                                StandardCharsets.UTF_8));
                if (!expectedAuthorization.equals(
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
                    byte[] payload = objects.get(key);
                    if (payload == null) {
                        exchange.sendResponseHeaders(404, -1);
                        return;
                    }
                    hits.incrementAndGet();
                    exchange.getResponseHeaders()
                            .set("Content-Type", "application/octet-stream");
                    exchange.sendResponseHeaders(200, payload.length);
                    exchange.getResponseBody().write(payload);
                    return;
                }
                if (exchange.getRequestMethod().equals("PUT")) {
                    puts.incrementAndGet();
                    objects.put(
                            key,
                            exchange.getRequestBody().readAllBytes());
                    exchange.sendResponseHeaders(201, -1);
                    return;
                }
                exchange.getResponseHeaders().set("Allow", "GET, PUT");
                exchange.sendResponseHeaders(405, -1);
            }
        }

        @Override
        public void close() {
            if (server != null) {
                server.stop(0);
            }
        }

        private static String encodedSecret(byte value) {
            byte[] secret = new byte[32];
            java.util.Arrays.fill(secret, value);
            return Base64.getUrlEncoder()
                    .withoutPadding()
                    .encodeToString(secret);
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
}
