package example;

import org.gradle.api.Plugin;
import org.gradle.api.Project;

/** Stable included-build plugin implementation for the A0-G08 fixture. */
public final class IncludedPlugin implements Plugin<Project> {
    @Override
    public void apply(Project project) {
        // The fixture proves the plugin build boundary, not plugin behavior.
    }
}
