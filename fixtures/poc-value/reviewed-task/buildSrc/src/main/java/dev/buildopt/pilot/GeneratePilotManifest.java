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
import org.gradle.api.tasks.Internal;
import org.gradle.api.tasks.TaskAction;

public abstract class GeneratePilotManifest extends DefaultTask {
    @Internal
    public abstract ListProperty<String> getEntries();

    @Internal
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
