package dev.buildopt.gradle;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.attribute.BasicFileAttributes;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.FileVisitResult;
import java.util.HashMap;
import java.util.Map;
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** Dependency-free TestKit entry point for the two Tier 1 DSL repositories. */
public final class TierOneTestKit {
    private TierOneTestKit() {}

    /**
     * Runs each fixture twice and proves task/transform output plus Configuration Cache reuse.
     *
     * @param arguments fixture root, Gradle home, plugin JAR, and expected Java major
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

        Path testRoot = Files.createTempDirectory("buildopt-tier-one-testkit-");
        try {
            copyTree(fixtureRoot, testRoot.resolve("tier1"));
            for (String dsl : new String[] {"kotlin", "groovy"}) {
                runFixture(testRoot.resolve("tier1"), dsl, gradleHome, pluginJar);
            }
        } finally {
            deleteTree(testRoot);
        }
        System.out.printf(
                "F0-040 TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy%n",
                gradleHome.getName(),
                expectedJava);
    }

    private static void runFixture(
            Path copiedRoot,
            String dsl,
            File gradleHome,
            Path pluginJar)
            throws IOException {
        Path project = copiedRoot.resolve(dsl);
        Path initScript = copiedRoot.resolve("buildopt.init.gradle");
        Files.writeString(
                initScript,
                "initscript { dependencies { classpath files('"
                        + groovyQuote(pluginJar)
                        + "') } }\n"
                        + "allprojects { apply plugin: dev.buildopt.gradle.BuildOptProjectPlugin }\n",
                StandardCharsets.UTF_8);

        Map<String, String> environment = new HashMap<>(System.getenv());
        environment.keySet().removeIf(key -> key.startsWith("BUILDOPT_"));
        environment.put(
                "GRADLE_USER_HOME",
                project.resolve(".gradle-user-home").toString());
        String[] arguments = {
            "clean",
            "verifyTierOne",
            "--init-script",
            initScript.toString(),
            "--configuration-cache",
            "--build-cache",
            "--stacktrace",
            "--warning-mode=fail",
            "--console=plain",
            "--offline"
        };
        BuildResult first =
                GradleRunner.create()
                        .withProjectDir(project.toFile())
                        .withGradleInstallation(gradleHome)
                        .withEnvironment(environment)
                        .withArguments(arguments)
                        .build();
        if (first.task(":verifyTierOne") == null) {
            throw new IllegalStateException(
                    dsl + " first task was absent:\n" + first.getOutput());
        }
        TaskOutcome firstOutcome = first.task(":verifyTierOne").getOutcome();
        if (firstOutcome != TaskOutcome.SUCCESS
                && firstOutcome != TaskOutcome.FROM_CACHE) {
            throw new IllegalStateException(
                    dsl
                            + " first task outcome was "
                            + firstOutcome
                            + ":\n"
                            + first.getOutput());
        }
        Path output = project.resolve("build/tier-one/verified.txt");
        String expected = "tier-one-input|transformed|verified\n";
        if (!Files.readString(output, StandardCharsets.UTF_8).equals(expected)) {
            throw new IllegalStateException(dsl + " produced unexpected transformed output");
        }

        BuildResult second =
                GradleRunner.create()
                        .withProjectDir(project.toFile())
                        .withGradleInstallation(gradleHome)
                        .withEnvironment(environment)
                        .withArguments(arguments)
                        .build();
        TaskOutcome secondOutcome = second.task(":verifyTierOne").getOutcome();
        if (secondOutcome != TaskOutcome.FROM_CACHE
                && secondOutcome != TaskOutcome.UP_TO_DATE) {
            throw new IllegalStateException(
                    dsl + " second task outcome was " + secondOutcome);
        }
        if (!second.getOutput().contains("Configuration cache entry reused.")) {
            throw new IllegalStateException(dsl + " did not reuse Configuration Cache");
        }
        if (!Files.readString(output, StandardCharsets.UTF_8).equals(expected)) {
            throw new IllegalStateException(dsl + " replay changed the output");
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
