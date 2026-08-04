package dev.buildopt.gradle;

import java.io.Serial;
import java.io.Serializable;
import org.gradle.api.Action;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.Task;
import org.gradle.api.execution.TaskExecutionGraph;

/**
 * Restricts a managed cache to the exact Tier 1 allowlist.
 *
 * <p>This plugin never enables or configures a build cache. The managed cache
 * owner applies it only after selecting the managed path; the neutral
 * {@code dev.buildopt} rendezvous remains behavior-preserving.
 */
public final class BuildOptTierOnePolicyPlugin implements Plugin<Project> {
    @Override
    public void apply(Project target) {
        Project root = target.getRootProject();
        if (target != root) {
            root.getPluginManager().apply(BuildOptTierOnePolicyPlugin.class);
            return;
        }
        target.getGradle()
                .projectsEvaluated(
                        ignored -> {
                            BuildOptTierOnePolicy.InvocationDecision invocation =
                                    BuildOptTierOnePolicy.inspectInvocation(root);
                            if (!invocation.managedCacheEnabled()) {
                                target.getGradle()
                                        .getStartParameter()
                                        .setBuildCacheEnabled(false);
                            }
                            String globalDisableReason =
                                    invocation.managedCacheEnabled()
                                            ? BuildOptTierOnePolicy.GLOBAL_DISABLE_REASON
                                            : BuildOptTierOnePolicy.GLOBAL_DISABLE_REASON
                                                    + ": "
                                                    + invocation.reason();
                            target.getGradle()
                                    .getTaskGraph()
                                    .whenReady(
                                            new ConfigureSelectedTasksAction(
                                                    invocation.managedCacheEnabled(),
                                                    globalDisableReason));
                        });
    }

    /** Applies the fail-closed policy only to tasks selected for this build. */
    private static final class ConfigureSelectedTasksAction
            implements Action<TaskExecutionGraph>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        private final boolean managedCacheEnabled;
        private final String globalDisableReason;

        private ConfigureSelectedTasksAction(
                boolean managedCacheEnabled,
                String globalDisableReason) {
            this.managedCacheEnabled = managedCacheEnabled;
            this.globalDisableReason = globalDisableReason;
        }

        @Override
        public void execute(TaskExecutionGraph graph) {
            for (Task task : graph.getAllTasks()) {
                boolean allowlistedIdentity =
                        BuildOptTierOnePolicy.isAllowlistedIdentity(task);
                boolean testTask = BuildOptTierOnePolicy.isTestTask(task);
                if (!managedCacheEnabled) {
                    task.notCompatibleWithConfigurationCache(
                            globalDisableReason);
                }
                task.getOutputs()
                        .doNotCacheIf(
                                globalDisableReason,
                                new BuildOptTierOnePolicy.InvocationDisableSpec(
                                        managedCacheEnabled));
                task.getOutputs()
                        .doNotCacheIf(
                                BuildOptTierOnePolicy.TASK_DEFAULT_DENY_REASON,
                                new BuildOptTierOnePolicy.TaskDefaultDenySpec(
                                        allowlistedIdentity,
                                        testTask));
                task.getOutputs()
                        .doNotCacheIf(
                                BuildOptTierOnePolicy.TEST_WITHOUT_GRANT_REASON,
                                new BuildOptTierOnePolicy.TestWithoutGrantSpec(
                                        testTask));
            }
        }
    }
}
