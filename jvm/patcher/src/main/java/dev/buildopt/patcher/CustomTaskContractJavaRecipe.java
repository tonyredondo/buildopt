package dev.buildopt.patcher;

import java.nio.ByteBuffer;
import java.nio.charset.CharacterCodingException;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;

/** Exact Java recipe for the first reviewed custom-task cache contract. */
public final class CustomTaskContractJavaRecipe {
    public static final String RECIPE_ID = "CUSTOM_TASK_CONTRACT_JAVA_V1";
    public static final String RECIPE_VERSION = "1.0";
    private static final int MAXIMUM_SOURCE_BYTES = 1024 * 1024;
    private static final String PATH =
            "buildSrc/src/main/java/dev/buildopt/pilot/GeneratePilotManifest.java";
    private static final String INTERNAL_IMPORT = "import org.gradle.api.tasks.Internal;\n";
    private static final String CONTRACT_IMPORTS =
            "import org.gradle.api.tasks.CacheableTask;\n"
                    + "import org.gradle.api.tasks.Input;\n"
                    + "import org.gradle.api.tasks.OutputFile;\n";
    private static final String CLASS =
            "public abstract class GeneratePilotManifest extends DefaultTask {";
    private static final String INPUT =
            "    @Internal\n"
                    + "    public abstract ListProperty<String> getEntries();";
    private static final String OUTPUT =
            "    @Internal\n"
                    + "    public abstract RegularFileProperty getOutputFile();";
    private static final String QUALIFIED_INPUT =
            "    @Input\n"
                    + "    public abstract ListProperty<String> getEntries();";
    private static final String QUALIFIED_OUTPUT =
            "    @OutputFile\n"
                    + "    public abstract RegularFileProperty getOutputFile();";
    private static final String MARKER_PREFIX =
            "// BuildOpt CUSTOM_TASK_CONTRACT_JAVA_V1 source=";

    private CustomTaskContractJavaRecipe() {}

    /** Produces an exact full-file replacement or rejects an ambiguous source. */
    public static Result apply(String relativePath, byte[] source) throws PatchFailure {
        if (!PATH.equals(relativePath) || source == null || source.length == 0
                || source.length > MAXIMUM_SOURCE_BYTES) {
            reject("custom-task recipe requires its bounded reviewed Java source path");
        }
        byte[] input = source.clone();
        String text = decode(input);
        if (text.indexOf('\0') >= 0 || text.indexOf('\r') >= 0 || !text.endsWith("\n")) {
            reject("custom-task recipe requires NUL-free LF source ending in a newline");
        }
        if (text.contains(MARKER_PREFIX)) {
            validateGenerated(text);
            return result(input, input, false);
        }
        validateBaseline(text);
        String digest = digest(input);
        String postimage = text.replace(INTERNAL_IMPORT, CONTRACT_IMPORTS)
                .replace(CLASS, MARKER_PREFIX + digest + "\n@CacheableTask\n" + CLASS)
                .replace(INPUT, QUALIFIED_INPUT)
                .replace(OUTPUT, QUALIFIED_OUTPUT);
        validateGenerated(postimage);
        return result(input, postimage.getBytes(StandardCharsets.UTF_8), true);
    }

    private static void validateBaseline(String text) throws PatchFailure {
        if (count(text, INTERNAL_IMPORT) != 1 || count(text, CLASS) != 1
                || count(text, INPUT) != 1 || count(text, OUTPUT) != 1
                || text.contains("@CacheableTask") || text.contains("@Input")
                || text.contains("@OutputFile") || text.contains(CONTRACT_IMPORTS)) {
            reject("custom-task recipe found an unsupported or ambiguous Java contract");
        }
    }

    private static void validateGenerated(String text) throws PatchFailure {
        int marker = text.indexOf(MARKER_PREFIX);
        int digestStart = marker + MARKER_PREFIX.length();
        int digestEnd = digestStart + 71;
        if (marker < 0 || digestEnd >= text.length()
                || !text.substring(digestStart, digestEnd).matches("sha256:[0-9a-f]{64}")
                || text.charAt(digestEnd) != '\n'
                || count(text, MARKER_PREFIX) != 1 || count(text, CONTRACT_IMPORTS) != 1
                || count(text, "@CacheableTask") != 1 || count(text, QUALIFIED_INPUT) != 1
                || count(text, QUALIFIED_OUTPUT) != 1 || text.contains(INTERNAL_IMPORT)) {
            reject("custom-task recipe generated marker or contract is inconsistent");
        }
        String original = text.replace(CONTRACT_IMPORTS, INTERNAL_IMPORT)
                .replace(MARKER_PREFIX + text.substring(digestStart, digestEnd)
                        + "\n@CacheableTask\n" + CLASS, CLASS)
                .replace(QUALIFIED_INPUT, INPUT)
                .replace(QUALIFIED_OUTPUT, OUTPUT);
        validateBaseline(original);
        if (!digest(original.getBytes(StandardCharsets.UTF_8))
                .equals(text.substring(digestStart, digestEnd))) {
            reject("custom-task recipe source digest does not match the reversible envelope");
        }
    }

    private static int count(String text, String value) {
        int result = 0;
        int offset = 0;
        while ((offset = text.indexOf(value, offset)) >= 0) {
            result++;
            offset += value.length();
        }
        return result;
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
                    "custom-task recipe source is not UTF-8",
                    exception);
        }
    }

    private static String digest(byte[] source) throws PatchFailure {
        try {
            return PatchBundleVerifier.digestBytes(source);
        } catch (GeneralSecurityException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "custom-task recipe cannot calculate content digest",
                    exception);
        }
    }

    private static Result result(byte[] preimage, byte[] postimage, boolean changed)
            throws PatchFailure {
        return new Result(changed, digest(preimage), digest(postimage), postimage);
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
