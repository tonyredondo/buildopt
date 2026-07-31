package dev.buildopt.patcher;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;

/** Focused golden, negative, idempotency, and defensive-copy cases for C4-004. */
final class CustomTaskContractRecipeSpike {
    private static final String PATH =
            "buildSrc/src/main/java/dev/buildopt/pilot/GeneratePilotManifest.java";
    private static final String SOURCE =
            "package dev.buildopt.pilot;\n\n"
                    + "import org.gradle.api.DefaultTask;\n"
                    + "import org.gradle.api.file.RegularFileProperty;\n"
                    + "import org.gradle.api.provider.ListProperty;\n"
                    + "import org.gradle.api.tasks.Internal;\n"
                    + "import org.gradle.api.tasks.TaskAction;\n\n"
                    + "public abstract class GeneratePilotManifest extends DefaultTask {\n"
                    + "    @Internal\n"
                    + "    public abstract ListProperty<String> getEntries();\n\n"
                    + "    @Internal\n"
                    + "    public abstract RegularFileProperty getOutputFile();\n\n"
                    + "    @TaskAction\n"
                    + "    public void generate() {}\n"
                    + "}\n";

    private CustomTaskContractRecipeSpike() {}

    static void assertConformance() throws Exception {
        byte[] source = SOURCE.getBytes(StandardCharsets.UTF_8);
        CustomTaskContractJavaRecipe.Result first =
                CustomTaskContractJavaRecipe.apply(PATH, source);
        require(first.changed(), "custom-task recipe changes baseline");
        String postimage = new String(first.postimage(), StandardCharsets.UTF_8);
        require(postimage.contains("@CacheableTask")
                        && postimage.contains("@Input")
                        && postimage.contains("@OutputFile")
                        && postimage.contains("source=sha256:"),
                "custom-task recipe exact contract");
        CustomTaskContractJavaRecipe.Result repeated =
                CustomTaskContractJavaRecipe.apply(PATH, first.postimage());
        require(!repeated.changed()
                        && repeated.preimageDigest().equals(repeated.postimageDigest())
                        && Arrays.equals(first.postimage(), repeated.postimage()),
                "custom-task recipe idempotency");
        byte[] defensive = first.postimage();
        defensive[0] ^= 1;
        require(!Arrays.equals(defensive, first.postimage()),
                "custom-task recipe defensive bytes");

        expectFailure(() -> CustomTaskContractJavaRecipe.apply("build.gradle", source));
        expectFailure(() -> CustomTaskContractJavaRecipe.apply(
                PATH, SOURCE.replace("@Internal", "@Input")
                        .getBytes(StandardCharsets.UTF_8)));
        byte[] tampered = first.postimage();
        tampered[tampered.length - 3] ^= 1;
        expectFailure(() -> CustomTaskContractJavaRecipe.apply(PATH, tampered));
        expectFailure(() -> CustomTaskContractJavaRecipe.apply(
                PATH, new byte[] {(byte) 0xc3, (byte) 0x28}));
    }

    private static void expectFailure(CheckedOperation operation) throws Exception {
        try {
            operation.run();
            throw new AssertionError("custom-task recipe accepted invalid input");
        } catch (PatchFailure failure) {
            require(failure.status() == PatchFailure.Status.PROPOSED,
                    "custom-task recipe failure state");
        }
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    @FunctionalInterface
    private interface CheckedOperation {
        void run() throws Exception;
    }
}
