package dev.buildopt.patcher;

import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardOpenOption;

/** Offline exact-file adapter used by the customer-side POC materialization flow. */
public final class CustomTaskContractRecipeCli {
    private CustomTaskContractRecipeCli() {}

    public static void main(String[] arguments) throws Exception {
        if (arguments.length != 2) {
            throw new IllegalArgumentException("usage: CustomTaskContractRecipeCli INPUT OUTPUT");
        }
        Path input = Path.of(arguments[0]).toAbsolutePath().normalize();
        Path output = Path.of(arguments[1]).toAbsolutePath().normalize();
        CustomTaskContractJavaRecipe.Result result =
                CustomTaskContractJavaRecipe.apply(
                        "buildSrc/src/main/java/dev/buildopt/pilot/GeneratePilotManifest.java",
                        Files.readAllBytes(input));
        if (!result.changed()) {
            throw new IllegalStateException("custom-task source is already qualified");
        }
        Files.write(output, result.postimage(), StandardOpenOption.CREATE_NEW);
        System.out.println("preimage=" + result.preimageDigest());
        System.out.println("postimage=" + result.postimageDigest());
    }
}
