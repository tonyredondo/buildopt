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
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** A0-G08 root, buildSrc, and included-plugin Test cache isolation. */
public final class TestCacheIsolationTestKit {
    private static final String NO_GRANT_REASON =
            "BuildOpt Test task has no TestCacheGrant";
    private static final String CACHE_PASSWORD =
            "a0-g08-observer-password";

    private TestCacheIsolationTestKit() {}

    /**
     * Proves an unguarded control can use the same remote cache while the
     * packaged no-grant policy performs no Test cache requests.
     *
     * @param arguments fixture root, Gradle home, plugin JAR, and Java major
     * @throws IOException when fixture or cache operations fail
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
        if (!gradleHome.getName().equals("gradle-9.6.1")
                || expectedJava != 21
                || Runtime.version().feature() != expectedJava) {
            throw new IllegalStateException(
                    "A0-G08 requires Gradle 9.6.1 and JDK 21");
        }
        if (!gradleHome.toPath().resolve("bin/gradle").toFile().canExecute()
                || !Files.isRegularFile(pluginJar)) {
            throw new IllegalArgumentException(
                    "invalid Gradle home or plugin JAR");
        }

        Path testRoot =
                Files.createTempDirectory("buildopt-test-cache-isolation-");
        try {
            Path project = testRoot.resolve("fixture");
            copyTree(fixtureRoot, project);
            Path initScript = writeInitScript(testRoot, pluginJar);
            String authorization =
                    "Basic "
                            + Base64.getEncoder()
                                    .encodeToString(
                                            ("buildopt:" + CACHE_PASSWORD)
                                                    .getBytes(
                                                            StandardCharsets.UTF_8));
            try (CacheServer cache = new CacheServer(authorization)) {
                runCompositeBoundary(
                        project,
                        initScript,
                        gradleHome,
                        cache);
                runBuildSrcBoundary(
                        project.resolve("buildSrc"),
                        initScript,
                        gradleHome,
                        cache);
            }
        } finally {
            deleteTree(testRoot);
        }
        System.out.println(
                "A0-G08 Test cache isolation OK: root, buildSrc, and included plugin; zero no-grant GET/PUT");
    }

    private static Path writeInitScript(Path testRoot, Path pluginJar)
            throws IOException {
        Path initScript = testRoot.resolve("test-cache-isolation.init.gradle");
        Files.writeString(
                initScript,
                "initscript { dependencies { classpath files('"
                        + groovyQuote(pluginJar)
                        + "') } }\n"
                        + "allprojects {\n"
                        + "    apply plugin: "
                        + "dev.buildopt.gradle.BuildOptTierOnePolicyPlugin\n"
                        + "}\n",
                StandardCharsets.UTF_8);
        return initScript;
    }

    private static void runCompositeBoundary(
            Path project,
            Path initScript,
            File gradleHome,
            CacheServer cache)
            throws IOException {
        String scenario = "root-composite";
        BuildResult stored =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        null,
                        "compositeTest");
        requireOutcome(
                stored,
                ":test",
                TaskOutcome.SUCCESS,
                scenario + "-store-root");
        requireOutcome(
                stored,
                ":included-plugin:test",
                TaskOutcome.SUCCESS,
                scenario + "-store-included");
        if (cache.puts() < 2 || cache.objectCount() < 2) {
            throw new IllegalStateException(
                    scenario + " control did not populate the remote cache");
        }

        deleteTestOutputs(project);
        deleteTestOutputs(project.resolve("included-plugin"));
        cache.resetCounters();
        BuildResult restored =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        null,
                        "compositeTest");
        requireOutcome(
                restored,
                ":test",
                TaskOutcome.FROM_CACHE,
                scenario + "-restore-root");
        requireOutcome(
                restored,
                ":included-plugin:test",
                TaskOutcome.FROM_CACHE,
                scenario + "-restore-included");
        if (cache.hits() < 2) {
            throw new IllegalStateException(
                    scenario + " control did not consume remote entries");
        }

        deleteTestOutputs(project);
        deleteTestOutputs(project.resolve("included-plugin"));
        cache.resetCounters();
        BuildResult guarded =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        initScript,
                        "compositeTest");
        requireGuardedTest(
                guarded,
                ":test",
                scenario + "-guarded-root");
        requireGuardedTest(
                guarded,
                ":included-plugin:test",
                scenario + "-guarded-included");
        requireNoRequests(cache, scenario + "-guarded");

        deleteTestOutputs(project);
        deleteTestOutputs(project.resolve("included-plugin"));
        cache.resetCounters();
        BuildResult reused =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        initScript,
                        "compositeTest");
        requireGuardedTest(
                reused,
                ":test",
                scenario + "-reused-root");
        requireGuardedTest(
                reused,
                ":included-plugin:test",
                scenario + "-reused-included");
        requireConfigurationCacheReuse(reused, scenario);
        requireNoRequests(cache, scenario + "-reused");
    }

    private static void runBuildSrcBoundary(
            Path project,
            Path initScript,
            File gradleHome,
            CacheServer cache)
            throws IOException {
        String scenario = "buildsrc";
        cache.resetCounters();
        BuildResult stored =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        null,
                        "test");
        requireOutcome(
                stored,
                ":test",
                TaskOutcome.SUCCESS,
                scenario + "-store");
        if (cache.puts() < 1) {
            throw new IllegalStateException(
                    scenario + " control did not populate the remote cache");
        }
        deleteTestOutputs(project);
        cache.resetCounters();
        BuildResult restored =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        null,
                        "test");
        requireOutcome(
                restored,
                ":test",
                TaskOutcome.FROM_CACHE,
                scenario + "-restore");
        if (cache.hits() < 1) {
            throw new IllegalStateException(
                    scenario + " control did not consume a remote entry");
        }

        deleteTestOutputs(project);
        cache.resetCounters();
        BuildResult guarded =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        initScript,
                        "test");
        requireGuardedTest(guarded, ":test", scenario + "-guarded");
        requireNoRequests(cache, scenario + "-guarded");

        deleteTestOutputs(project);
        cache.resetCounters();
        BuildResult reused =
                run(
                        project,
                        gradleHome,
                        scenario,
                        cache,
                        initScript,
                        "test");
        requireGuardedTest(reused, ":test", scenario + "-reused");
        requireConfigurationCacheReuse(reused, scenario);
        requireNoRequests(cache, scenario + "-reused");
    }

    private static BuildResult run(
            Path project,
            File gradleHome,
            String scenario,
            CacheServer cache,
            Path initScript,
            String... tasks) {
        List<String> arguments = new ArrayList<>(Arrays.asList(tasks));
        if (initScript != null) {
            arguments.add("--init-script");
            arguments.add(initScript.toString());
        }
        arguments.add(
                "-PbuildoptTestCacheUrl=" + cache.endpoint());
        arguments.add(
                "-PbuildoptTestCachePassword=" + CACHE_PASSWORD);
        arguments.add("--configuration-cache");
        arguments.add("--build-cache");
        arguments.add("--stacktrace");
        arguments.add("--warning-mode=fail");
        arguments.add("--console=plain");
        arguments.add("--info");

        Map<String, String> environment = new HashMap<>(System.getenv());
        environment.keySet().removeIf(key -> key.startsWith("BUILDOPT_"));
        environment.put(
                "GRADLE_USER_HOME",
                project.resolve(".gradle-user-home-" + scenario).toString());
        BuildResult result =
                GradleRunner.create()
                        .withProjectDir(project.toFile())
                        .withGradleInstallation(gradleHome)
                        .withTestKitDir(
                                project.resolve(".test-kit-" + scenario).toFile())
                        .withEnvironment(environment)
                        .withArguments(arguments)
                        .build();
        if (result.getOutput().contains(CACHE_PASSWORD)) {
            throw new IllegalStateException(
                    scenario + " exposed the cache observer credential");
        }
        return result;
    }

    private static void requireGuardedTest(
            BuildResult result,
            String taskPath,
            String scenario) {
        requireOutcome(result, taskPath, TaskOutcome.SUCCESS, scenario);
        if (!result.getOutput().contains(NO_GRANT_REASON)) {
            throw new IllegalStateException(
                    scenario
                            + " did not report the no-grant reason\n"
                            + result.getOutput());
        }
    }

    private static void requireNoRequests(
            CacheServer cache,
            String scenario) {
        if (cache.gets() != 0
                || cache.puts() != 0
                || cache.unauthorized() != 0) {
            throw new IllegalStateException(
                    scenario
                            + " contacted the remote cache: GET="
                            + cache.gets()
                            + " PUT="
                            + cache.puts()
                            + " unauthorized="
                            + cache.unauthorized());
        }
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
        if (!result.getOutput().contains("Configuration cache entry reused.")) {
            throw new IllegalStateException(
                    scenario
                            + " did not reuse Configuration Cache\n"
                            + result.getOutput());
        }
    }

    private static String groovyQuote(Path path) {
        return path.toString().replace("\\", "\\\\").replace("'", "\\'");
    }

    private static void deleteTestOutputs(Path project) throws IOException {
        deleteTree(project.resolve("build/test-results"));
        deleteTree(project.resolve("build/reports/tests"));
        deleteTree(project.resolve("build/tmp/test"));
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

    private static final class CacheServer implements AutoCloseable {
        private final HttpServer server;
        private final String authorization;
        private final Map<String, byte[]> objects =
                new ConcurrentHashMap<>();
        private final AtomicInteger gets = new AtomicInteger();
        private final AtomicInteger hits = new AtomicInteger();
        private final AtomicInteger puts = new AtomicInteger();
        private final AtomicInteger unauthorized = new AtomicInteger();

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
            return "http://127.0.0.1:"
                    + server.getAddress().getPort()
                    + "/cache/";
        }

        void resetCounters() {
            gets.set(0);
            hits.set(0);
            puts.set(0);
            unauthorized.set(0);
        }

        int gets() {
            return gets.get();
        }

        int hits() {
            return hits.get();
        }

        int puts() {
            return puts.get();
        }

        int unauthorized() {
            return unauthorized.get();
        }

        int objectCount() {
            return objects.size();
        }

        private void serve(HttpExchange exchange) throws IOException {
            try (exchange) {
                if (!authorization.equals(
                        exchange.getRequestHeaders()
                                .getFirst("Authorization"))) {
                    unauthorized.incrementAndGet();
                    exchange.sendResponseHeaders(401, -1);
                    return;
                }
                String key = exchange.getRequestURI().getPath();
                if (!key.startsWith("/cache/")
                        || key.length() <= "/cache/".length()) {
                    exchange.sendResponseHeaders(404, -1);
                    return;
                }
                switch (exchange.getRequestMethod()) {
                    case "GET" -> serveGet(exchange, key);
                    case "PUT" -> servePut(exchange, key);
                    default -> {
                        exchange.getResponseHeaders().set("Allow", "GET, PUT");
                        exchange.sendResponseHeaders(405, -1);
                    }
                }
            }
        }

        private void serveGet(HttpExchange exchange, String key)
                throws IOException {
            gets.incrementAndGet();
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
        }

        private void servePut(HttpExchange exchange, String key)
                throws IOException {
            puts.incrementAndGet();
            byte[] payload = exchange.getRequestBody().readAllBytes();
            byte[] existing = objects.putIfAbsent(key, payload);
            if (existing != null && !Arrays.equals(existing, payload)) {
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
