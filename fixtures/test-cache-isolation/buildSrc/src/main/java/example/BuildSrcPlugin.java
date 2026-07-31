package example;

import org.gradle.api.Plugin;
import org.gradle.api.Project;

/** Stable buildSrc plugin implementation for the A0-G08 fixture. */
public final class BuildSrcPlugin implements Plugin<Project> {
    @Override
    public void apply(Project project) {
        // The fixture proves the buildSrc boundary, not plugin behavior.
    }
}
