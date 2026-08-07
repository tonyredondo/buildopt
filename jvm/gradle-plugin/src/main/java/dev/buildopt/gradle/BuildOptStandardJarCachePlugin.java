package dev.buildopt.gradle;

import java.io.Serial;
import java.io.Serializable;
import java.util.List;
import org.gradle.api.Action;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.Task;
import org.gradle.api.specs.Spec;
import org.gradle.api.tasks.bundling.Jar;

/** Adds cache eligibility only to unmodified standard Gradle Jar producers. */
public final class BuildOptStandardJarCachePlugin implements Plugin<Project> {
    static final String CACHE_REASON = "BuildOpt POC standard Jar producer cache";

    private static final String STANDARD_TASK_ACTION =
            "org.gradle.api.internal.project.taskfactory.StandardTaskAction";
    private static final String JAR_DECORATED =
            "org.gradle.api.tasks.bundling.Jar_Decorated";
    private static final String JAR_BASE = "org.gradle.api.tasks.bundling.Jar";
    private static final Spec<Task> ALWAYS_CACHE = new AlwaysCacheSpec();

    @Override
    public void apply(Project target) {
        Project root = target.getRootProject();
        if (target != root) {
            root.getPluginManager().apply(BuildOptStandardJarCachePlugin.class);
            return;
        }
        root.getGradle()
                .projectsEvaluated(
                        ignored -> {
                            for (Project project : root.getAllprojects()) {
                                project.getTasks()
                                        .withType(Jar.class)
                                        .configureEach(
                                                task -> {
                                                    if (isStandardJarProducer(task)) {
                                                        task.getOutputs()
                                                                .cacheIf(
                                                                        CACHE_REASON,
                                                                        ALWAYS_CACHE);
                                                    }
                                                });
                            }
                        });
    }

    private static boolean isStandardJarProducer(Task task) {
        List<Action<? super Task>> actions = task.getActions();
        return task.getClass().getName().equals(JAR_DECORATED)
                && task.getClass().getSuperclass().getName().equals(JAR_BASE)
                && actions.size() == 1
                && actions.get(0).getClass().getName().equals(STANDARD_TASK_ACTION);
    }

    private static final class AlwaysCacheSpec implements Spec<Task>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        @Override
        public boolean isSatisfiedBy(Task task) {
            return true;
        }
    }
}
