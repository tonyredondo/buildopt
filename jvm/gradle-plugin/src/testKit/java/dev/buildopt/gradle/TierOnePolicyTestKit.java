package dev.buildopt.gradle;

import java.io.File;
import java.io.IOException;
import java.lang.reflect.Method;
import java.net.URLClassLoader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.attribute.BasicFileAttributes;
import java.nio.file.FileVisitResult;
import java.nio.file.SimpleFileVisitor;
import java.util.HashMap;
import java.util.Map;
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** Functional conformance for the opt-in Tier 1 managed-cache policy. */
public final class TierOnePolicyTestKit {
    private static final String GLOBAL_DISABLE_REASON =
            "BuildOpt managed cache is disabled for this invocation";
    private static final String TASK_DEFAULT_DENY_REASON =
            "BuildOpt Tier 1 default-deny allowlist rejected this task";

    private TierOnePolicyTestKit() {}

    /**
     * Runs the positive policy path on every row and negative paths on the
     * golden runtime.
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
        requireArtifactContracts(pluginJar);

        Path testRoot = Files.createTempDirectory("buildopt-tier-one-policy-");
        try {
            copyTree(fixtureRoot, testRoot.resolve("tier1"));
            for (String dsl : new String[] {"kotlin", "groovy"}) {
                Path project = testRoot.resolve("tier1").resolve(dsl);
                Path initScript = writeInitScript(testRoot, pluginJar);
                runSafePolicy(project, initScript, gradleHome, dsl);
                runModifiedBuiltIn(project, initScript, gradleHome, dsl);
                runUnknownTransform(project, initScript, gradleHome, dsl);
            }
        } finally {
            deleteTree(testRoot);
        }
        System.out.printf(
                "A0-002 TestKit OK: Gradle %s / JDK %d / Kotlin+Groovy%n",
                gradleHome.getName(),
                expectedJava);
    }

    private static void requireArtifactContracts(Path pluginJar) throws IOException {
        try (URLClassLoader loader =
                new URLClassLoader(
                        new java.net.URL[] {pluginJar.toUri().toURL()},
                        TierOnePolicyTestKit.class.getClassLoader())) {
            Class<?> policy = loader.loadClass("dev.buildopt.gradle.BuildOptTierOnePolicy");
            Method matcher =
                    policy.getDeclaredMethod("isAllowlistedErrorProneArtifactName", String.class);
            matcher.setAccessible(true);
            for (String artifact : new String[] {
                "gradle-errorprone-plugin-4.2.0.jar",
                "gradle-errorprone-plugin-4.3.0.jar"
            }) {
                if (!(boolean) matcher.invoke(null, artifact)) {
                    throw new IllegalStateException(
                            "allowlisted Error Prone artifact was rejected: " + artifact);
                }
            }
            for (String artifact : new String[] {
                "gradle-errorprone-plugin-4.1.0.jar",
                "gradle-errorprone-plugin-4.4.0.jar",
                "renamed-gradle-errorprone-plugin-4.2.0.jar"
            }) {
                if ((boolean) matcher.invoke(null, artifact)) {
                    throw new IllegalStateException(
                            "unproven Error Prone artifact was accepted: " + artifact);
                }
            }
            Method kotlinMatcher =
                    policy.getDeclaredMethod(
                            "isAllowlistedKotlinScriptExtensionsArtifactName", String.class);
            kotlinMatcher.setAccessible(true);
            for (String artifact : new String[] {
                "kotlin-gradle-plugin-1.6.10.jar",
                "kotlin-gradle-plugin-2.2.0-gradle813.jar"
            }) {
                if (!(boolean) kotlinMatcher.invoke(null, artifact)) {
                    throw new IllegalStateException(
                            "allowlisted Kotlin transform artifact was rejected: " + artifact);
                }
            }
            for (String artifact : new String[] {
                "kotlin-gradle-plugin-1.6.0.jar",
                "kotlin-gradle-plugin-1.6.20.jar",
                "renamed-kotlin-gradle-plugin-1.6.10.jar",
                "kotlin-gradle-plugin-2.2.0.jar"
            }) {
                if ((boolean) kotlinMatcher.invoke(null, artifact)) {
                    throw new IllegalStateException(
                            "unproven Kotlin transform artifact was accepted: " + artifact);
                }
            }
        } catch (ReflectiveOperationException failure) {
            throw new IOException("cannot inspect packaged Tier One artifact policy", failure);
        }
    }

    private static Path writeInitScript(Path testRoot, Path pluginJar)
            throws IOException {
        Path initScript = testRoot.resolve("buildopt-tier-one-policy.init.gradle");
        Files.writeString(
                initScript,
                "initscript { dependencies { classpath files('"
                        + groovyQuote(pluginJar)
                        + "') } }\n"
                        + "allprojects {\n"
                        + "    apply plugin: dev.buildopt.gradle.BuildOptProjectPlugin\n"
                        + "    apply plugin: dev.buildopt.gradle.BuildOptTierOnePolicyPlugin\n"
                        + "}\n",
                StandardCharsets.UTF_8);
        return initScript;
    }

    private static void runSafePolicy(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl) {
        String scenario = dsl + "-safe";
        boolean executeTest = gradleHome.getName().equals("gradle-9.6.1");
        String[] arguments =
                executeTest
                        ? arguments(
                                initScript,
                                "-PbuildoptTierOneRegisterTransform=false",
                                "clean",
                                "compileJava",
                                "compileTestJava",
                                "test",
                                "unknownCacheable")
                        : arguments(
                                initScript,
                                "-PbuildoptTierOneRegisterTransform=false",
                                "clean",
                                "compileJava",
                                "compileTestJava",
                                "unknownCacheable");
        BuildResult first = run(project, gradleHome, scenario, arguments);
        requireOutcome(first, ":compileJava", TaskOutcome.SUCCESS, scenario);
        requireOutcome(first, ":compileTestJava", TaskOutcome.SUCCESS, scenario);
        if (executeTest) {
            requireOutcome(first, ":test", TaskOutcome.SUCCESS, scenario);
        }
        requireOutcome(first, ":unknownCacheable", TaskOutcome.SUCCESS, scenario);

        BuildResult second = run(project, gradleHome, scenario, arguments);
        requireOutcome(second, ":compileJava", TaskOutcome.FROM_CACHE, scenario);
        requireOutcome(second, ":compileTestJava", TaskOutcome.FROM_CACHE, scenario);
        if (executeTest) {
            requireOutcome(second, ":test", TaskOutcome.SUCCESS, scenario);
        }
        requireOutcome(second, ":unknownCacheable", TaskOutcome.SUCCESS, scenario);
        requireConfigurationCacheReuse(second, scenario);
        requireReason(second, TASK_DEFAULT_DENY_REASON, scenario);
    }

    private static void runModifiedBuiltIn(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl) {
        String scenario = dsl + "-modified-built-in";
        String[] arguments = arguments(
                initScript,
                "-PbuildoptTierOneRegisterTransform=false",
                "-PbuildoptTierOneModifyJavaCompile=true",
                "clean",
                "compileJava");
        requireOutcome(
                run(project, gradleHome, scenario, arguments),
                ":compileJava",
                TaskOutcome.SUCCESS,
                scenario);
        BuildResult second = run(project, gradleHome, scenario, arguments);
        requireOutcome(second, ":compileJava", TaskOutcome.SUCCESS, scenario);
        requireConfigurationCacheReuse(second, scenario);
        requireReason(second, TASK_DEFAULT_DENY_REASON, scenario);
    }

    private static void runUnknownTransform(
            Path project,
            Path initScript,
            File gradleHome,
            String dsl) {
        String scenario = dsl + "-unknown-transform";
        String[] arguments = arguments(
                initScript,
                "-PbuildoptTierOneRegisterTransform=true",
                "clean",
                "compileJava");
        BuildResult first = run(project, gradleHome, scenario, arguments);
        requireOutcome(first, ":compileJava", TaskOutcome.SUCCESS, scenario);
        requireReason(first, "Build cache is disabled", scenario);
        BuildResult second = run(project, gradleHome, scenario, arguments);
        requireOutcome(second, ":compileJava", TaskOutcome.SUCCESS, scenario);
        if (second.getOutput().contains("Configuration cache entry reused.")) {
            throw new IllegalStateException(
                    scenario
                            + " reused Configuration Cache after a fail-closed invocation\n"
                            + second.getOutput());
        }
        requireReason(second, GLOBAL_DISABLE_REASON, scenario);
    }

    private static String[] arguments(
            Path initScript,
            String... scenarioArguments) {
        String[] arguments = new String[scenarioArguments.length + 8];
        System.arraycopy(scenarioArguments, 0, arguments, 0, scenarioArguments.length);
        int index = scenarioArguments.length;
        arguments[index++] = "--init-script";
        arguments[index++] = initScript.toString();
        arguments[index++] = "--configuration-cache";
        arguments[index++] = "--build-cache";
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
            String[] arguments) {
        Map<String, String> environment = new HashMap<>(System.getenv());
        environment.keySet().removeIf(key -> key.startsWith("BUILDOPT_"));
        environment.put(
                "GRADLE_USER_HOME",
                project.resolve(".gradle-user-home-" + scenario).toString());
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
        IOException lastFailure = null;
        for (int attempt = 0; attempt < 20; attempt++) {
            if (!Files.exists(root)) {
                return;
            }
            try {
                deleteTreeOnce(root);
                return;
            } catch (IOException failure) {
                lastFailure = failure;
                try {
                    Thread.sleep(100);
                } catch (InterruptedException interrupted) {
                    Thread.currentThread().interrupt();
                    throw new IOException("interrupted while deleting " + root, interrupted);
                }
            }
        }
        throw lastFailure;
    }

    private static void deleteTreeOnce(Path root) throws IOException {
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
