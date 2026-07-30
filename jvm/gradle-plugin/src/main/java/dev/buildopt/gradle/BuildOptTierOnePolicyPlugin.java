package dev.buildopt.gradle;

import org.gradle.api.Plugin;
import org.gradle.api.Project;

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
                            for (Project project : root.getAllprojects()) {
                                project.getTasks()
                                        .configureEach(
                                                task -> {
                                                    boolean allowlistedIdentity =
                                                            BuildOptTierOnePolicy
                                                                    .isAllowlistedIdentity(task);
                                                    if (!invocation.managedCacheEnabled()) {
                                                        task.notCompatibleWithConfigurationCache(
                                                                globalDisableReason);
                                                    }
                                                    task.getOutputs()
                                                            .doNotCacheIf(
                                                                    globalDisableReason,
                                                                    new BuildOptTierOnePolicy
                                                                            .InvocationDisableSpec(
                                                                            invocation
                                                                                    .managedCacheEnabled()));
                                                    task.getOutputs()
                                                            .doNotCacheIf(
                                                                    BuildOptTierOnePolicy
                                                                            .TASK_DEFAULT_DENY_REASON,
                                                                    new BuildOptTierOnePolicy
                                                                            .TaskDefaultDenySpec(
                                                                            allowlistedIdentity));
                                                });
                            }
                        });
    }
}
