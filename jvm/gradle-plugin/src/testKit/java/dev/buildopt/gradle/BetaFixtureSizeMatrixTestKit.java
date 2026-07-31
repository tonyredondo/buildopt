package dev.buildopt.gradle;

import java.io.File;
import java.io.IOException;
import java.lang.reflect.InvocationTargetException;
import java.net.URL;
import java.net.URLClassLoader;
import java.nio.charset.StandardCharsets;
import java.nio.file.FileVisitResult;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.nio.file.attribute.PosixFilePermissions;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.regex.Pattern;
import java.util.stream.Stream;
import org.gradle.testkit.runner.BuildResult;
import org.gradle.testkit.runner.BuildTask;
import org.gradle.testkit.runner.GradleRunner;
import org.gradle.testkit.runner.TaskOutcome;

/** Executes the benchmark-bound small, medium, and large Gradle fixtures. */
public final class BetaFixtureSizeMatrixTestKit {
    private static final String SCOPE =
            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
    private static final Pattern PROFILE_ID =
            Pattern.compile("TIER1_(SMALL|MEDIUM|LARGE)");

    private BetaFixtureSizeMatrixTestKit() {}

    /**
     * Materializes and executes the exact fixture profiles supplied by the
     * checked machine-readable contract.
     *
     * @param arguments Gradle home, plugin JAR, Java major, report path,
     *     benchmark digest, and encoded profiles
     * @throws Exception when fixture materialization, execution, or inspection
     *     fails
     */
    public static void main(String[] arguments) throws Exception {
        if (arguments.length != 6) {
            throw new IllegalArgumentException(
                    "expected Gradle home, plugin JAR, Java major, report path, "
                            + "benchmark digest, and profiles");
        }
        File gradleHome = Path.of(arguments[0]).toAbsolutePath().normalize().toFile();
        Path pluginJar = Path.of(arguments[1]).toAbsolutePath().normalize();
        int expectedJava = Integer.parseInt(arguments[2]);
        Path report = Path.of(arguments[3]).toAbsolutePath().normalize();
        String benchmarkDigest = arguments[4];
        List<Profile> profiles = parseProfiles(arguments[5]);

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
        if (!benchmarkDigest.matches("sha256:[0-9a-f]{64}")) {
            throw new IllegalArgumentException("invalid benchmark digest");
        }

        Path testRoot = Files.createTempDirectory("buildopt-beta-gradle-fixtures-");
        List<FixtureResult> results = new ArrayList<>();
        try {
            Path initScript = writeInitScript(testRoot, pluginJar);
            for (int index = 0; index < profiles.size(); index++) {
                results.add(
                        runFixture(
                                testRoot,
                                initScript,
                                gradleHome,
                                profiles.get(index),
                                91L + index));
            }
        } finally {
            deleteTree(testRoot);
        }
        writeReport(report, benchmarkDigest, gradleHome.getName(), expectedJava, results);
        System.out.printf(
                "OPS-001/A1 Gradle fixture matrix OK: Gradle %s / JDK %d / small+medium+large%n",
                gradleHome.getName(),
                expectedJava);
    }

    private static List<Profile> parseProfiles(String encoded) {
        List<Profile> profiles = new ArrayList<>();
        for (String rawProfile : encoded.split(";", -1)) {
            String[] fields = rawProfile.split("\\|", -1);
            if (fields.length != 5 || !PROFILE_ID.matcher(fields[0]).matches()) {
                throw new IllegalArgumentException("invalid fixture profile");
            }
            int projects = parsePositive(fields[2], "projects");
            int sourcesPerProject = parsePositive(fields[3], "sources per project");
            int expectedValue = parsePositive(fields[4], "expected value");
            if (!fields[0].equals("TIER1_" + fields[1])
                    || projects > 64
                    || sourcesPerProject > 64
                    || Math.multiplyExact(projects, sourcesPerProject) != expectedValue) {
                throw new IllegalArgumentException("inconsistent fixture profile");
            }
            profiles.add(
                    new Profile(
                            fields[0],
                            fields[1],
                            projects,
                            sourcesPerProject,
                            expectedValue));
        }
        if (profiles.size() != 3
                || !profiles.get(0).sizeClass().equals("SMALL")
                || !profiles.get(1).sizeClass().equals("MEDIUM")
                || !profiles.get(2).sizeClass().equals("LARGE")) {
            throw new IllegalArgumentException("fixture profile order is invalid");
        }
        return profiles;
    }

    private static int parsePositive(String value, String label) {
        try {
            int parsed = Integer.parseInt(value);
            if (parsed < 1 || !Integer.toString(parsed).equals(value)) {
                throw new IllegalArgumentException("invalid " + label);
            }
            return parsed;
        } catch (NumberFormatException failure) {
            throw new IllegalArgumentException("invalid " + label, failure);
        }
    }

    private static FixtureResult runFixture(
            Path testRoot,
            Path initScript,
            File gradleHome,
            Profile profile,
            long generation)
            throws Exception {
        String profileName = profile.sizeClass().toLowerCase(Locale.ROOT);
        Path project = testRoot.resolve(profileName);
        materializeRepository(project, profile);
        Path cache =
                testRoot.resolve("state-" + profileName)
                        .resolve("l1")
                        .resolve("scopes")
                        .resolve(SCOPE)
                        .resolve("generation-" + generation)
                        .resolve("cache")
                        .toAbsolutePath()
                        .normalize();
        Files.createDirectories(cache);
        Map<String, String> environment = new HashMap<>(System.getenv());
        environment.keySet().removeIf(key -> key.startsWith("BUILDOPT_"));
        environment.put(
                "GRADLE_USER_HOME",
                testRoot.resolve("gradle-home-" + profileName).toString());
        environment.put("BUILDOPT_MANAGED_L1_DIRECTORY", cache.toString());
        environment.put("BUILDOPT_MANAGED_L1_MODE", "READ_WRITE");
        environment.put("BUILDOPT_MANAGED_L1_SECURITY_GENERATION", Long.toString(generation));
        environment.put("BUILDOPT_MANAGED_L1_RETENTION_DAYS", "7");

        List<String> criticalPath = criticalPath(profile.projects());
        List<String> buildArguments = new ArrayList<>();
        for (int module = 0; module < profile.projects(); module++) {
            buildArguments.add(modulePath(module) + ":clean");
        }
        buildArguments.add("fixture");
        buildArguments.add("--init-script");
        buildArguments.add(initScript.toString());
        buildArguments.add("--configuration-cache");
        buildArguments.add("--build-cache");
        buildArguments.add("--offline");
        buildArguments.add("--stacktrace");
        buildArguments.add("--warning-mode=fail");
        buildArguments.add("--console=plain");

        GradleRunner runner =
                GradleRunner.create()
                        .withProjectDir(project.toFile())
                        .withGradleInstallation(gradleHome)
                        .withTestKitDir(testRoot.resolve("test-kit-" + profileName).toFile())
                        .withEnvironment(environment)
                        .withArguments(buildArguments);
        BuildResult cold = runner.build();
        requireCriticalPath(cold, criticalPath, TaskOutcome.SUCCESS, profile.id() + " cold");
        int coldValue = inspectKnownOutput(project, profile);

        BuildResult replay = runner.build();
        requireCriticalPath(replay, criticalPath, TaskOutcome.FROM_CACHE, profile.id() + " replay");
        if (!replay.getOutput().contains("Configuration cache entry reused.")) {
            throw new IllegalStateException(profile.id() + " did not reuse Configuration Cache");
        }
        int replayValue = inspectKnownOutput(project, profile);
        if (coldValue != profile.expectedValue() || replayValue != profile.expectedValue()) {
            throw new IllegalStateException(profile.id() + " produced an unexpected value");
        }
        try (Stream<Path> entries = Files.walk(cache)) {
            if (entries.noneMatch(path -> !path.equals(cache))) {
                throw new IllegalStateException(profile.id() + " did not populate managed L1");
            }
        }
        return new FixtureResult(profile, criticalPath, replayValue);
    }

    private static void materializeRepository(Path project, Profile profile) throws IOException {
        Files.createDirectories(project);
        StringBuilder settings = new StringBuilder("rootProject.name = \"")
                .append(profile.id().toLowerCase(Locale.ROOT))
                .append("\"\n");
        for (int module = 0; module < profile.projects(); module++) {
            settings.append("include(\"").append(modulePath(module)).append("\")\n");
        }
        Files.writeString(project.resolve("settings.gradle.kts"), settings, StandardCharsets.UTF_8);

        String lastModule = modulePath(profile.projects() - 1);
        String build =
                "plugins { base }\n"
                        + "subprojects {\n"
                        + "    apply(plugin = \"java\")\n"
                        + "    tasks.withType<JavaCompile>().configureEach {\n"
                        + "        options.release.set(17)\n"
                        + "        options.encoding = \"UTF-8\"\n"
                        + "    }\n"
                        + "    val moduleNumber = name.removePrefix(\"module-\").toInt()\n"
                        + "    if (moduleNumber > 0) {\n"
                        + "        dependencies.add(\"implementation\", "
                        + "dependencies.project(\":module-%03d\".format(moduleNumber - 1)))\n"
                        + "    }\n"
                        + "}\n"
                        + "tasks.register(\"fixture\") { dependsOn(\""
                        + lastModule
                        + ":classes\") }\n";
        Files.writeString(project.resolve("build.gradle.kts"), build, StandardCharsets.UTF_8);

        int value = 0;
        for (int module = 0; module < profile.projects(); module++) {
            Path sourceRoot =
                    project.resolve(moduleName(module))
                            .resolve("src/main/java/fixture")
                            .resolve(packageName(module));
            Files.createDirectories(sourceRoot);
            for (int source = 0; source < profile.sourcesPerProject(); source++) {
                value++;
                String className = className(module, source);
                String predecessor;
                String importLine = "";
                if (module == 0 && source == 0) {
                    predecessor = "0";
                } else if (source == 0) {
                    String previous = className(module - 1, profile.sourcesPerProject() - 1);
                    importLine =
                            "import fixture."
                                    + packageName(module - 1)
                                    + "."
                                    + previous
                                    + ";\n";
                    predecessor = previous + ".value()";
                } else {
                    predecessor = className(module, source - 1) + ".value()";
                }
                String java =
                        "package fixture."
                                + packageName(module)
                                + ";\n"
                                + importLine
                                + "public final class "
                                + className
                                + " {\n"
                                + "    private "
                                + className
                                + "() {}\n"
                                + "    public static int value() { return "
                                + predecessor
                                + " + 1; }\n"
                                + "}\n";
                Files.writeString(
                        sourceRoot.resolve(className + ".java"),
                        java,
                        StandardCharsets.UTF_8);
            }
        }
        if (value != profile.expectedValue()) {
            throw new IllegalStateException("materialized source count drifted");
        }
    }

    private static int inspectKnownOutput(Path project, Profile profile)
            throws IOException, ClassNotFoundException, NoSuchMethodException,
                    IllegalAccessException, InvocationTargetException {
        URL[] classes = new URL[profile.projects()];
        int classFiles = 0;
        for (int module = 0; module < profile.projects(); module++) {
            Path classRoot =
                    project.resolve(moduleName(module)).resolve("build/classes/java/main");
            classes[module] = classRoot.toUri().toURL();
            try (Stream<Path> paths = Files.walk(classRoot)) {
                classFiles += (int) paths.filter(path -> path.toString().endsWith(".class")).count();
            }
        }
        if (classFiles != profile.expectedValue()) {
            throw new IllegalStateException(
                    profile.id()
                            + " compiled "
                            + classFiles
                            + " classes, expected "
                            + profile.expectedValue());
        }
        int finalModule = profile.projects() - 1;
        int finalSource = profile.sourcesPerProject() - 1;
        String finalClass =
                "fixture."
                        + packageName(finalModule)
                        + "."
                        + className(finalModule, finalSource);
        try (URLClassLoader loader =
                new URLClassLoader(classes, ClassLoader.getPlatformClassLoader())) {
            Object value = loader.loadClass(finalClass).getMethod("value").invoke(null);
            return ((Integer) value).intValue();
        }
    }

    private static void requireCriticalPath(
            BuildResult result,
            List<String> criticalPath,
            TaskOutcome expected,
            String scenario) {
        int previousIndex = -1;
        List<BuildTask> executed = result.getTasks();
        for (String taskPath : criticalPath) {
            BuildTask task = result.task(taskPath);
            if (task == null || task.getOutcome() != expected) {
                throw new IllegalStateException(
                        scenario
                                + " task "
                                + taskPath
                                + " was "
                                + (task == null ? "absent" : task.getOutcome())
                                + ", expected "
                                + expected
                                + "\n"
                                + result.getOutput());
            }
            int taskIndex = executed.indexOf(task);
            if (taskIndex <= previousIndex) {
                throw new IllegalStateException(scenario + " critical path executed out of order");
            }
            previousIndex = taskIndex;
        }
    }

    private static Path writeInitScript(Path root, Path pluginJar) throws IOException {
        Path initScript = root.resolve("buildopt-beta-gradle-fixtures.init.gradle");
        Files.writeString(
                initScript,
                "initscript { dependencies { classpath files('"
                        + groovyQuote(pluginJar)
                        + "') } }\n"
                        + "beforeSettings { settings ->\n"
                        + "    settings.pluginManager.apply(dev.buildopt.gradle.BuildOptManagedL1Plugin)\n"
                        + "}\n",
                StandardCharsets.UTF_8);
        return initScript;
    }

    private static List<String> criticalPath(int projects) {
        List<String> paths = new ArrayList<>();
        for (int module = 0; module < projects; module++) {
            paths.add(modulePath(module) + ":compileJava");
        }
        return List.copyOf(paths);
    }

    private static String modulePath(int module) {
        return ":" + moduleName(module);
    }

    private static String moduleName(int module) {
        return String.format(Locale.ROOT, "module-%03d", module);
    }

    private static String packageName(int module) {
        return String.format(Locale.ROOT, "module%03d", module);
    }

    private static String className(int module, int source) {
        return String.format(Locale.ROOT, "Node%03d%03d", module, source);
    }

    private static String groovyQuote(Path path) {
        return path.toString().replace("\\", "\\\\").replace("'", "\\'");
    }

    private static void writeReport(
            Path report,
            String benchmarkDigest,
            String gradle,
            int java,
            List<FixtureResult> results)
            throws IOException {
        if (report.getParent() == null || Files.exists(report)) {
            throw new IllegalArgumentException("report path must be a new file with a parent");
        }
        Files.createDirectories(report.getParent());
        StringBuilder json = new StringBuilder();
        json.append("{\n")
                .append("  \"schemaVersion\": \"buildopt.evidence/beta-gradle-fixtures/v1\",\n")
                .append("  \"benchmarkDigest\": ").append(jsonString(benchmarkDigest)).append(",\n")
                .append("  \"gradle\": ").append(jsonString(gradle.replace("gradle-", ""))).append(",\n")
                .append("  \"jdk\": ").append(java).append(",\n")
                .append("  \"platform\": \"linux-amd64\",\n")
                .append("  \"fixtures\": [\n");
        for (int index = 0; index < results.size(); index++) {
            FixtureResult result = results.get(index);
            Profile profile = result.profile();
            json.append("    {\n")
                    .append("      \"id\": ").append(jsonString(profile.id())).append(",\n")
                    .append("      \"sizeClass\": ").append(jsonString(profile.sizeClass())).append(",\n")
                    .append("      \"dsl\": \"KOTLIN\",\n")
                    .append("      \"projects\": ").append(profile.projects()).append(",\n")
                    .append("      \"sourceFiles\": ").append(profile.expectedValue()).append(",\n")
                    .append("      \"expectedValue\": ").append(profile.expectedValue()).append(",\n")
                    .append("      \"actualValue\": ").append(result.actualValue()).append(",\n")
                    .append("      \"criticalPath\": [");
            for (int task = 0; task < result.criticalPath().size(); task++) {
                if (task > 0) {
                    json.append(", ");
                }
                json.append(jsonString(result.criticalPath().get(task)));
            }
            json.append("],\n")
                    .append("      \"coldOutcome\": \"SUCCESS\",\n")
                    .append("      \"replayOutcome\": \"FROM_CACHE\",\n")
                    .append("      \"configurationCacheReused\": true,\n")
                    .append("      \"managedL1Populated\": true\n")
                    .append("    }");
            json.append(index + 1 == results.size() ? "\n" : ",\n");
        }
        json.append("  ],\n")
                .append("  \"containsSecrets\": false,\n")
                .append("  \"boundaries\": {\n")
                .append("    \"closes\": [\"OPS-001/A1-GRADLE-FIXTURE-SIZE-MATRIX\"],\n")
                .append("    \"doesNotClose\": [\"OPS-001/A1\", \"A1-G02\", \"EIGHT_HOUR_SOAK\"]\n")
                .append("  }\n")
                .append("}\n");
        Files.createFile(
                report,
                PosixFilePermissions.asFileAttribute(
                        PosixFilePermissions.fromString("rw-------")));
        Files.writeString(report, json, StandardCharsets.UTF_8);
    }

    private static String jsonString(String value) {
        return "\""
                + value.replace("\\", "\\\\").replace("\"", "\\\"")
                + "\"";
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

    private record Profile(
            String id,
            String sizeClass,
            int projects,
            int sourcesPerProject,
            int expectedValue) {}

    private record FixtureResult(Profile profile, List<String> criticalPath, int actualValue) {}
}
