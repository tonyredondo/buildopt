package dev.buildopt.pilot;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HexFormat;
import java.util.List;

import org.gradle.api.DefaultTask;
import org.gradle.api.file.RegularFileProperty;
import org.gradle.api.provider.ListProperty;
import org.gradle.api.tasks.CacheableTask;
import org.gradle.api.tasks.Input;
import org.gradle.api.tasks.OutputFile;
import org.gradle.api.tasks.TaskAction;

// BuildOpt CUSTOM_TASK_CONTRACT_JAVA_V1 source=sha256:392e0197d9143304d8ce3c1a598aa552d62e5f2cc200a5766c5d06e516ca5083
@CacheableTask
public abstract class GeneratePilotManifest extends DefaultTask {
    @Input
    public abstract ListProperty<String> getEntries();

    @OutputFile
    public abstract RegularFileProperty getOutputFile();

    @TaskAction
    public void generate() throws IOException, NoSuchAlgorithmException {
        List<String> entries = new ArrayList<>(getEntries().get());
        Collections.sort(entries);
        MessageDigest digest = MessageDigest.getInstance("SHA-256");
        byte[] state = String.join("\n", entries).getBytes(StandardCharsets.UTF_8);
        for (int round = 0; round < 2_000_000; round++) {
            state = digest.digest(state);
        }
        entries.add("workDigest=" + HexFormat.of().formatHex(state));
        Path output = getOutputFile().get().getAsFile().toPath();
        Files.createDirectories(output.getParent());
        Files.write(output, entries, StandardCharsets.UTF_8);
    }
}
