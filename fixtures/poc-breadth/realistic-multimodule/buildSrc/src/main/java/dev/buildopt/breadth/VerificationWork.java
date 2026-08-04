package dev.buildopt.breadth;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import org.gradle.api.DefaultTask;
import org.gradle.api.file.RegularFileProperty;
import org.gradle.api.provider.Property;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.OutputFile;
import org.gradle.api.tasks.TaskAction;
import org.gradle.work.DisableCachingByDefault;

@DisableCachingByDefault(because = "Models deterministic verification that is not cacheable")
public abstract class VerificationWork extends DefaultTask {
    @Input
    public abstract Property<String> getLabel();

    @Input
    public abstract Property<Integer> getRounds();

    @OutputFile
    public abstract RegularFileProperty getOutputFile();

    public VerificationWork() {
        getOutputs().upToDateWhen(task -> false);
    }

    @TaskAction
    public void verify() throws IOException {
        long value = 0x9e3779b97f4a7c15L;
        for (int index = 0; index < getRounds().get(); index++) {
            value ^= (value << 13);
            value ^= (value >>> 7);
            value ^= (value << 17);
            value += index;
        }
        var output = getOutputFile().get().getAsFile().toPath();
        Files.createDirectories(output.getParent());
        Files.writeString(
                output,
                getLabel().get() + ":" + Long.toUnsignedString(value) + "\n",
                StandardCharsets.UTF_8);
    }
}
