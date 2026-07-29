package dev.buildopt.gradle;

import org.gradle.api.Plugin;
import org.gradle.api.Project;

/**
 * Neutral project-plugin entry point used to verify the Gradle API and Java 17 build contract.
 *
 * <p>The handshake and all optimization behavior remain gated by later tracker items.
 */
public final class BuildOptProjectPlugin implements Plugin<Project> {
    @Override
    public void apply(Project target) {
        // ENV-004 verifies packaging only; WS-003 introduces the first handshake.
    }
}
