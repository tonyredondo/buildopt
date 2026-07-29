package dev.buildopt.fixtures.tierone;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.SecureRandom;
import java.time.Clock;
import java.util.List;
import java.util.Locale;
import java.util.Random;
import java.util.TimeZone;
import java.util.concurrent.ThreadLocalRandom;
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
import org.gradle.api.tasks.UntrackedTask;

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
        project.getTasks()
                .register(
                        "agentProbe",
                        AgentProbeTask.class,
                        task ->
                                task.getOutputFile()
                                        .set(
                                                project.getLayout()
                                                        .getBuildDirectory()
                                                        .file("agent/probe.txt")));
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

    /** Executes every access class required by the bounded JVM Agent spike. */
    @UntrackedTask(because = "SPK-002 intentionally executes external access on every run")
    public abstract static class AgentProbeTask extends DefaultTask {
        /** Returns the deterministic proof that all access classes executed. */
        @OutputFile
        public abstract RegularFileProperty getOutputFile();

        /** Performs real accesses without persisting any observed values. */
        @TaskAction
        public final void probe() {
            Path temporary = getTemporaryDir().toPath().resolve("probe.txt");
            try {
                Files.createDirectories(temporary.getParent());
                Files.writeString(
                        temporary,
                        "agent-probe\n",
                        StandardCharsets.UTF_8);
                Files.readString(temporary, StandardCharsets.UTF_8);
                try (FileInputStream input = new FileInputStream(temporary.toFile())) {
                    if (input.read() < 0) {
                        throw new IllegalStateException("empty I/O probe");
                    }
                }

                System.getenv("PATH");
                System.getProperty("java.version");

                Process process =
                        new ProcessBuilder("sh", "-c", "exit 0")
                                .redirectErrorStream(true)
                                .start();
                if (process.waitFor() != 0) {
                    throw new IllegalStateException("process probe failed");
                }

                try (Socket socket = new Socket()) {
                    try {
                        socket.connect(new InetSocketAddress("127.0.0.1", 9), 100);
                    } catch (IOException expected) {
                        // A refused loopback connection still proves a real network attempt.
                    }
                }

                Clock.systemUTC().instant();
                Locale.getDefault();
                TimeZone.getDefault();
                new Random(1).nextInt();
                new SecureRandom().nextInt();
                ThreadLocalRandom.current().nextInt();

                File output = getOutputFile().get().getAsFile();
                Files.createDirectories(output.toPath().getParent());
                Files.writeString(
                        output.toPath(),
                        String.join(
                                        "\n",
                                        "IO_NIO=EXECUTED",
                                        "ENVIRONMENT_PROPERTIES=EXECUTED",
                                        "PROCESS=EXECUTED",
                                        "NETWORK=EXECUTED",
                                        "CLOCK_LOCALE_TIMEZONE=EXECUTED",
                                        "RANDOMNESS=EXECUTED")
                                + "\n",
                        StandardCharsets.UTF_8);
            } catch (InterruptedException exception) {
                Thread.currentThread().interrupt();
                throw new IllegalStateException("process probe interrupted", exception);
            } catch (IOException exception) {
                throw new IllegalStateException("agent access probe failed", exception);
            }
        }
    }
}
