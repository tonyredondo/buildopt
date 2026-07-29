package dev.buildopt.fixtures.correlation;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.List;
import javax.inject.Inject;
import org.gradle.api.BuildCancelledException;
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
import org.gradle.process.ExecOperations;
import org.gradle.workers.WorkAction;
import org.gradle.workers.WorkParameters;
import org.gradle.workers.WorkQueue;
import org.gradle.workers.WorkerExecutor;

/** Installs one deterministic cacheable task in each correlation-fixture project. */
public final class CorrelationFixturePlugin implements Plugin<Project> {
    private static final String PAYLOAD = "equivalent-output-v1\n";

    @Override
    public void apply(Project project) {
        project.getPluginManager().apply("base");
        project.getTasks().register(
                "correlationFixture",
                CorrelationFixtureTask.class,
                task -> configure(task, project, "correlation"));
        project.getTasks().register(
                "workerNoIsolationFixture",
                NoIsolationWorkerFixtureTask.class,
                task -> configure(task, project, "worker-no-isolation"));
        project.getTasks().register(
                "workerProcessIsolationFixture",
                ProcessIsolationWorkerFixtureTask.class,
                task -> configure(task, project, "worker-process-isolation"));
        project.getTasks().register(
                "childProcessFixture",
                ChildProcessFixtureTask.class,
                task -> configure(task, project, "child-process"));
        project.getTasks().register(
                "failureFixture",
                FailureFixtureTask.class,
                task -> configure(task, project, "failure"));
        project.getTasks().register(
                "cancellationFixture",
                CancellationFixtureTask.class,
                task -> configure(task, project, "cancellation"));
    }

    private static void configure(
            CorrelationTask task,
            Project project,
            String variant) {
        task.setGroup("verification");
        task.setDescription("Exercises the " + variant + " correlation path.");
        task.getPayload().set(PAYLOAD);
        task.getMarkerName().set(project.getName() + ".ready");
        task.getExpectedMarkers().set(
                "correlation".equals(variant)
                        ? List.of("alpha.ready", "beta.ready")
                        : List.of(project.getName() + ".ready"));
        task.getBarrierDirectory()
                .set(project.getRootProject()
                        .getLayout()
                        .getBuildDirectory()
                        .dir(variant + "-barrier"));
        task.getOutputFile()
                .set(project.getLayout()
                        .getBuildDirectory()
                        .file(variant + "/output.txt"));
    }

    /** Shared declared inputs and outputs for every correlation path. */
    public abstract static class CorrelationTask extends DefaultTask {
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

        protected final void materialize() {
            CorrelationFixturePlugin.materialize(
                    getBarrierDirectory().get().getAsFile().toPath(),
                    getMarkerName().get(),
                    getExpectedMarkers().get(),
                    getOutputFile().get().getAsFile().toPath(),
                    getPayload().get());
        }
    }

    /**
     * Produces the same cache key and bytes in two projects while requiring their
     * first executions to overlap.
     */
    @CacheableTask
    public abstract static class CorrelationFixtureTask extends CorrelationTask {
        @TaskAction
        public void produce() {
            materialize();
        }
    }

    /** Runs the fixture work through Worker API no-isolation mode. */
    @CacheableTask
    public abstract static class NoIsolationWorkerFixtureTask extends CorrelationTask {
        private final WorkerExecutor workerExecutor;

        @Inject
        public NoIsolationWorkerFixtureTask(WorkerExecutor workerExecutor) {
            this.workerExecutor = workerExecutor;
        }

        @TaskAction
        public void produce() {
            submit(workerExecutor.noIsolation());
        }

        private void submit(WorkQueue queue) {
            configureWork(queue, this);
            queue.await();
        }
    }

    /** Runs the fixture work in a process-isolated Worker API daemon. */
    @CacheableTask
    public abstract static class ProcessIsolationWorkerFixtureTask
            extends CorrelationTask {
        private final WorkerExecutor workerExecutor;

        @Inject
        public ProcessIsolationWorkerFixtureTask(WorkerExecutor workerExecutor) {
            this.workerExecutor = workerExecutor;
        }

        @TaskAction
        public void produce() {
            WorkQueue queue = workerExecutor.processIsolation();
            configureWork(queue, this);
            queue.await();
        }
    }

    /** Executes a real child JVM before materializing the declared output. */
    @CacheableTask
    public abstract static class ChildProcessFixtureTask extends CorrelationTask {
        private final ExecOperations execOperations;

        @Inject
        public ChildProcessFixtureTask(ExecOperations execOperations) {
            this.execOperations = execOperations;
        }

        @TaskAction
        public void produce() {
            Path barrier = getBarrierDirectory().get().getAsFile().toPath();
            markAndAwait(
                    barrier,
                    getMarkerName().get(),
                    getExpectedMarkers().get());

            Path javaExecutable = Path.of(
                    System.getProperty("java.home"),
                    "bin",
                    "java");
            Path codeSource;
            try {
                codeSource = Path.of(ChildProcessMain.class
                        .getProtectionDomain()
                        .getCodeSource()
                        .getLocation()
                        .toURI());
            } catch (Exception exception) {
                throw new GradleException(
                        "cannot locate child-process fixture classes",
                        exception);
            }

            ByteArrayOutputStream stdout = new ByteArrayOutputStream();
            ByteArrayOutputStream stderr = new ByteArrayOutputStream();
            execOperations.exec(spec -> {
                spec.setExecutable(javaExecutable.toFile());
                spec.args(
                        "-cp",
                        codeSource.toString(),
                        ChildProcessMain.class.getName(),
                        getPayload().get());
                spec.setStandardOutput(stdout);
                spec.setErrorOutput(stderr);
            }).assertNormalExitValue();

            if (stderr.size() != 0) {
                throw new GradleException(
                        "child-process fixture wrote stderr: "
                                + stderr.toString(StandardCharsets.UTF_8));
            }
            writeOutput(
                    getOutputFile().get().getAsFile().toPath(),
                    stdout.toString(StandardCharsets.UTF_8));
        }
    }

    /** Writes a declared output and then terminates with an ordinary failure. */
    @CacheableTask
    public abstract static class FailureFixtureTask extends CorrelationTask {
        @TaskAction
        public void fail() {
            writeOutput(
                    getOutputFile().get().getAsFile().toPath(),
                    getPayload().get());
            throw new GradleException("intentional correlation fixture failure");
        }
    }

    /** Writes a declared output and then terminates as a cancelled build. */
    @CacheableTask
    public abstract static class CancellationFixtureTask extends CorrelationTask {
        @TaskAction
        public void cancel() {
            writeOutput(
                    getOutputFile().get().getAsFile().toPath(),
                    getPayload().get());
            throw new BuildCancelledException(
                    "intentional correlation fixture cancellation");
        }
    }

    /** Worker parameters are serialized for both no- and process-isolated modes. */
    public interface CorrelationWorkParameters extends WorkParameters {
        Property<String> getPayload();

        RegularFileProperty getOutputFile();

        DirectoryProperty getBarrierDirectory();

        Property<String> getMarkerName();

        ListProperty<String> getExpectedMarkers();
    }

    /** Materializes one fixture result inside the selected Worker API mode. */
    public abstract static class CorrelationWorkAction
            implements WorkAction<CorrelationWorkParameters> {
        @Override
        public void execute() {
            CorrelationWorkParameters parameters = getParameters();
            materialize(
                    parameters.getBarrierDirectory().get().getAsFile().toPath(),
                    parameters.getMarkerName().get(),
                    parameters.getExpectedMarkers().get(),
                    parameters.getOutputFile().get().getAsFile().toPath(),
                    parameters.getPayload().get());
        }
    }

    /** Minimal entrypoint proving that the child-process path leaves the daemon. */
    public static final class ChildProcessMain {
        private ChildProcessMain() {}

        public static void main(String[] arguments) {
            if (arguments.length != 1) {
                throw new IllegalArgumentException("expected one payload argument");
            }
            System.out.print(arguments[0]);
        }
    }

    private static void configureWork(
            WorkQueue queue,
            CorrelationTask task) {
        queue.submit(CorrelationWorkAction.class, parameters -> {
            parameters.getPayload().set(task.getPayload());
            parameters.getOutputFile().set(task.getOutputFile());
            parameters.getBarrierDirectory().set(task.getBarrierDirectory());
            parameters.getMarkerName().set(task.getMarkerName());
            parameters.getExpectedMarkers().set(task.getExpectedMarkers());
        });
    }

    private static void materialize(
            Path barrier,
            String markerName,
            List<String> expectedMarkers,
            Path output,
            String payload) {
        markAndAwait(barrier, markerName, expectedMarkers);
        writeOutput(output, payload);
    }

    private static void markAndAwait(
            Path barrier,
            String markerName,
            List<String> expectedMarkers) {
        try {
            Files.createDirectories(barrier);
            Files.writeString(
                    barrier.resolve(markerName),
                    "ready\n",
                    StandardCharsets.UTF_8);
        } catch (IOException exception) {
            throw new GradleException(
                    "cannot create correlation fixture barrier",
                    exception);
        }

        long deadline = System.nanoTime() + Duration.ofSeconds(10).toNanos();
        while (expectedMarkers.stream()
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
                        "parallel fixture barrier was interrupted",
                        exception);
            }
        }
    }

    private static void writeOutput(Path output, String payload) {
        try {
            Files.createDirectories(output.getParent());
            Files.writeString(output, payload, StandardCharsets.UTF_8);
        } catch (IOException exception) {
            throw new GradleException(
                    "cannot materialize correlation fixture",
                    exception);
        }
    }
}
