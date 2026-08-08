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
import java.nio.file.attribute.BasicFileAttributes;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HashMap;
import java.util.HexFormat;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** Proves read-only POC Edge hits and native execution after Edge failure. */
public final class POCEdgeCacheTestKit {
    private POCEdgeCacheTestKit() {}

    /**
     * Seeds a loopback HTTP cache, restores through the POC plugin, then
     * returns HTTP 503 and proves exact local execution.
     *
     * @param arguments Gradle home and packaged plugin JAR
     * @throws IOException when the isolated fixture cannot be created or read
     */
    public static void main(String[] arguments) throws IOException {
        if (arguments.length != 2) {
            throw new IllegalArgumentException("expected Gradle home and plugin JAR");
        }
        File gradleHome = Path.of(arguments[0]).toAbsolutePath().normalize().toFile();
        Path pluginJar = Path.of(arguments[1]).toAbsolutePath().normalize();
        if (!gradleHome.toPath().resolve("bin/gradle").toFile().canExecute()
                || !Files.isRegularFile(pluginJar)) {
            throw new IllegalArgumentException("invalid Gradle or plugin input");
        }

        Map<String, byte[]> objects = new ConcurrentHashMap<>();
        AtomicBoolean unavailable = new AtomicBoolean();
        AtomicInteger unavailableRequests = new AtomicInteger();
        HttpServer server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext(
                "/cache/",
                exchange ->
                        serveCache(exchange, objects, unavailable, unavailableRequests));
        server.start();

        Path root = Files.createTempDirectory("buildopt-poc-edge-cache-");
        try {
            Path project = root.resolve("project");
            Files.createDirectories(project);
            Files.writeString(
                    project.resolve("settings.gradle"),
                    "rootProject.name = 'poc-edge-cache-fixture'\n",
                    StandardCharsets.UTF_8);
            Files.writeString(
                    project.resolve("build.gradle"),
                    "plugins { id 'base' }\n"
                            + "@CacheableTask\n"
                            + "abstract class Produce extends DefaultTask {\n"
                            + "  @Input abstract Property<String> getValue()\n"
                            + "  @OutputFile abstract RegularFileProperty getOutput()\n"
                            + "  @TaskAction void produce() { output.get().asFile.text = value.get() + '\\n' }\n"
                            + "}\n"
                            + "tasks.register('produce', Produce) {\n"
                            + "  value = 'stable-output'\n"
                            + "  output = layout.buildDirectory.file('result.txt')\n"
                            + "}\n",
                    StandardCharsets.UTF_8);
            String origin = "http://127.0.0.1:" + server.getAddress().getPort();
            Path seedInit = root.resolve("seed.init.gradle");
            Files.writeString(
                    seedInit,
                    "import org.gradle.caching.http.HttpBuildCache\n"
                            + "settingsEvaluated { settings -> settings.buildCache {\n"
                            + "  local { enabled = false }\n"
                            + "  remote(HttpBuildCache) { url = uri('"
                            + origin
                            + "/cache/'); push = true; allowInsecureProtocol = true }\n"
                            + "} }\n",
                    StandardCharsets.UTF_8);
            Path candidateInit = root.resolve("candidate.init.gradle");
            Files.writeString(
                    candidateInit,
                    "initscript { dependencies { classpath files('"
                            + groovyQuote(pluginJar)
                            + "') } }\n"
                            + "beforeSettings { settings -> settings.pluginManager.apply("
                            + "dev.buildopt.gradle.BuildOptPOCEdgeCachePlugin) }\n",
                    StandardCharsets.UTF_8);

            BuildResult seed = run(project, root.resolve("seed-home"), seedInit, gradleHome, origin);
            requireOutcome(seed, ":produce", TaskOutcome.SUCCESS);
            Path output = project.resolve("build/result.txt");
            String expectedDigest = sha256(output);
            if (objects.isEmpty()) {
                throw new IllegalStateException("native seed did not populate the loopback cache");
            }

            BuildResult hit = run(project, root.resolve("candidate-home"), candidateInit, gradleHome, origin);
            requireOutcome(hit, ":produce", TaskOutcome.FROM_CACHE);
            if (!expectedDigest.equals(sha256(output))) {
                throw new IllegalStateException("POC Edge hit changed output bytes");
            }

            unavailable.set(true);
            BuildResult fallback = run(project, root.resolve("failure-home"), candidateInit, gradleHome, origin);
            requireOutcome(fallback, ":produce", TaskOutcome.SUCCESS);
            if (unavailableRequests.get() == 0 || !expectedDigest.equals(sha256(output))) {
                throw new IllegalStateException("POC Edge failure did not execute the exact native fallback");
            }
        } finally {
            server.stop(0);
            deleteTree(root);
        }
        System.out.println("POC Edge cache TestKit OK: read-only hit and exact HTTP 503 fallback");
    }

    private static BuildResult run(
            Path project, Path home, Path initScript, File gradleHome, String origin) {
        Map<String, String> environment = new HashMap<>(System.getenv());
        environment.keySet().removeIf(key -> key.startsWith("BUILDOPT_"));
        environment.put("GRADLE_USER_HOME", home.toString());
        environment.put("BUILDOPT_POC_EDGE_CACHE_URL", origin);
        return GradleRunner.create()
                .withProjectDir(project.toFile())
                .withGradleInstallation(gradleHome)
                .withTestKitDir(home.resolve("test-kit").toFile())
                .withEnvironment(environment)
                .withArguments(
                        "--init-script",
                        initScript.toString(),
                        "--build-cache",
                        "--stacktrace",
                        "--console=plain",
                        "clean",
                        "produce")
                .build();
    }

    private static void serveCache(
            HttpExchange exchange,
            Map<String, byte[]> objects,
            AtomicBoolean unavailable,
            AtomicInteger unavailableRequests)
            throws IOException {
        try (exchange) {
            if (unavailable.get()) {
                unavailableRequests.incrementAndGet();
                exchange.sendResponseHeaders(503, -1);
                return;
            }
            String key = exchange.getRequestURI().getPath().substring("/cache/".length());
            if (exchange.getRequestMethod().equals("PUT")) {
                byte[] body = exchange.getRequestBody().readAllBytes();
                objects.put(key, body);
                exchange.sendResponseHeaders(201, -1);
                return;
            }
            if (exchange.getRequestMethod().equals("GET")) {
                byte[] body = objects.get(key);
                if (body == null) {
                    exchange.sendResponseHeaders(404, -1);
                    return;
                }
                exchange.sendResponseHeaders(200, body.length);
                exchange.getResponseBody().write(body);
                return;
            }
            exchange.sendResponseHeaders(405, -1);
        }
    }

    private static void requireOutcome(BuildResult result, String taskPath, TaskOutcome expected) {
        if (result.task(taskPath) == null || result.task(taskPath).getOutcome() != expected) {
            throw new IllegalStateException(
                    taskPath + " did not produce " + expected + "\n" + result.getOutput());
        }
    }

    private static String sha256(Path path) throws IOException {
        try {
            return HexFormat.of()
                    .formatHex(MessageDigest.getInstance("SHA-256").digest(Files.readAllBytes(path)));
        } catch (NoSuchAlgorithmException impossible) {
            throw new IllegalStateException("SHA-256 unavailable", impossible);
        }
    }

    private static String groovyQuote(Path path) {
        return path.toString().replace("\\", "\\\\").replace("'", "\\'");
    }

    private static void deleteTree(Path root) throws IOException {
        if (!Files.exists(root)) {
            return;
        }
        Files.walkFileTree(
                root,
                new SimpleFileVisitor<>() {
                    @Override
                    public FileVisitResult visitFile(Path file, BasicFileAttributes attributes)
                            throws IOException {
                        Files.delete(file);
                        return FileVisitResult.CONTINUE;
                    }

                    @Override
                    public FileVisitResult postVisitDirectory(Path directory, IOException failure)
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
