package dev.buildopt.gradle;

import java.io.File;
import java.io.Serial;
import java.io.Serializable;
import java.nio.file.Path;
import java.util.List;
import org.gradle.api.Action;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.Task;
import org.gradle.api.specs.Spec;
import org.gradle.api.tasks.Copy;
import org.gradle.util.GradleVersion;

/** Adds cache eligibility only to unmodified standard Gradle Copy tasks. */
public final class BuildOptStandardCopyCachePlugin implements Plugin<Project> {
    static final String CACHE_REASON = "BuildOpt POC standard Copy task cache";

    private static final String SUPPORTED_GRADLE_VERSION = "9.6.1";
    private static final String STANDARD_TASK_ACTION =
            "org.gradle.api.internal.project.taskfactory.StandardTaskAction";
    private static final String COPY_DECORATED = "org.gradle.api.tasks.Copy_Decorated";
    private static final String COPY_BASE = "org.gradle.api.tasks.Copy";

    @Override
    public void apply(Project target) {
        Project root = target.getRootProject();
        if (target != root) {
            root.getPluginManager().apply(BuildOptStandardCopyCachePlugin.class);
            return;
        }
        if (!GradleVersion.current().getVersion().equals(SUPPORTED_GRADLE_VERSION)) {
            return;
        }
        root.getGradle()
                .projectsEvaluated(
                        ignored -> {
                            for (Project project : root.getAllprojects()) {
                                project.getTasks()
                                        .configureEach(
                                                task -> {
                                                    if (isStandardBuildDirectoryCopy(project, task)) {
                                                        task.getOutputs()
                                                                .cacheIf(
                                                                        CACHE_REASON,
                                                                        new AlwaysCacheSpec());
                                                    }
                                                });
                            }
                        });
    }

    private static boolean isStandardBuildDirectoryCopy(Project project, Task task) {
        List<Action<? super Task>> actions = task.getActions();
        if (!task.getClass().getName().equals(COPY_DECORATED)
                || !task.getClass().getSuperclass().getName().equals(COPY_BASE)
                || actions.size() != 1
                || !actions.get(0).getClass().getName().equals(STANDARD_TASK_ACTION)) {
            return false;
        }

        File destination = ((Copy) task).getDestinationDir();
        File buildDirectory = project.getLayout().getBuildDirectory().get().getAsFile();
        if (destination == null) {
            return false;
        }
        Path normalizedDestination = destination.toPath().toAbsolutePath().normalize();
        Path normalizedBuildDirectory = buildDirectory.toPath().toAbsolutePath().normalize();
        return !normalizedDestination.equals(normalizedBuildDirectory)
                && normalizedDestination.startsWith(normalizedBuildDirectory);
    }

    private static final class AlwaysCacheSpec implements Spec<Task>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        @Override
        public boolean isSatisfiedBy(Task task) {
            return true;
        }
    }
}
