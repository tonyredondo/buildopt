package dev.buildopt.gradle;

import javax.inject.Inject;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.provider.Provider;
import org.gradle.build.event.BuildEventsListenerRegistry;

/**
 * Project-plugin entry point for the neutral authenticated BuildOpt rendezvous.
 *
 * <p>The shared service is registered as a task-completion listener so every
 * invocation, including a Configuration Cache reuse, sends one
 * {@code ProducerHello}. It does not change task inputs, outputs, cache policy,
 * or execution.
 */
public final class BuildOptProjectPlugin implements Plugin<Project> {
    private final BuildEventsListenerRegistry listenerRegistry;

    /** Creates the plugin with Gradle's configuration-cache-safe event registry. */
    @Inject
    public BuildOptProjectPlugin(BuildEventsListenerRegistry listenerRegistry) {
        this.listenerRegistry = listenerRegistry;
    }

    @Override
    public void apply(Project target) {
        Provider<BuildOptHandshakeService> handshake =
                target.getGradle()
                        .getSharedServices()
                        .registerIfAbsent(
                                "buildOptPluginHandshake",
                                BuildOptHandshakeService.class,
                                ignored -> {});
        listenerRegistry.onTaskCompletion(handshake);
    }
}
