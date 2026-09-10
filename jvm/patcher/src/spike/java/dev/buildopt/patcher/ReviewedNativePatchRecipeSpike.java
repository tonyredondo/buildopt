package dev.buildopt.patcher;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Arrays;
import java.util.List;

/** Focused exact-apply, drift, ambiguity, idempotency and defensive-copy cases. */
final class ReviewedNativePatchRecipeSpike {
    private ReviewedNativePatchRecipeSpike() {}

    static void assertConformance() throws Exception {
        assertElasticsearchMaterialization();
        byte[] source = "alpha\nbeta\n".getBytes(StandardCharsets.UTF_8);
        List<ReviewedNativePatchJavaRecipe.Edit> edits = List.of(
                new ReviewedNativePatchJavaRecipe.Edit(0, "marker\n"),
                new ReviewedNativePatchJavaRecipe.Edit(6, "relative\n"));
        byte[] expected = "marker\nalpha\nrelative\nbeta\n".getBytes(StandardCharsets.UTF_8);
        String preimage = PatchBundleVerifier.digestBytes(source);
        String postimage = PatchBundleVerifier.digestBytes(expected);
        ReviewedNativePatchJavaRecipe.Result first =
                ReviewedNativePatchJavaRecipe.applyForTest(
                        "Exact.java", source, preimage, postimage, edits);
        require(first.changed() && Arrays.equals(expected, first.postimage()),
                "reviewed native recipe exact output");
        ReviewedNativePatchJavaRecipe.Result repeated =
                ReviewedNativePatchJavaRecipe.applyForTest(
                        "Exact.java", first.postimage(), preimage, postimage, edits);
        require(!repeated.changed() && repeated.preimageDigest().equals(postimage),
                "reviewed native recipe idempotency");
        byte[] defensive = first.postimage();
        defensive[0] ^= 1;
        require(!Arrays.equals(defensive, first.postimage()),
                "reviewed native recipe defensive output");

        expectFailure(() -> ReviewedNativePatchJavaRecipe.applyForTest(
                "Exact.java", "drift\n".getBytes(StandardCharsets.UTF_8),
                preimage, postimage, edits));
        expectFailure(() -> ReviewedNativePatchJavaRecipe.applyForTest(
                "Exact.java", source, preimage, postimage,
                List.of(new ReviewedNativePatchJavaRecipe.Edit(3, "a"),
                        new ReviewedNativePatchJavaRecipe.Edit(3, "b"))));
        expectFailure(() -> ReviewedNativePatchJavaRecipe.applyForTest(
                "Exact.java", source, preimage, postimage,
                List.of(new ReviewedNativePatchJavaRecipe.Edit(100, "x"))));
    }

    static byte[] elasticsearchSource() throws Exception {
        try (var stream = ReviewedNativePatchRecipeSpike.class.getResourceAsStream(
                "/reviewed-native/ForbiddenPatternsTask.java.txt")) {
            require(stream != null, "Elasticsearch source fixture is available");
            byte[] source = stream.readAllBytes();
            require(PatchBundleVerifier.digestBytes(source).equals(
                            "sha256:61fe2eaa06ff463c2a49cea656b147889060855acfa094288ebb8b31567e11b6"),
                    "Elasticsearch fixture matches the qualified source");
            return source;
        }
    }

    private static void assertElasticsearchMaterialization() throws Exception {
        Path root = Files.createTempDirectory("buildopt-elasticsearch-recipe-");
        Path input = root.resolve("input.java");
        Path output = root.resolve("output.java");
        try {
            byte[] source = elasticsearchSource();
            String path = ReviewedNativePatchJavaRecipe.ELASTICSEARCH_FORBIDDEN_PATTERNS_PATH;
            expectFailure(() -> ReviewedNativePatchJavaRecipe.applyElasticsearchForbiddenPatterns(
                    "ForbiddenPatternsTask.java", source));
            byte[] drifted = source.clone();
            drifted[0] ^= 1;
            expectFailure(() -> ReviewedNativePatchJavaRecipe.applyElasticsearchForbiddenPatterns(path, drifted));
            var first = ReviewedNativePatchJavaRecipe.applyElasticsearchForbiddenPatterns(path, source);
            require(first.changed(), "Elasticsearch first application changes source");
            require(!ReviewedNativePatchJavaRecipe.applyElasticsearchForbiddenPatterns(
                    path, first.postimage()).changed(), "Elasticsearch exact application is idempotent");
            byte[] mutableCopy = first.postimage();
            mutableCopy[0] ^= 1;
            require(!Arrays.equals(mutableCopy, first.postimage()), "Elasticsearch result owns its bytes");
            Files.write(input, source);
            ReviewedNativePatchRecipeCli.main(new String[] {
                    "REVIEWED_ELASTICSEARCH_FORBIDDEN_PATTERNS_JAVA_V1",
                    input.toString(), output.toString()});
            require(PatchBundleVerifier.digestBytes(Files.readAllBytes(output)).equals(
                            "sha256:d6858f5ac43ad671496cf1e54e7af309cb98ebda4a9be53e61578df8baefa2d0"),
                    "Elasticsearch CLI produces the retained exact correction");
        } finally {
            Files.deleteIfExists(output);
            Files.deleteIfExists(input);
            Files.delete(root);
        }
    }

    private static void expectFailure(CheckedOperation operation) throws Exception {
        try {
            operation.run();
            throw new AssertionError("reviewed native recipe accepted invalid input");
        } catch (PatchFailure failure) {
            require(failure.status() == PatchFailure.Status.PROPOSED,
                    "reviewed native recipe failure state");
        }
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    @FunctionalInterface
    private interface CheckedOperation { void run() throws Exception; }
}
