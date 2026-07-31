package dev.buildopt.patcher;

import java.nio.ByteBuffer;
import java.nio.charset.CharacterCodingException;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;

/** Exact Kotlin DSL recipe for reproducible Gradle archive tasks. */
public final class ArchiveReproducibilityRecipe {
    public static final String RECIPE_ID = "ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1";
    public static final String RECIPE_VERSION = "1.0";
    private static final int MAXIMUM_POSTIMAGE_BYTES = 1024 * 1024;
    private static final String IMPORT =
            "import org.gradle.api.tasks.bundling.AbstractArchiveTask\n";
    private static final String CONFIGURATION =
            "tasks.withType<AbstractArchiveTask>().configureEach {\n"
                    + "    isReproducibleFileOrder = true\n"
                    + "    isPreserveFileTimestamps = false\n"
                    + "}\n";
    private static final int GENERATED_OVERHEAD_BYTES =
            (IMPORT + "\n\n" + CONFIGURATION).getBytes(StandardCharsets.UTF_8).length;
    private static final int MAXIMUM_SOURCE_BYTES =
            MAXIMUM_POSTIMAGE_BYTES - GENERATED_OVERHEAD_BYTES;

    private ArchiveReproducibilityRecipe() {
    }

    /**
     * Produces one exact full-file replacement for a supported Kotlin DSL root build.
     *
     * @param relativePath must be exactly {@code build.gradle.kts}
     * @param source current exact preimage bytes
     * @return immutable preimage/postimage metadata
     * @throws PatchFailure when the source is ambiguous or outside the recipe scope
     */
    public static Result apply(String relativePath, byte[] source) throws PatchFailure {
        if (!"build.gradle.kts".equals(relativePath)
                || source == null
                || source.length == 0
                || source.length > MAXIMUM_SOURCE_BYTES + GENERATED_OVERHEAD_BYTES) {
            reject("archive recipe requires a bounded root build.gradle.kts");
        }
        byte[] input = source.clone();
        String text = decode(input);
        if (text.indexOf('\u0000') >= 0
                || text.indexOf('\r') >= 0
                || !text.endsWith("\n")) {
            reject("archive recipe requires NUL-free LF source ending in a newline");
        }
        if (isGeneratedOutput(text)) {
            return result(input, input, false);
        }
        if (source.length > MAXIMUM_SOURCE_BYTES) {
            reject("archive recipe source exceeds the bounded input limit");
        }
        if (text.startsWith("@file:")
                || text.contains("AbstractArchiveTask")
                || text.contains("isReproducibleFileOrder")
                || text.contains("isPreserveFileTimestamps")) {
            reject("archive recipe found an existing or ambiguous archive configuration");
        }
        String exactOutput = IMPORT + "\n" + text + "\n" + CONFIGURATION;
        return result(input, exactOutput.getBytes(StandardCharsets.UTF_8), true);
    }

    private static boolean isGeneratedOutput(String text) {
        String prefix = IMPORT + "\n";
        String suffix = "\n" + CONFIGURATION;
        if (!text.startsWith(prefix)
                || !text.endsWith(suffix)
                || text.length() <= prefix.length() + suffix.length()) {
            return false;
        }
        String original = text.substring(prefix.length(), text.length() - suffix.length());
        return original.endsWith("\n")
                && !original.startsWith("@file:")
                && !original.contains("AbstractArchiveTask")
                && !original.contains("isReproducibleFileOrder")
                && !original.contains("isPreserveFileTimestamps");
    }

    private static String decode(byte[] source) throws PatchFailure {
        try {
            return StandardCharsets.UTF_8.newDecoder()
                    .onMalformedInput(CodingErrorAction.REPORT)
                    .onUnmappableCharacter(CodingErrorAction.REPORT)
                    .decode(ByteBuffer.wrap(source))
                    .toString();
        } catch (CharacterCodingException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "archive recipe source is not UTF-8",
                    exception);
        }
    }

    private static Result result(byte[] preimage, byte[] postimage, boolean changed)
            throws PatchFailure {
        try {
            return new Result(
                    changed,
                    PatchBundleVerifier.digestBytes(preimage),
                    PatchBundleVerifier.digestBytes(postimage),
                    postimage);
        } catch (GeneralSecurityException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "archive recipe cannot calculate content digests",
                    exception);
        }
    }

    private static void reject(String message) throws PatchFailure {
        throw new PatchFailure(PatchFailure.Status.PROPOSED, message);
    }

    /** Immutable exact replacement generated by this recipe version. */
    public static final class Result {
        private final boolean changed;
        private final String preimageDigest;
        private final String postimageDigest;
        private final byte[] postimage;

        private Result(
                boolean changed,
                String preimageDigest,
                String postimageDigest,
                byte[] postimage) {
            this.changed = changed;
            this.preimageDigest = preimageDigest;
            this.postimageDigest = postimageDigest;
            this.postimage = postimage.clone();
        }

        public boolean changed() {
            return changed;
        }

        public String preimageDigest() {
            return preimageDigest;
        }

        public String postimageDigest() {
            return postimageDigest;
        }

        public byte[] postimage() {
            return postimage.clone();
        }
    }
}
