package dev.buildopt.patcher;

import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardOpenOption;

/** Offline materializer for one exact reviewed native recipe. */
public final class ReviewedNativePatchRecipeCli {
    private ReviewedNativePatchRecipeCli() {}

    public static void main(String[] arguments) throws Exception {
        if (arguments.length != 3) {
            throw new IllegalArgumentException("usage: RECIPE_ID INPUT OUTPUT");
        }
        String recipeId = arguments[0];
        Path input = Path.of(arguments[1]).toAbsolutePath().normalize();
        Path output = Path.of(arguments[2]).toAbsolutePath().normalize();
        byte[] source = Files.readAllBytes(input);
        long started = System.nanoTime();
        ReviewedNativePatchJavaRecipe.Result result = apply(recipeId, source);
        long generationNanos = System.nanoTime() - started;
        if (!result.changed()) {
            throw new IllegalStateException("reviewed native source is already qualified");
        }
        long repeatStarted = System.nanoTime();
        ReviewedNativePatchJavaRecipe.Result repeated = apply(recipeId, result.postimage());
        long validationNanos = System.nanoTime() - repeatStarted;
        if (repeated.changed()) {
            throw new IllegalStateException("reviewed native recipe is not idempotent");
        }
        Files.write(output, result.postimage(), StandardOpenOption.CREATE_NEW);
        System.out.println("recipeId=" + recipeId);
        System.out.println("preimage=" + result.preimageDigest());
        System.out.println("postimage=" + result.postimageDigest());
        System.out.println("generationNanos=" + generationNanos);
        System.out.println("idempotencyValidationNanos=" + validationNanos);
    }

    private static ReviewedNativePatchJavaRecipe.Result apply(String recipeId, byte[] source)
            throws PatchFailure {
        if (ReviewedNativePatchJavaRecipe.RELATIVE_CACHEABILITY_RECIPE_ID.equals(recipeId)) {
            return ReviewedNativePatchJavaRecipe.applyRelativeCacheability(
                    "buildSrc/src/main/groovy/io/micronaut/build/internal/python/"
                            + "PythonVfsBytecodeCompile.java",
                    source);
        } else if (ReviewedNativePatchJavaRecipe.MARKER_ONLY_CACHEABILITY_RECIPE_ID
                .equals(recipeId)) {
            return ReviewedNativePatchJavaRecipe.applyMarkerOnlyCacheability(
                    "buildSrc/src/main/java/org/springframework/build/architecture/"
                            + "ArchitectureCheck.java",
                    source);
        } else if (ReviewedNativePatchJavaRecipe.ELASTICSEARCH_FORBIDDEN_PATTERNS_RECIPE_ID
                .equals(recipeId)) {
            return ReviewedNativePatchJavaRecipe.applyElasticsearchForbiddenPatterns(
                    ReviewedNativePatchJavaRecipe.ELASTICSEARCH_FORBIDDEN_PATTERNS_PATH, source);
        } else {
            throw new IllegalArgumentException("unknown reviewed native recipe");
        }
    }
}
