package dev.buildopt.gradle;

import java.io.File;
import java.io.IOException;
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
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** Proves the explicit POC cache adapter for unmodified standard Jar tasks. */
public final class StandardJarCacheTestKit {
    private StandardJarCacheTestKit() {}

    /**
     * Runs one miss and one replay while proving that custom Jar and Copy tasks
     * remain outside the added allowlist.
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

        Path root = Files.createTempDirectory("buildopt-standard-jar-cache-");
        try {
            Path project = root.resolve("project");
            Files.createDirectories(project.resolve("payload"));
            Files.writeString(
                    project.resolve("settings.gradle"),
                    "rootProject.name = 'standard-jar-cache-fixture'\n",
                    StandardCharsets.UTF_8);
            Files.writeString(
                    project.resolve("payload/value.txt"),
                    "stable-payload\n",
                    StandardCharsets.UTF_8);
            Files.writeString(
                    project.resolve("build.gradle"),
                    "plugins { id 'java' }\n"
                            + "tasks.named('jar') {\n"
                            + "    archiveFileName = 'standard.jar'\n"
                            + "    from('payload')\n"
                            + "}\n"
                            + "tasks.register('customJar', Jar) {\n"
                            + "    archiveFileName = 'custom.jar'\n"
                            + "    from('payload')\n"
                            + "    doLast { file('build/custom-marker').text = 'custom' }\n"
                            + "}\n"
                            + "tasks.register('copyPayload', Copy) {\n"
                            + "    from('payload')\n"
                            + "    into(layout.buildDirectory.dir('copied'))\n"
                            + "}\n",
                    StandardCharsets.UTF_8);
            Path initScript = root.resolve("buildopt.init.gradle");
            Files.writeString(
                    initScript,
                    "initscript { dependencies { classpath files('"
                            + groovyQuote(pluginJar)
                            + "') } }\n"
                            + "if (gradle.getParent() == null && "
                            + "System.getenv('BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS') == '1') {\n"
                            + "    gradle.projectsLoaded {\n"
                            + "        gradle.rootProject.pluginManager.apply("
                            + "dev.buildopt.gradle.BuildOptStandardJarCachePlugin)\n"
                            + "    }\n"
                            + "}\n",
                    StandardCharsets.UTF_8);

            BuildResult first = run(project, root, initScript, gradleHome);
            requireOutcome(first, ":jar", TaskOutcome.SUCCESS);
            requireOutcome(first, ":customJar", TaskOutcome.SUCCESS);
            requireOutcome(first, ":copyPayload", TaskOutcome.SUCCESS);
            Path standardJar = project.resolve("build/libs/standard.jar");
            String firstDigest = sha256(standardJar);

            BuildResult second = run(project, root, initScript, gradleHome);
            requireOutcome(second, ":jar", TaskOutcome.FROM_CACHE);
            requireOutcome(second, ":customJar", TaskOutcome.SUCCESS);
            requireOutcome(second, ":copyPayload", TaskOutcome.SUCCESS);
            String secondDigest = sha256(standardJar);
            if (!firstDigest.equals(secondDigest)) {
                throw new IllegalStateException("standard Jar replay changed output bytes");
            }
        } finally {
            deleteTree(root);
        }
        System.out.println(
                "POC standard Jar producer cache TestKit OK: exact Jar replay; custom Jar and Copy denied");
    }

    private static BuildResult run(
            Path project,
            Path root,
            Path initScript,
            File gradleHome) {
        Map<String, String> environment = new HashMap<>(System.getenv());
        environment.keySet().removeIf(key -> key.startsWith("BUILDOPT_"));
        environment.put("GRADLE_USER_HOME", root.resolve("gradle-user-home").toString());
        environment.put("BUILDOPT_CACHE_STANDARD_JAR_PRODUCERS", "1");
        return GradleRunner.create()
                .withProjectDir(project.toFile())
                .withGradleInstallation(gradleHome)
                .withTestKitDir(root.resolve("test-kit").toFile())
                .withEnvironment(environment)
                .withArguments(
                        "--init-script",
                        initScript.toString(),
                        "--build-cache",
                        "--stacktrace",
                        "--console=plain",
                        "--info",
                        "clean",
                        "jar",
                        "customJar",
                        "copyPayload")
                .build();
    }

    private static void requireOutcome(
            BuildResult result, String taskPath, TaskOutcome expected) {
        if (result.task(taskPath) == null
                || result.task(taskPath).getOutcome() != expected) {
            throw new IllegalStateException(
                    taskPath + " did not produce " + expected + "\n" + result.getOutput());
        }
    }

    private static String sha256(Path path) throws IOException {
        try {
            return HexFormat.of().formatHex(
                    MessageDigest.getInstance("SHA-256").digest(Files.readAllBytes(path)));
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
                    public FileVisitResult visitFile(
                            Path file, BasicFileAttributes attributes) throws IOException {
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
