package dev.buildopt.patcher;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.List;

/** Focused exact-apply, drift, ambiguity, idempotency and defensive-copy cases. */
final class ReviewedNativePatchRecipeSpike {
    private ReviewedNativePatchRecipeSpike() {}

    static void assertConformance() throws Exception {
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
