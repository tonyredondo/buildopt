package dev.buildopt.pocvalue;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;

import org.gradle.api.DefaultTask;
import org.gradle.api.file.RegularFileProperty;
import org.gradle.api.provider.Property;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.OutputFile;
import org.gradle.api.tasks.TaskAction;
import org.gradle.work.DisableCachingByDefault;

@DisableCachingByDefault(because = "The POC models deterministic work not reusable by native cache")
public abstract class ExpensiveUnrelated extends DefaultTask {
    @Input
    public abstract Property<Integer> getRounds();

    @OutputFile
    public abstract RegularFileProperty getOutputFile();

    @TaskAction
    public void generate() throws IOException, NoSuchAlgorithmException {
        MessageDigest digest = MessageDigest.getInstance("SHA-256");
        byte[] state = "unrelated-service-work".getBytes(StandardCharsets.UTF_8);
        for (int round = 0; round < getRounds().get(); round++) {
            state = digest.digest(state);
        }
        Path output = getOutputFile().get().getAsFile().toPath();
        Files.createDirectories(output.getParent());
        Files.writeString(output, HexFormat.of().formatHex(state) + "\n");
    }
}
