package dev.buildopt.gradle;

import java.io.Serial;
import java.io.Serializable;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import org.gradle.api.Action;
import org.gradle.api.Describable;
import org.gradle.api.Project;
import org.gradle.api.Task;
import org.gradle.api.internal.artifacts.TransformRegistration;
import org.gradle.api.internal.artifacts.VariantTransformRegistry;
import org.gradle.api.internal.project.ProjectInternal;
import org.gradle.api.plugins.JavaPluginExtension;
import org.gradle.api.specs.Spec;
import org.gradle.api.tasks.SourceSet;
import org.gradle.api.tasks.compile.JavaCompile;
import org.gradle.api.tasks.testing.Test;

/** Exact, fail-closed cache policy for the currently proven Tier 1 rows. */
final class BuildOptTierOnePolicy {
    static final String GLOBAL_DISABLE_REASON =
            "BuildOpt managed cache is disabled for this invocation";
    static final String TASK_DEFAULT_DENY_REASON =
            "BuildOpt Tier 1 default-deny allowlist rejected this task";
    static final String TEST_WITHOUT_GRANT_REASON =
            "BuildOpt Test task has no TestCacheGrant";

    private static final String JAVA_COMPILE_DECORATED =
            "org.gradle.api.tasks.compile.JavaCompile_Decorated";
    private static final String INCREMENTAL_TASK_ACTION =
            "org.gradle.api.internal.project.taskfactory.IncrementalTaskAction";
    private static final String TASK_ACTION_WRAPPER =
            "org.gradle.api.internal.AbstractTask$TaskActionWrapper";
    private static final String ERROR_PRONE_PLUGIN_ID = "net.ltgt.errorprone";
    private static final String ERROR_PRONE_ACTION_DISPLAY_NAME =
            "Execute Configure forking for errorprone";
    private static final String ERROR_PRONE_COMPILER_ARGUMENT_PROVIDER =
            "net.ltgt.gradle.errorprone.ErrorProneCompilerArgumentProvider";
    private static final String ERROR_PRONE_JVM_ARGUMENT_PROVIDER =
            "net.ltgt.gradle.errorprone.ErrorProneJvmArgumentProvider";
    private static final String ERROR_PRONE_PLUGIN_ARTIFACT =
            "gradle-errorprone-plugin-4.3.0.jar";
    private static final String GRAALVM_JAR_ANALYZER_TRANSFORM =
            "org.graalvm.buildtools.gradle.tasks.scanner.JarAnalyzerTransform";
    private static final String GRAALVM_NATIVE_PLUGIN_ARTIFACT =
            "native-gradle-plugin-0.11.1.jar";
    private static final String KOTLIN_BUILDTOOLS_SNAPSHOT_TRANSFORM =
            "org.jetbrains.kotlin.gradle.internal.transforms."
                    + "BuildToolsApiClasspathEntrySnapshotTransform";
    private static final String KOTLIN_SCRIPT_EXTENSIONS_TRANSFORM =
            "org.jetbrains.kotlin.gradle.scripting.internal."
                    + "DiscoverScriptExtensionsTransformAction";
    private static final String KOTLIN_GRADLE_PLUGIN_ARTIFACT =
            "kotlin-gradle-plugin-2.2.0-gradle813.jar";

    private BuildOptTierOnePolicy() {}

    static InvocationDecision inspectInvocation(Project rootProject) {
        String gradleVersion = rootProject.getGradle().getGradleVersion();
        int javaFeature = Runtime.version().feature();
        String operatingSystem =
                System.getProperty("os.name", "").toLowerCase(Locale.ROOT);
        String architecture =
                System.getProperty("os.arch", "").toLowerCase(Locale.ROOT);
        if (!isProvenRuntime(
                gradleVersion,
                javaFeature,
                operatingSystem,
                architecture)) {
            return InvocationDecision.disabled("unsupported runtime");
        }

        try {
            List<String> unknownTransforms = new ArrayList<>();
            for (Project project : rootProject.getAllprojects()) {
                ProjectInternal internal = (ProjectInternal) project;
                VariantTransformRegistry registry =
                        internal.getServices().get(VariantTransformRegistry.class);
                for (TransformRegistration registration : registry.getRegistrations()) {
                    Class<?> implementation =
                            registration
                                    .getTransformStep()
                                    .getTransform()
                                    .getImplementationClass();
                    if (isAllowlistedArtifactTransform(implementation)) {
                        continue;
                    }
                    if (!unknownTransforms.contains(implementation.getName())) {
                        unknownTransforms.add(implementation.getName());
                    }
                }
            }
            if (!unknownTransforms.isEmpty()) {
                return InvocationDecision.disabled(
                        "unknown artifact transforms " + String.join(",", unknownTransforms));
            }
        } catch (LinkageError | RuntimeException failure) {
            return InvocationDecision.disabled("artifact transform inventory unavailable");
        }
        return InvocationDecision.enabled();
    }

    private static boolean isAllowlistedArtifactTransform(Class<?> implementation) {
        String expectedArtifact = switch (implementation.getName()) {
            case GRAALVM_JAR_ANALYZER_TRANSFORM -> GRAALVM_NATIVE_PLUGIN_ARTIFACT;
            case KOTLIN_BUILDTOOLS_SNAPSHOT_TRANSFORM -> KOTLIN_GRADLE_PLUGIN_ARTIFACT;
            case KOTLIN_SCRIPT_EXTENSIONS_TRANSFORM -> KOTLIN_GRADLE_PLUGIN_ARTIFACT;
            default -> "";
        };
        if (expectedArtifact.isEmpty()) {
            return false;
        }
        try {
            Path codeSource =
                    Path.of(
                            implementation
                                    .getProtectionDomain()
                                    .getCodeSource()
                                    .getLocation()
                                    .toURI());
            return codeSource.getFileName().toString().equals(expectedArtifact);
        } catch (Exception failure) {
            return false;
        }
    }

    private static boolean isClassFromArtifact(
            Object value, String className, String artifactName) {
        if (!value.getClass().getName().equals(className)) {
            return false;
        }
        try {
            Path codeSource =
                    Path.of(
                            value.getClass()
                                    .getProtectionDomain()
                                    .getCodeSource()
                                    .getLocation()
                                    .toURI());
            return codeSource.getFileName().toString().equals(artifactName);
        } catch (Exception failure) {
            return false;
        }
    }

    private static boolean hasAllowlistedActions(Task task, boolean errorPronePluginApplied) {
        List<Action<? super Task>> actions = task.getActions();
        if (actions.size() == 1) {
            return actions.get(0).getClass().getName().equals(INCREMENTAL_TASK_ACTION);
        }
        if (actions.size() != 2
                || !errorPronePluginApplied
                || !actions.get(0).getClass().getName().equals(TASK_ACTION_WRAPPER)
                || !(actions.get(0) instanceof Describable describable)
                || !describable.getDisplayName().equals(ERROR_PRONE_ACTION_DISPLAY_NAME)
                || !actions.get(1).getClass().getName().equals(INCREMENTAL_TASK_ACTION)
                || !(task instanceof JavaCompile javaCompile)) {
            return false;
        }
        List<?> compilerArgumentProviders = javaCompile.getOptions().getCompilerArgumentProviders();
        List<?> jvmArgumentProviders =
                javaCompile.getOptions().getForkOptions().getJvmArgumentProviders();
        return compilerArgumentProviders.size() == 1
                && isClassFromArtifact(
                        compilerArgumentProviders.get(0),
                        ERROR_PRONE_COMPILER_ARGUMENT_PROVIDER,
                        ERROR_PRONE_PLUGIN_ARTIFACT)
                && jvmArgumentProviders.size() == 1
                && isClassFromArtifact(
                        jvmArgumentProviders.get(0),
                        ERROR_PRONE_JVM_ARGUMENT_PROVIDER,
                        ERROR_PRONE_PLUGIN_ARTIFACT);
    }

    static boolean isAllowlistedIdentity(Task task) {
        if (!task.getClass().getName().equals(JAVA_COMPILE_DECORATED)
                || task.getClass().getSuperclass() != JavaCompile.class
                || !task.getProject().getPluginManager().hasPlugin("java")) {
            return false;
        }
        JavaPluginExtension java =
                task.getProject().getExtensions().findByType(JavaPluginExtension.class);
        if (java == null) {
            return false;
        }
        for (SourceSet sourceSet : java.getSourceSets()) {
            if (sourceSet.getCompileJavaTaskName().equals(task.getName())) {
                return true;
            }
        }
        return false;
    }

    static boolean isTestTask(Task task) {
        return task instanceof Test;
    }

    static boolean isErrorPronePluginApplied(Task task) {
        return task.getProject().getPluginManager().hasPlugin(ERROR_PRONE_PLUGIN_ID);
    }

    private static boolean isProvenRuntime(
            String gradleVersion,
            int javaFeature,
            String operatingSystem,
            String architecture) {
        if (!operatingSystem.contains("linux")
                || !(architecture.equals("amd64") || architecture.equals("x86_64"))) {
            return false;
        }
        return switch (gradleVersion) {
            case "8.14.2" -> javaFeature == 21;
            case "8.14.3" -> javaFeature == 17 || javaFeature == 21;
            case "9.6.1" ->
                    javaFeature == 17
                            || javaFeature == 21
                            || javaFeature == 25;
            default -> false;
        };
    }

    record InvocationDecision(boolean managedCacheEnabled, String reason) {
        static InvocationDecision enabled() {
            return new InvocationDecision(true, "");
        }

        static InvocationDecision disabled(String reason) {
            return new InvocationDecision(false, reason);
        }
    }

    static final class InvocationDisableSpec implements Spec<Task>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        private final boolean managedCacheEnabled;

        InvocationDisableSpec(boolean managedCacheEnabled) {
            this.managedCacheEnabled = managedCacheEnabled;
        }

        @Override
        public boolean isSatisfiedBy(Task task) {
            return !managedCacheEnabled;
        }
    }

    static final class TaskDefaultDenySpec implements Spec<Task>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        private final boolean allowlistedIdentity;
        private final boolean testTask;
        private final boolean errorPronePluginApplied;

        TaskDefaultDenySpec(
                boolean allowlistedIdentity,
                boolean testTask,
                boolean errorPronePluginApplied) {
            this.allowlistedIdentity = allowlistedIdentity;
            this.testTask = testTask;
            this.errorPronePluginApplied = errorPronePluginApplied;
        }

        @Override
        public boolean isSatisfiedBy(Task task) {
            if (testTask) {
                return false;
            }
            if (!allowlistedIdentity) {
                return true;
            }
            return !hasAllowlistedActions(task, errorPronePluginApplied);
        }
    }

    static final class TestWithoutGrantSpec implements Spec<Task>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        private final boolean testTask;

        TestWithoutGrantSpec(boolean testTask) {
            this.testTask = testTask;
        }

        @Override
        public boolean isSatisfiedBy(Task task) {
            return testTask;
        }
    }
}
