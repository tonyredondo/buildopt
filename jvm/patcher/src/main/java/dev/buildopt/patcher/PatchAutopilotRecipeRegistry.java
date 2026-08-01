package dev.buildopt.patcher;

import java.util.List;
import java.util.Optional;

/** Closed, versioned catalog of recipes accepted by Patch Autopilot. */
public final class PatchAutopilotRecipeRegistry {
    private static final List<Definition> DEFINITIONS = List.of(
            new Definition(
                    ArchiveReproducibilityRecipe.RECIPE_ID,
                    ArchiveReproducibilityRecipe.RECIPE_VERSION,
                    "ROOT_BUILD_GRADLE_KTS",
                    Risk.LOW,
                    "ARCHIVE_CONTENTS_V1",
                    Inverse.EXACT_MODIFY_ONLY,
                    false),
            new Definition(
                    CustomTaskContractJavaRecipe.RECIPE_ID,
                    CustomTaskContractJavaRecipe.RECIPE_VERSION,
                    "REVIEWED_BUILD_SRC_JAVA_ADAPTER",
                    Risk.LOW,
                    "EXACT_BYTES",
                    Inverse.UNAVAILABLE,
                    true));

    private PatchAutopilotRecipeRegistry() {
    }

    /** Returns the immutable allowlist in deterministic declaration order. */
    public static List<Definition> definitions() {
        return DEFINITIONS;
    }

    /** Resolves one exact recipe/version pair without negotiating a fallback. */
    public static Optional<Definition> find(String id, String version) {
        if (id == null || version == null) {
            return Optional.empty();
        }
        return DEFINITIONS.stream()
                .filter(definition -> definition.id().equals(id)
                        && definition.version().equals(version))
                .findFirst();
    }

    /** Risk level permitted for draft-only automatic proposals. */
    public enum Risk {
        LOW
    }

    /** Exact inverse capability available after a patch is merged. */
    public enum Inverse {
        EXACT_MODIFY_ONLY,
        UNAVAILABLE
    }

    /** Immutable closed metadata for one recipe version. */
    public record Definition(
            String id,
            String version,
            String applicability,
            Risk risk,
            String validationAdapter,
            Inverse inverse,
            boolean reviewedAdapterRequired) {
    }
}
