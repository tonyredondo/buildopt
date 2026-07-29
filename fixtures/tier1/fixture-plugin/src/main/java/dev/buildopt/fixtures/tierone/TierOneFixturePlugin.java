package dev.buildopt.fixtures.tierone;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.util.List;
import org.gradle.api.DefaultTask;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.artifacts.Configuration;
import org.gradle.api.artifacts.transform.InputArtifact;
import org.gradle.api.artifacts.transform.TransformAction;
import org.gradle.api.artifacts.transform.TransformOutputs;
import org.gradle.api.artifacts.transform.TransformParameters;
import org.gradle.api.attributes.Attribute;
import org.gradle.api.file.ConfigurableFileCollection;
import org.gradle.api.file.FileSystemLocation;
import org.gradle.api.file.RegularFileProperty;
import org.gradle.api.provider.Provider;
import org.gradle.api.tasks.CacheableTask;
import org.gradle.api.tasks.InputFiles;
import org.gradle.api.tasks.OutputFile;
import org.gradle.api.tasks.PathSensitive;
import org.gradle.api.tasks.PathSensitivity;
import org.gradle.api.tasks.TaskAction;

/** Shared fixture logic used from both Kotlin and Groovy DSL repositories. */
public final class TierOneFixturePlugin implements Plugin<Project> {
    private static final Attribute<String> ARTIFACT_TYPE =
            Attribute.of("artifactType", String.class);

    @Override
    public void apply(Project project) {
        project.getDependencies()
                .registerTransform(
                        MarkerTransform.class,
                        specification -> {
                            specification.getFrom().attribute(ARTIFACT_TYPE, "txt");
                            specification.getTo().attribute(ARTIFACT_TYPE, "tier-one-marker");
                        });
        Configuration input =
                project.getConfigurations()
                        .create(
                                "tierOneInput",
                                configuration -> {
                                    configuration.setCanBeConsumed(false);
                                    configuration.setCanBeResolved(true);
                                    configuration
                                            .getAttributes()
                                            .attribute(ARTIFACT_TYPE, "tier-one-marker");
                                });
        project.getDependencies()
                .add(input.getName(), project.files("input/source.txt"));
        project.getTasks()
                .register(
                        "verifyTierOne",
                        VerifyTierOneTask.class,
                        task -> {
                            task.getTransformedInput().from(input);
                            task.getOutputFile()
                                    .set(
                                            project.getLayout()
                                                    .getBuildDirectory()
                                                    .file("tier-one/verified.txt"));
                        });
    }

    /** Exact deterministic transform used to prove adapter execution. */
    public abstract static class MarkerTransform
            implements TransformAction<TransformParameters.None> {
        /** Returns Gradle's exact input artifact. */
        @InputArtifact
        public abstract Provider<FileSystemLocation> getInputArtifact();

        @Override
        public void transform(TransformOutputs outputs) {
            File input = getInputArtifact().get().getAsFile();
            File output = outputs.file(input.getName() + ".marked");
            try {
                String value = Files.readString(input.toPath(), StandardCharsets.UTF_8).strip();
                Files.writeString(
                        output.toPath(),
                        value + "|transformed\n",
                        StandardCharsets.UTF_8);
            } catch (IOException exception) {
                throw new IllegalStateException("cannot transform fixture input", exception);
            }
        }
    }

    /** Cacheable custom task that consumes the transformed artifact. */
    @CacheableTask
    public abstract static class VerifyTierOneTask extends DefaultTask {
        /** Returns the transformed fixture input. */
        @InputFiles
        @PathSensitive(PathSensitivity.RELATIVE)
        public abstract ConfigurableFileCollection getTransformedInput();

        /** Returns the deterministic verification output. */
        @OutputFile
        public abstract RegularFileProperty getOutputFile();

        /** Verifies the transform and materializes the stable fixture marker. */
        @TaskAction
        public final void verify() {
            List<File> files = getTransformedInput().getFiles().stream().sorted().toList();
            if (files.size() != 1) {
                throw new IllegalStateException(
                        "expected one transformed input, found " + files.size());
            }
            try {
                String transformed =
                        Files.readString(files.get(0).toPath(), StandardCharsets.UTF_8);
                if (!transformed.equals("tier-one-input|transformed\n")) {
                    throw new IllegalStateException("unexpected transform output");
                }
                File output = getOutputFile().get().getAsFile();
                Files.createDirectories(output.toPath().getParent());
                Files.writeString(
                        output.toPath(),
                        transformed.strip() + "|verified\n",
                        StandardCharsets.UTF_8);
            } catch (IOException exception) {
                throw new IllegalStateException("cannot verify fixture input", exception);
            }
        }
    }
}
