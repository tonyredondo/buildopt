package dev.buildopt.patcher;

import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;

/** Digest-bound Java source recipes qualified by the reviewed native patch portfolio. */
public final class ReviewedNativePatchJavaRecipe {
    public static final String RELATIVE_CACHEABILITY_RECIPE_ID =
            "REVIEWED_RELATIVE_CACHEABILITY_JAVA_V1";
    public static final String MARKER_ONLY_CACHEABILITY_RECIPE_ID =
            "REVIEWED_MARKER_ONLY_CACHEABILITY_JAVA_V1";
    public static final String RECIPE_VERSION = "1.0";
    private static final int MAXIMUM_SOURCE_BYTES = 1024 * 1024;
    private static final Definition MICRONAUT_PYTHON_VFS = new Definition(
            "buildSrc/src/main/groovy/io/micronaut/build/internal/python/"
                    + "PythonVfsBytecodeCompile.java",
            "sha256:2751e8434d1cdd6be5052abe4d3205662e4669d78fcbc9168e3c73a261c0adca",
            "sha256:2cfc774746985253205acb2f360df84c70a9f4ba985d2ef1f32e8ecbcfa464ed",
            List.of(
                    new Edit(1257, "@org.gradle.api.tasks.CacheableTask\n"),
                    new Edit(1460, "    @org.gradle.api.tasks.PathSensitive("
                            + "org.gradle.api.tasks.PathSensitivity.RELATIVE)\n")));
    private static final Definition SPRING_ARCHITECTURE_CHECK = new Definition(
            "buildSrc/src/main/java/org/springframework/build/architecture/"
                    + "ArchitectureCheck.java",
            "sha256:22b20c6a01db4cc0ca93f871e8add20cbb105b8c20d9e6e8b3c55d705a3efd04",
            "sha256:dd75bc28223141ac121f6377f7ba95832fd250739861196f60120264dd4c2d1d",
            List.of(new Edit(2452, "@org.gradle.api.tasks.CacheableTask\n")));

    private ReviewedNativePatchJavaRecipe() {}

    /** Applies the reviewed relative-normalization correction to its exact source revision. */
    public static Result applyRelativeCacheability(String relativePath, byte[] source)
            throws PatchFailure {
        return apply(MICRONAUT_PYTHON_VFS, relativePath, source);
    }

    /** Applies the reviewed marker-only correction to its exact source revision. */
    public static Result applyMarkerOnlyCacheability(String relativePath, byte[] source)
            throws PatchFailure {
        return apply(SPRING_ARCHITECTURE_CHECK, relativePath, source);
    }

    static Result applyForTest(
            String path,
            byte[] source,
            String preimageDigest,
            String postimageDigest,
            List<Edit> edits) throws PatchFailure {
        return apply(new Definition(path, preimageDigest, postimageDigest, edits), path, source);
    }

    private static Result apply(Definition definition, String relativePath, byte[] source)
            throws PatchFailure {
        if (!definition.path().equals(relativePath) || source == null || source.length == 0
                || source.length > MAXIMUM_SOURCE_BYTES) {
            reject("reviewed native recipe requires its bounded exact Java source path");
        }
        byte[] input = source.clone();
        String inputDigest = digest(input);
        if (definition.postimageDigest().equals(inputDigest)) {
            return new Result(false, inputDigest, inputDigest, input);
        }
        if (!definition.preimageDigest().equals(inputDigest)) {
            reject("reviewed native recipe source digest does not match its qualified preimage");
        }
        List<Edit> descending = new ArrayList<>(definition.edits());
        descending.sort(Comparator.comparingInt(Edit::offset).reversed());
        byte[] output = input;
        int previousOffset = input.length + 1;
        for (Edit edit : descending) {
            if (edit.offset() < 0 || edit.offset() > input.length
                    || edit.offset() >= previousOffset) {
                reject("reviewed native recipe contains invalid or ambiguous edit offsets");
            }
            byte[] insertion = edit.insertedText().getBytes(StandardCharsets.UTF_8);
            byte[] next = new byte[output.length + insertion.length];
            System.arraycopy(output, 0, next, 0, edit.offset());
            System.arraycopy(insertion, 0, next, edit.offset(), insertion.length);
            System.arraycopy(output, edit.offset(), next, edit.offset() + insertion.length,
                    output.length - edit.offset());
            output = next;
            previousOffset = edit.offset();
        }
        String outputDigest = digest(output);
        if (!definition.postimageDigest().equals(outputDigest)) {
            reject("reviewed native recipe output digest does not match its qualified postimage");
        }
        return new Result(true, inputDigest, outputDigest, output);
    }

    private static String digest(byte[] source) throws PatchFailure {
        try {
            return PatchBundleVerifier.digestBytes(source);
        } catch (GeneralSecurityException exception) {
            throw new PatchFailure(PatchFailure.Status.PROPOSED,
                    "reviewed native recipe cannot calculate content digest", exception);
        }
    }

    private static void reject(String message) throws PatchFailure {
        throw new PatchFailure(PatchFailure.Status.PROPOSED, message);
    }

    record Edit(int offset, String insertedText) {}

    private record Definition(
            String path, String preimageDigest, String postimageDigest, List<Edit> edits) {}

    /** Immutable exact replacement generated by one qualified recipe. */
    public static final class Result {
        private final boolean changed;
        private final String preimageDigest;
        private final String postimageDigest;
        private final byte[] postimage;

        private Result(boolean changed, String preimageDigest, String postimageDigest,
                byte[] postimage) {
            this.changed = changed;
            this.preimageDigest = preimageDigest;
            this.postimageDigest = postimageDigest;
            this.postimage = postimage.clone();
        }

        public boolean changed() { return changed; }
        public String preimageDigest() { return preimageDigest; }
        public String postimageDigest() { return postimageDigest; }
        public byte[] postimage() { return postimage.clone(); }
    }
}
