package dev.buildopt.fixtures.correlation;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.List;
import org.gradle.api.DefaultTask;
import org.gradle.api.GradleException;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.file.DirectoryProperty;
import org.gradle.api.file.RegularFileProperty;
import org.gradle.api.provider.ListProperty;
import org.gradle.api.provider.Property;
import org.gradle.api.tasks.CacheableTask;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.Internal;
import org.gradle.api.tasks.OutputFile;
import org.gradle.api.tasks.TaskAction;

/** Installs one deterministic cacheable task in each correlation-fixture project. */
public final class CorrelationFixturePlugin implements Plugin<Project> {
    private static final List<String> EXPECTED_MARKERS =
            List.of("alpha.ready", "beta.ready");

    @Override
    public void apply(Project project) {
        project.getPluginManager().apply("base");
        project.getTasks().register(
                "correlationFixture",
                CorrelationFixtureTask.class,
                task -> {
                    task.setGroup("verification");
                    task.setDescription(
                            "Produces an output equivalent to the peer fixture project.");
                    task.getPayload().set("equivalent-output-v1\n");
                    task.getMarkerName().set(project.getName() + ".ready");
                    task.getExpectedMarkers().set(EXPECTED_MARKERS);
                    task.getBarrierDirectory()
                            .set(project.getRootProject()
                                    .getLayout()
                                    .getBuildDirectory()
                                    .dir("correlation-barrier"));
                    task.getOutputFile()
                            .set(project.getLayout()
                                    .getBuildDirectory()
                                    .file("correlation/output.txt"));
                });
    }

    /**
     * Produces the same cache key and bytes in two projects while requiring their
     * first executions to overlap.
     */
    @CacheableTask
    public abstract static class CorrelationFixtureTask extends DefaultTask {
        private static final Duration BARRIER_TIMEOUT = Duration.ofSeconds(10);

        @Input
        public abstract Property<String> getPayload();

        @OutputFile
        public abstract RegularFileProperty getOutputFile();

        @Internal
        public abstract DirectoryProperty getBarrierDirectory();

        @Internal
        public abstract Property<String> getMarkerName();

        @Internal
        public abstract ListProperty<String> getExpectedMarkers();

        @TaskAction
        public void produce() {
            Path barrier = getBarrierDirectory().get().getAsFile().toPath();
            try {
                Files.createDirectories(barrier);
                Files.writeString(
                        barrier.resolve(getMarkerName().get()),
                        "ready\n",
                        StandardCharsets.UTF_8);
                awaitPeers(barrier);

                Path output = getOutputFile().get().getAsFile().toPath();
                Files.createDirectories(output.getParent());
                Files.writeString(output, getPayload().get(), StandardCharsets.UTF_8);
            } catch (IOException exception) {
                throw new GradleException("cannot materialize correlation fixture", exception);
            }
        }

        private void awaitPeers(Path barrier) {
            long deadline = System.nanoTime() + BARRIER_TIMEOUT.toNanos();
            while (getExpectedMarkers().get().stream()
                    .map(barrier::resolve)
                    .anyMatch(marker -> !Files.isRegularFile(marker))) {
                if (System.nanoTime() >= deadline) {
                    throw new GradleException(
                            "parallel fixture barrier timed out in " + barrier);
                }
                try {
                    Thread.sleep(10);
                } catch (InterruptedException exception) {
                    Thread.currentThread().interrupt();
                    throw new GradleException(
                            "parallel fixture barrier was interrupted", exception);
                }
            }
        }
    }
}
