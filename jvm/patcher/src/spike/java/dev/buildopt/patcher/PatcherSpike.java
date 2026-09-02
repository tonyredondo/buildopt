package dev.buildopt.patcher;

import static java.nio.file.LinkOption.NOFOLLOW_LINKS;
import static java.nio.file.StandardCopyOption.ATOMIC_MOVE;
import static java.nio.file.StandardCopyOption.REPLACE_EXISTING;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardOpenOption;
import java.nio.file.attribute.BasicFileAttributes;
import java.nio.file.attribute.PosixFilePermissions;
import java.security.GeneralSecurityException;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.atomic.AtomicInteger;

import dev.buildopt.patcher.PatchBundleApplier.DraftPullRequest;
import dev.buildopt.patcher.PatchBundleApplier.DraftPullRequests;
import dev.buildopt.patcher.PatchBundleApplier.Fault;
import dev.buildopt.patcher.PatchBundleApplier.Identity;
import dev.buildopt.patcher.PatchBundleApplier.Outcome;
import dev.buildopt.patcher.PatchBundleVerifier.VerifiedPatchBundle;

/**
 * Executable SPK-004 acceptance matrix over real temporary Git repositories.
 */
public final class PatcherSpike {
    private static final Instant NOW = Instant.parse("2026-07-29T12:00:00Z");
    private static final String REPOSITORY_ID = "tenant-7/repo-42";
    private static final String KEY_ID = "patch-signing-spike-2026-q3";
    private static final AtomicInteger NEXT_BUNDLE = new AtomicInteger(1);

    private final Path repositoryRoot;
    private final Path contractBlobs;
    private final Path temporaryRoot;
    private final KeyPair signingKey;

    private PatcherSpike(Path repositoryRoot, Path temporaryRoot)
            throws GeneralSecurityException {
        this.repositoryRoot = repositoryRoot;
        this.contractBlobs = repositoryRoot.resolve(
                "contracts/jsonschema/testdata/patch-bundle.v1/blobs");
        this.temporaryRoot = temporaryRoot;
        this.signingKey = KeyPairGenerator.getInstance("Ed25519").generateKeyPair();
    }

    /**
     * Runs every machine-readable case and writes a bounded JSON report.
     *
     * @param arguments repository root and report path
     * @throws Exception when any expected outcome is not reproduced
     */
    public static void main(String[] arguments) throws Exception {
        if (arguments.length != 2) {
            throw new IllegalArgumentException(
                    "usage: PatcherSpike <repository-root> <report>");
        }
        Path repositoryRoot = Path.of(arguments[0]).toRealPath(NOFOLLOW_LINKS);
        Path report = Path.of(arguments[1]).toAbsolutePath().normalize();
        Path temporaryRoot = Files.createTempDirectory("buildopt-patcher-spike-");
        try {
            PatcherSpike spike = new PatcherSpike(repositoryRoot, temporaryRoot);
            spike.assertRecipeRegistry();
            spike.assertDuplicateKeysRejected();
            spike.assertSignerRejectsIncompleteManifest();
            spike.assertArchiveRecipeSafety();
            spike.assertExpandedRecipeSafety();
            CustomTaskContractRecipeSpike.assertConformance();
            ReviewedNativePatchRecipeSpike.assertConformance();
            PatchCandidateValidationSpike.assertConformance();
            FullRelevantValidationSpike.assertConformance();
            PostMergePatchMonitorSpike.assertConformance();
            spike.assertExactPostMergeRevert();
            spike.assertExpandedRecipeReverts();
            List<CaseDefinition> definitions = spike.readPlan();
            List<CaseResult> results = new ArrayList<>();
            for (CaseDefinition definition : definitions) {
                String actual = spike.runCase(definition.id());
                if (!actual.equals(definition.expected())) {
                    throw new AssertionError(
                            definition.id() + " returned " + actual
                                    + ", expected " + definition.expected());
                }
                results.add(new CaseResult(
                        definition.id(),
                        definition.expected(),
                        actual));
                System.out.println(
                        "SPK-004 case OK: " + definition.id() + " -> " + actual);
            }
            spike.writeReport(report, results);
            System.out.println(
                    "SPK-004 DONE: 15/15 real-Git parser/applier cases passed");
        } finally {
            deleteRecursively(temporaryRoot);
        }
    }

    private void assertRecipeRegistry() {
        List<PatchAutopilotRecipeRegistry.Definition> definitions =
                PatchAutopilotRecipeRegistry.definitions();
        require(definitions.size() == 6, "recipe registry size");
        PatchAutopilotRecipeRegistry.Definition archive =
                PatchAutopilotRecipeRegistry.find(
                        ArchiveReproducibilityRecipe.RECIPE_ID,
                        ArchiveReproducibilityRecipe.RECIPE_VERSION)
                        .orElseThrow();
        require(archive.risk() == PatchAutopilotRecipeRegistry.Risk.LOW
                        && archive.inverse()
                                == PatchAutopilotRecipeRegistry.Inverse.EXACT_MODIFY_ONLY
                        && "ARCHIVE_CONTENTS_V1".equals(archive.validationAdapter())
                        && !archive.reviewedAdapterRequired(),
                "archive registry metadata");
        PatchAutopilotRecipeRegistry.Definition custom =
                PatchAutopilotRecipeRegistry.find(
                        CustomTaskContractJavaRecipe.RECIPE_ID,
                        CustomTaskContractJavaRecipe.RECIPE_VERSION)
                        .orElseThrow();
        require(custom.reviewedAdapterRequired()
                        && custom.inverse()
                                == PatchAutopilotRecipeRegistry.Inverse.EXACT_MODIFY_ONLY
                        && "EXACT_BYTES".equals(custom.validationAdapter()),
                "custom-task registry metadata");
        for (String recipeId : List.of(
                ReviewedNativePatchJavaRecipe.RELATIVE_CACHEABILITY_RECIPE_ID,
                ReviewedNativePatchJavaRecipe.MARKER_ONLY_CACHEABILITY_RECIPE_ID)) {
            PatchAutopilotRecipeRegistry.Definition reviewed =
                    PatchAutopilotRecipeRegistry.find(
                            recipeId, ReviewedNativePatchJavaRecipe.RECIPE_VERSION)
                            .orElseThrow();
            require(reviewed.reviewedAdapterRequired()
                            && "DIGEST_BOUND_REVIEWED_JAVA_SOURCE".equals(
                                    reviewed.applicability())
                            && "EXACT_BYTES".equals(reviewed.validationAdapter())
                            && reviewed.inverse()
                                    == PatchAutopilotRecipeRegistry.Inverse.EXACT_MODIFY_ONLY,
                    "reviewed native registry metadata");
        }
        PatchAutopilotRecipeRegistry.Definition groovy =
                PatchAutopilotRecipeRegistry.find(
                        ArchiveReproducibilityGroovyDslRecipe.RECIPE_ID,
                        ArchiveReproducibilityGroovyDslRecipe.RECIPE_VERSION)
                        .orElseThrow();
        require("ROOT_BUILD_GRADLE".equals(groovy.applicability())
                        && "ARCHIVE_CONTENTS_V1".equals(groovy.validationAdapter())
                        && groovy.inverse()
                                == PatchAutopilotRecipeRegistry.Inverse.EXACT_MODIFY_ONLY,
                "Groovy archive registry metadata");
        PatchAutopilotRecipeRegistry.Definition buildCache =
                PatchAutopilotRecipeRegistry.find(
                        GradleBuildCachePropertiesRecipe.RECIPE_ID,
                        GradleBuildCachePropertiesRecipe.RECIPE_VERSION)
                        .orElseThrow();
        require("EXISTING_ROOT_GRADLE_PROPERTIES_WITHOUT_CACHING_KEY".equals(
                        buildCache.applicability())
                        && "ARCHIVE_CONTENTS_V1".equals(buildCache.validationAdapter())
                        && buildCache.inverse()
                                == PatchAutopilotRecipeRegistry.Inverse.EXACT_MODIFY_ONLY,
                "build-cache registry metadata");
        require(PatchAutopilotRecipeRegistry.find(archive.id(), "2.0").isEmpty()
                        && PatchAutopilotRecipeRegistry.find("UNKNOWN", "1.0").isEmpty(),
                "registry rejects fallback and unknown recipes");
        try {
            definitions.clear();
            throw new IllegalStateException("recipe registry is mutable");
        } catch (UnsupportedOperationException expected) {
            // Immutable catalog is part of the trust boundary.
        }
    }

    private List<CaseDefinition> readPlan() throws IOException {
        Map<String, Object> plan = object(StrictJson.parse(
                repositoryRoot.resolve("specs/patch-bundle-v1.json"),
                1024L * 1024L));
        if (!"buildopt.specs/patch-bundle-application/v1".equals(
                plan.get("schemaVersion"))) {
            throw new AssertionError("unexpected PatchBundle plan version");
        }
        List<Object> rawCases = array(plan.get("cases"));
        if (rawCases.size() != 15) {
            throw new AssertionError("PatchBundle plan must contain 15 cases");
        }
        List<CaseDefinition> definitions = new ArrayList<>();
        for (Object rawCase : rawCases) {
            Map<String, Object> testCase = object(rawCase);
            definitions.add(new CaseDefinition(
                    string(testCase, "id"),
                    string(testCase, "expected")));
        }
        return definitions;
    }

    private String runCase(String id) throws Exception {
        return switch (id) {
            case "apply-archive-reproducibility-recipe" -> applyArchive();
            case "apply-custom-task-contract-recipe" -> applyCustomTask();
            case "exact-repeat-is-idempotent" -> exactRepeat();
            case "divergent-preimage-rejected" -> divergentPreimage();
            case "base-tree-mismatch-rejected" -> baseTreeMismatch();
            case "expired-or-untrusted-bundle-rejected" -> expiredOrUntrusted();
            case "traversal-or-git-path-rejected" -> traversalOrGitPath();
            case "symlink-target-rejected" -> symlinkTarget();
            case "symlink-parent-rejected" -> symlinkParent();
            case "submodule-or-nested-repository-rejected" -> submoduleBoundary();
            case "postimage-mismatch-rolls-back-staging" -> postimageMismatch();
            case "branch-without-pr-is-recovered" -> branchWithoutPr();
            case "existing-branch-with-different-marker-conflicts" ->
                    existingBranchConflict();
            case "existing-matching-pr-replays" -> existingPrReplay();
            case "interrupted-staging-leaves-customer-checkout-unchanged" ->
                    interruptedStaging();
            default -> throw new AssertionError("unimplemented plan case " + id);
        };
    }

    private String applyArchive() throws Exception {
        try (Fixture fixture = fixture("apply-archive")) {
            fixture.installExplosiveCheckoutMechanisms();
            Snapshot before = fixture.snapshot();
            ArchiveReproducibilityRecipe.Result expected =
                    ArchiveReproducibilityRecipe.apply(
                            "build.gradle.kts",
                            Files.readAllBytes(
                                    fixture.repository.resolve("build.gradle.kts")));
            BundleFile bundle = archiveBundle(fixture, "archive-apply").write();
            PatchBundleApplier.Result result = apply(
                    fixture,
                    verify(bundle, signingKey),
                    Fault.NONE);
            require(result.outcome() == Outcome.DRAFT_PR_CREATED, "archive draft PR");
            require(
                    fixture.show(result.branch(), "build.gradle.kts")
                            .equals(new String(expected.postimage(), StandardCharsets.UTF_8)),
                    "archive exact branch bytes");
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            fixture.assertNoUnexpectedExecution();
            return "APPLIED_STAGED";
        }
    }

    private String applyCustomTask() throws Exception {
        try (Fixture fixture = fixture("apply-custom-task")) {
            Snapshot before = fixture.snapshot();
            BundleFile bundle = customTaskBundle(fixture, "task-apply").write();
            PatchBundleApplier.Result result = apply(
                    fixture,
                    verify(bundle, signingKey),
                    Fault.NONE);
            require(result.outcome() == Outcome.DRAFT_PR_CREATED, "custom-task draft PR");
            require(
                    fixture.show(
                            result.branch(),
                            "buildSrc/src/main/java/com/example/build/BundleFrontend.java")
                            .equals(new String(
                                    Files.readAllBytes(
                                            contractBlobs.resolve("custom-task.java")),
                                    StandardCharsets.UTF_8)),
                    "custom-task exact branch bytes");
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "APPLIED_STAGED";
        }
    }

    private String exactRepeat() throws Exception {
        try (Fixture fixture = fixture("exact-repeat")) {
            BundleFile bundle = archiveBundle(fixture, "archive-repeat").write();
            VerifiedPatchBundle verified = verify(bundle, signingKey);
            PatchBundleApplier.Result first = apply(fixture, verified, Fault.NONE);
            PatchBundleApplier.Result second = apply(fixture, verified, Fault.NONE);
            require(first.headCommit().equals(second.headCommit()), "repeat commit identity");
            require(second.outcome() == Outcome.EXISTING_DRAFT_PR, "repeat PR outcome");
            fixture.assertOnlyCustomerWorktree();
            return "EXISTING_DRAFT_PR";
        }
    }

    private String divergentPreimage() throws Exception {
        try (Fixture fixture = fixture("divergent-preimage")) {
            Snapshot before = fixture.snapshot();
            BundleBuilder builder = archiveBundle(fixture, "archive-divergent");
            builder.operations.get(0).preimage =
                    "sha256:" + "0".repeat(64);
            BundleFile bundle = builder.write();
            expectFailure(
                    PatchFailure.Status.PROPOSED,
                    () -> apply(fixture, verify(bundle, signingKey), Fault.NONE));
            fixture.assertNoActionBranch(builder.actionId);
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "PROPOSED";
        }
    }

    private String baseTreeMismatch() throws Exception {
        try (Fixture fixture = fixture("base-tree-mismatch")) {
            Snapshot before = fixture.snapshot();
            BundleBuilder builder = archiveBundle(fixture, "archive-tree-mismatch");
            builder.baseTree = "a".repeat(40);
            BundleFile bundle = builder.write();
            expectFailure(
                    PatchFailure.Status.PROPOSED,
                    () -> apply(fixture, verify(bundle, signingKey), Fault.NONE));
            fixture.assertNoActionBranch(builder.actionId);

            BundleBuilder sourceState = archiveBundle(
                    fixture,
                    "archive-source-state-mismatch");
            sourceState.sourceStateDigest = "sha256:" + "b".repeat(64);
            expectFailure(
                    PatchFailure.Status.PROPOSED,
                    () -> apply(
                            fixture,
                            verify(sourceState.write(), signingKey),
                            Fault.NONE));
            fixture.assertNoActionBranch(sourceState.actionId);

            BundleBuilder unsupportedDigest = archiveBundle(
                    fixture,
                    "archive-source-state-hmac");
            unsupportedDigest.sourceStateDigest =
                    "hmac-sha256:" + "c".repeat(64);
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> verify(unsupportedDigest.write(), signingKey));
            fixture.assertNoActionBranch(unsupportedDigest.actionId);
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "PROPOSED";
        }
    }

    private String expiredOrUntrusted() throws Exception {
        try (Fixture fixture = fixture("authority")) {
            BundleBuilder expiredBuilder = archiveBundle(fixture, "archive-expired");
            expiredBuilder.createdAt = NOW.minus(2, ChronoUnit.HOURS);
            expiredBuilder.expiresAt = NOW.minus(1, ChronoUnit.HOURS);
            BundleFile expired = expiredBuilder.write();
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> verify(expired, signingKey));

            BundleFile untrusted = archiveBundle(
                    fixture,
                    "archive-untrusted").write();
            KeyPair otherKey = KeyPairGenerator.getInstance("Ed25519").generateKeyPair();
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> verify(untrusted, otherKey));

            BundleFile unknown = archiveBundle(
                    fixture,
                    "archive-unknown-field").write();
            String unknownJson = Files.readString(
                    unknown.manifest(),
                    StandardCharsets.UTF_8);
            Files.writeString(
                    unknown.manifest(),
                    "{\"command\":\"forbidden\","
                            + unknownJson.substring(1),
                    StandardCharsets.UTF_8);
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> verify(unknown, signingKey));

            BundleFile tamperedBlob = archiveBundle(
                    fixture,
                    "archive-tampered-blob").write();
            Files.writeString(
                    tamperedBlob.root().resolve("blobs/archive-build.gradle.kts"),
                    "x",
                    StandardCharsets.UTF_8,
                    StandardOpenOption.APPEND);
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> verify(tamperedBlob, signingKey));
            fixture.assertNoActionBranch(expiredBuilder.actionId);
            fixture.assertOnlyCustomerWorktree();
            return "REJECTED";
        }
    }

    private String traversalOrGitPath() throws Exception {
        try (Fixture fixture = fixture("unsafe-paths")) {
            BundleBuilder traversal = archiveBundle(fixture, "archive-traversal");
            traversal.operations.get(0).path = "../escape";
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> verify(traversal.write(), signingKey));

            BundleBuilder gitPath = archiveBundle(fixture, "archive-git-path");
            gitPath.operations.get(0).path = ".git/config";
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> verify(gitPath.write(), signingKey));
            fixture.assertOnlyCustomerWorktree();
            return "REJECTED";
        }
    }

    private String symlinkTarget() throws Exception {
        try (Fixture fixture = fixture("symlink-target")) {
            fixture.addSymlink("linked.txt", "README.md");
            Snapshot before = fixture.snapshot();
            byte[] replacement = "replacement\n".getBytes(StandardCharsets.UTF_8);
            BundleBuilder builder = bundle(
                    fixture,
                    "symlink-target",
                    "ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1");
            builder.modify(
                    "linked.txt",
                    sha("README baseline\n".getBytes(StandardCharsets.UTF_8)),
                    "blobs/link.txt",
                    replacement);
            BundleFile bundle = builder.write();
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> apply(fixture, verify(bundle, signingKey), Fault.NONE));
            fixture.assertNoActionBranch(builder.actionId);
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "REJECTED";
        }
    }

    private String symlinkParent() throws Exception {
        try (Fixture fixture = fixture("symlink-parent")) {
            fixture.addSymlinkParent();
            Snapshot before = fixture.snapshot();
            BundleBuilder builder = bundle(
                    fixture,
                    "symlink-parent",
                    "ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1");
            builder.add(
                    "alias/new.txt",
                    "blobs/new.txt",
                    "new\n".getBytes(StandardCharsets.UTF_8));
            BundleFile bundle = builder.write();
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> apply(fixture, verify(bundle, signingKey), Fault.NONE));
            fixture.assertNoActionBranch(builder.actionId);
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "REJECTED";
        }
    }

    private String submoduleBoundary() throws Exception {
        try (Fixture fixture = fixture("submodule")) {
            fixture.addGitlink("modules/dep");
            Snapshot before = fixture.snapshot();
            BundleBuilder builder = bundle(
                    fixture,
                    "submodule-boundary",
                    "ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1");
            builder.add(
                    "modules/dep/new.txt",
                    "blobs/submodule-new.txt",
                    "new\n".getBytes(StandardCharsets.UTF_8));
            BundleFile bundle = builder.write();
            expectFailure(
                    PatchFailure.Status.REJECTED,
                    () -> apply(fixture, verify(bundle, signingKey), Fault.NONE));
            fixture.assertNoActionBranch(builder.actionId);
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "REJECTED";
        }
    }

    private String postimageMismatch() throws Exception {
        try (Fixture fixture = fixture("postimage-mismatch")) {
            Snapshot before = fixture.snapshot();
            BundleBuilder builder = archiveBundle(fixture, "archive-postimage");
            BundleFile bundle = builder.write();
            expectFailure(
                    PatchFailure.Status.UNCHANGED,
                    () -> apply(
                            fixture,
                            verify(bundle, signingKey),
                            Fault.CORRUPT_POSTIMAGE));
            fixture.assertNoActionBranch(builder.actionId);
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "UNCHANGED";
        }
    }

    private String branchWithoutPr() throws Exception {
        try (Fixture fixture = fixture("branch-without-pr")) {
            BundleBuilder builder = archiveBundle(fixture, "archive-recovery");
            BundleFile bundle = builder.write();
            VerifiedPatchBundle verified = verify(bundle, signingKey);
            fixture.pullRequests.failNextCreate = true;
            expectFailure(
                    PatchFailure.Status.UNCHANGED,
                    () -> apply(fixture, verified, Fault.NONE));
            String head = fixture.readActionBranch(builder.actionId);
            require(!head.isEmpty(), "branch retained after PR failure");
            require(fixture.pullRequests.entries.isEmpty(), "no PR after injected failure");
            PatchBundleApplier.Result recovered = apply(
                    fixture,
                    verified,
                    Fault.NONE);
            require(recovered.outcome() == Outcome.DRAFT_PR_CREATED, "recovered PR");
            require(recovered.headCommit().equals(head), "recovery reused immutable head");
            fixture.assertOnlyCustomerWorktree();
            return "DRAFT_PR_CREATED";
        }
    }

    private String existingBranchConflict() throws Exception {
        try (Fixture fixture = fixture("branch-conflict")) {
            BundleBuilder builder = archiveBundle(fixture, "archive-conflict");
            BundleFile bundle = builder.write();
            fixture.createActionBranch(builder.actionId, fixture.baseRevision);
            expectFailure(
                    PatchFailure.Status.PROPOSED,
                    () -> apply(fixture, verify(bundle, signingKey), Fault.NONE));
            require(
                    fixture.readActionBranch(builder.actionId)
                            .equals(fixture.baseRevision),
                    "conflicting branch was not modified");
            fixture.assertOnlyCustomerWorktree();
            return "PROPOSED";
        }
    }

    private String existingPrReplay() throws Exception {
        try (Fixture fixture = fixture("pr-replay")) {
            BundleFile bundle = customTaskBundle(fixture, "task-pr-replay").write();
            VerifiedPatchBundle verified = verify(bundle, signingKey);
            PatchBundleApplier.Result first = apply(fixture, verified, Fault.NONE);
            require(first.outcome() == Outcome.DRAFT_PR_CREATED, "first PR creation");
            PatchBundleApplier.Result replay = apply(fixture, verified, Fault.NONE);
            require(replay.outcome() == Outcome.EXISTING_DRAFT_PR, "existing PR replay");
            require(replay.headCommit().equals(first.headCommit()), "PR head replay");
            fixture.assertOnlyCustomerWorktree();
            return "EXISTING_DRAFT_PR";
        }
    }

    private String interruptedStaging() throws Exception {
        try (Fixture fixture = fixture("interrupted-staging")) {
            Snapshot before = fixture.snapshot();
            BundleBuilder builder = customTaskBundle(fixture, "task-interrupted");
            BundleFile bundle = builder.write();
            expectFailure(
                    PatchFailure.Status.UNCHANGED,
                    () -> apply(
                            fixture,
                            verify(bundle, signingKey),
                            Fault.INTERRUPT_AFTER_WRITE));
            fixture.assertNoActionBranch(builder.actionId);
            fixture.assertCheckoutUnchanged(before);
            fixture.assertOnlyCustomerWorktree();
            return "UNCHANGED";
        }
    }

    private void assertExactPostMergeRevert() throws Exception {
        try (Fixture fixture = fixture("post-merge-revert")) {
            String path = "build.gradle.kts";
            String originalSource = fixture.show(fixture.baseRevision, path);
            BundleFile originalFile = archiveBundle(fixture, "post-merge").write();
            VerifiedPatchBundle original = verify(originalFile, signingKey);
            PatchBundleApplier.Result applied = apply(fixture, original, Fault.NONE);
            fixture.git("update-ref", "refs/heads/main", applied.headCommit());
            String mergedMain = fixture.git("rev-parse", "refs/heads/main");

            ExactRevertBundleGenerator.Validation validation =
                    new ExactRevertBundleGenerator.Validation(
                            "request-revert-spk",
                            "result-revert-spk",
                            "sha256:" + "c".repeat(64),
                            NOW.minusSeconds(120),
                            NOW.plusSeconds(3600));
            ExactRevertBundleGenerator.GeneratedRevert generated =
                    ExactRevertBundleGenerator.generate(
                            original,
                            fixture.repository,
                            applied.headCommit(),
                            validation,
                            NOW.minusSeconds(60),
                            NOW.plusSeconds(1800),
                            KEY_ID,
                            signingKey.getPrivate());
            Path revertRoot = fixture.bundles.resolve("generated-revert");
            Path manifest = generated.write(revertRoot);
            VerifiedPatchBundle verifiedRevert = PatchBundleVerifier.verify(
                    manifest,
                    revertRoot,
                    REPOSITORY_ID,
                    generated.actionId(),
                    KEY_ID,
                    signingKey.getPublic(),
                    NOW);
            PatchBundleApplier.Result reverted = apply(
                    fixture,
                    verifiedRevert,
                    Fault.NONE);
            require(reverted.outcome() == Outcome.DRAFT_PR_CREATED,
                    "regression creates a draft revert PR");
            require(fixture.show(reverted.branch(), path).equals(originalSource),
                    "revert branch restores the exact original bytes");
            require(fixture.git("rev-parse", "refs/heads/main").equals(mergedMain),
                    "revert path does not modify the default branch");

            expectFailure(
                    PatchFailure.Status.PROPOSED,
                    () -> ExactRevertBundleGenerator.generate(
                            original,
                            fixture.repository,
                            fixture.baseRevision,
                            validation,
                            NOW.minusSeconds(60),
                            NOW.plusSeconds(1800),
                            KEY_ID,
                            signingKey.getPrivate()));

            BundleBuilder addBuilder = bundle(
                    fixture,
                    "add-only-post-merge",
                    "ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1");
            addBuilder.add(
                    "new-file.txt",
                    "blobs/new-file.txt",
                    "new file\n".getBytes(StandardCharsets.UTF_8));
            VerifiedPatchBundle addBundle = verify(addBuilder.write(), signingKey);
            expectFailure(
                    PatchFailure.Status.PROPOSED,
                    () -> ExactRevertBundleGenerator.generate(
                            addBundle,
                            fixture.repository,
                            fixture.baseRevision,
                            validation,
                            NOW.minusSeconds(60),
                            NOW.plusSeconds(1800),
                            KEY_ID,
                            signingKey.getPrivate()));
            fixture.assertOnlyCustomerWorktree();
        }
    }

    private void assertExpandedRecipeReverts() throws Exception {
        try (Fixture fixture = fixture("groovy-post-merge-revert")) {
            assertExactRecipeRevert(
                    fixture,
                    groovyArchiveBundle(fixture, "groovy-post-merge"),
                    "build.gradle",
                    "groovy");
        }
        try (Fixture fixture = fixture("build-cache-post-merge-revert")) {
            assertExactRecipeRevert(
                    fixture,
                    buildCacheBundle(fixture, "build-cache-post-merge"),
                    "gradle.properties",
                    "build-cache");
        }
        try (Fixture fixture = fixture("custom-task-post-merge-revert")) {
            assertExactRecipeRevert(
                    fixture,
                    customTaskBundle(fixture, "custom-task-post-merge"),
                    "buildSrc/src/main/java/com/example/build/BundleFrontend.java",
                    "custom-task");
        }
        for (String recipeId : List.of(
                ReviewedNativePatchJavaRecipe.RELATIVE_CACHEABILITY_RECIPE_ID,
                ReviewedNativePatchJavaRecipe.MARKER_ONLY_CACHEABILITY_RECIPE_ID)) {
            try (Fixture fixture = fixture("reviewed-native-post-merge-revert")) {
                assertExactRecipeRevert(
                        fixture,
                        reviewedNativeBundle(fixture, recipeId),
                        "build.gradle.kts",
                        recipeId.toLowerCase(Locale.ROOT));
            }
        }
    }

    private void assertExactRecipeRevert(
            Fixture fixture,
            BundleBuilder builder,
            String path,
            String label) throws Exception {
        String originalSource = fixture.show(fixture.baseRevision, path);
        VerifiedPatchBundle original = verify(builder.write(), signingKey);
        PatchBundleApplier.Result applied = apply(fixture, original, Fault.NONE);
        fixture.git("update-ref", "refs/heads/main", applied.headCommit());
        String mergedMain = fixture.git("rev-parse", "refs/heads/main");
        ExactRevertBundleGenerator.Validation validation =
                new ExactRevertBundleGenerator.Validation(
                        "request-" + label + "-revert",
                        "result-" + label + "-revert",
                        "sha256:" + "d".repeat(64),
                        NOW.minusSeconds(120),
                        NOW.plusSeconds(3600));
        ExactRevertBundleGenerator.GeneratedRevert generated =
                ExactRevertBundleGenerator.generate(
                        original,
                        fixture.repository,
                        applied.headCommit(),
                        validation,
                        NOW.minusSeconds(60),
                        NOW.plusSeconds(1800),
                        KEY_ID,
                        signingKey.getPrivate());
        Path revertRoot = fixture.bundles.resolve("generated-" + label + "-revert");
        VerifiedPatchBundle verifiedRevert = PatchBundleVerifier.verify(
                generated.write(revertRoot),
                revertRoot,
                REPOSITORY_ID,
                generated.actionId(),
                KEY_ID,
                signingKey.getPublic(),
                NOW);
        PatchBundleApplier.Result reverted = apply(
                fixture,
                verifiedRevert,
                Fault.NONE);
        require(reverted.outcome() == Outcome.DRAFT_PR_CREATED,
                label + " regression creates a draft revert PR");
        require(fixture.show(reverted.branch(), path).equals(originalSource),
                label + " revert restores exact original bytes");
        require(fixture.git("rev-parse", "refs/heads/main").equals(mergedMain),
                label + " revert preserves the default branch");
        fixture.assertOnlyCustomerWorktree();
    }

    private Fixture fixture(String name) throws Exception {
        return new Fixture(temporaryRoot.resolve(name));
    }

    private BundleBuilder archiveBundle(Fixture fixture, String action)
            throws Exception {
        BundleBuilder builder = bundle(
                fixture,
                action,
                "ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1");
        ArchiveReproducibilityRecipe.Result recipe =
                ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts",
                        Files.readAllBytes(
                                fixture.repository.resolve("build.gradle.kts")));
        builder.modify(
                "build.gradle.kts",
                recipe.preimageDigest(),
                "blobs/archive-build.gradle.kts",
                recipe.postimage());
        return builder;
    }

    private BundleBuilder groovyArchiveBundle(Fixture fixture, String action)
            throws Exception {
        BundleBuilder builder = bundle(
                fixture,
                action,
                ArchiveReproducibilityGroovyDslRecipe.RECIPE_ID);
        ArchiveReproducibilityGroovyDslRecipe.Result recipe =
                ArchiveReproducibilityGroovyDslRecipe.apply(
                        "build.gradle",
                        Files.readAllBytes(fixture.repository.resolve("build.gradle")));
        builder.modify(
                "build.gradle",
                recipe.preimageDigest(),
                "blobs/archive-build.gradle",
                recipe.postimage());
        return builder;
    }

    private BundleBuilder buildCacheBundle(Fixture fixture, String action)
            throws Exception {
        BundleBuilder builder = bundle(
                fixture,
                action,
                GradleBuildCachePropertiesRecipe.RECIPE_ID);
        GradleBuildCachePropertiesRecipe.Result recipe =
                GradleBuildCachePropertiesRecipe.apply(
                        "gradle.properties",
                        Files.readAllBytes(fixture.repository.resolve("gradle.properties")));
        builder.modify(
                "gradle.properties",
                recipe.preimageDigest(),
                "blobs/gradle.properties",
                recipe.postimage());
        return builder;
    }

    private BundleBuilder customTaskBundle(Fixture fixture, String action)
            throws Exception {
        BundleBuilder builder = bundle(
                fixture,
                action,
                "CUSTOM_TASK_CONTRACT_JAVA_V1");
        String path = "buildSrc/src/main/java/com/example/build/BundleFrontend.java";
        byte[] content = Files.readAllBytes(contractBlobs.resolve("custom-task.java"));
        builder.modify(
                path,
                sha(Files.readAllBytes(fixture.repository.resolve(path))),
                "blobs/custom-task.java",
                content);
        return builder;
    }

    private BundleBuilder reviewedNativeBundle(Fixture fixture, String recipeId)
            throws Exception {
        BundleBuilder builder = bundle(fixture, "reviewed-native", recipeId);
        String path = "build.gradle.kts";
        byte[] preimage = Files.readAllBytes(fixture.repository.resolve(path));
        byte[] suffix = "// reviewed native patch\n".getBytes(StandardCharsets.UTF_8);
        byte[] postimage = Arrays.copyOf(preimage, preimage.length + suffix.length);
        System.arraycopy(suffix, 0, postimage, preimage.length, suffix.length);
        builder.modify(path, sha(preimage), "blobs/reviewed-native.gradle.kts", postimage);
        return builder;
    }

    private BundleBuilder bundle(Fixture fixture, String action, String recipe) {
        return new BundleBuilder(fixture, "action-" + action, recipe);
    }

    private VerifiedPatchBundle verify(BundleFile bundle, KeyPair trusted)
            throws PatchFailure {
        return PatchBundleVerifier.verify(
                bundle.manifest(),
                bundle.root(),
                REPOSITORY_ID,
                bundle.actionId(),
                KEY_ID,
                trusted.getPublic(),
                NOW);
    }

    private static PatchBundleApplier.Result apply(
            Fixture fixture,
            VerifiedPatchBundle bundle,
            Fault fault) throws PatchFailure {
        return new PatchBundleApplier().apply(
                bundle,
                fixture.repository,
                fixture.staging,
                fixture.pullRequests,
                fault);
    }

    private void assertArchiveRecipeSafety() throws Exception {
        byte[] source = "plugins { base }\n".getBytes(StandardCharsets.UTF_8);
        ArchiveReproducibilityRecipe.Result first =
                ArchiveReproducibilityRecipe.apply("build.gradle.kts", source);
        require(first.changed()
                        && new String(first.postimage(), StandardCharsets.UTF_8)
                                .contains("isPreserveFileTimestamps = false"),
                "archive recipe exact output");
        ArchiveReproducibilityRecipe.Result repeated =
                ArchiveReproducibilityRecipe.apply("build.gradle.kts", first.postimage());
        require(!repeated.changed()
                        && repeated.preimageDigest().equals(repeated.postimageDigest())
                        && Arrays.equals(repeated.postimage(), first.postimage()),
                "archive recipe idempotency");
        byte[] maximumSource = new byte[1024 * 1024 - 187];
        Arrays.fill(maximumSource, (byte) ' ');
        maximumSource[maximumSource.length - 1] = '\n';
        ArchiveReproducibilityRecipe.Result maximum =
                ArchiveReproducibilityRecipe.apply("build.gradle.kts", maximumSource);
        ArchiveReproducibilityRecipe.Result maximumRepeated =
                ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts", maximum.postimage());
        require(!maximumRepeated.changed()
                        && maximum.postimage().length == 1024 * 1024
                        && Arrays.equals(maximum.postimage(), maximumRepeated.postimage()),
                "archive recipe maximum-size idempotency");
        byte[] oversizedSource = new byte[1024 * 1024 - 186];
        Arrays.fill(oversizedSource, (byte) ' ');
        oversizedSource[oversizedSource.length - 1] = '\n';
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts", oversizedSource));
        byte[] defensive = first.postimage();
        defensive[0] ^= 1;
        require(!Arrays.equals(defensive, first.postimage()),
                "archive recipe defensive output");
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityRecipe.apply(
                        "build.gradle", source));
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts",
                        "isReproducibleFileOrder = true\n"
                                .getBytes(StandardCharsets.UTF_8)));
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts",
                        "@file:Suppress(\"unused\")\nplugins { base }\n"
                                .getBytes(StandardCharsets.UTF_8)));
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts",
                        "plugins { base }\r\n".getBytes(StandardCharsets.UTF_8)));
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts",
                        new byte[] {'x', 0, '\n'}));
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityRecipe.apply(
                        "build.gradle.kts", new byte[] {(byte) 0xc3, (byte) 0x28}));
    }

    private void assertExpandedRecipeSafety() throws Exception {
        byte[] groovySource = "plugins { id 'base' }\n".getBytes(StandardCharsets.UTF_8);
        ArchiveReproducibilityGroovyDslRecipe.Result groovy =
                ArchiveReproducibilityGroovyDslRecipe.apply(
                        "build.gradle", groovySource);
        require(groovy.changed()
                        && new String(groovy.postimage(), StandardCharsets.UTF_8)
                                .contains("preserveFileTimestamps = false"),
                "Groovy archive recipe exact output");
        ArchiveReproducibilityGroovyDslRecipe.Result repeatedGroovy =
                ArchiveReproducibilityGroovyDslRecipe.apply(
                        "build.gradle", groovy.postimage());
        require(!repeatedGroovy.changed()
                        && Arrays.equals(groovy.postimage(), repeatedGroovy.postimage()),
                "Groovy archive recipe idempotency");
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityGroovyDslRecipe.apply(
                        "build.gradle.kts", groovySource));
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> ArchiveReproducibilityGroovyDslRecipe.apply(
                        "build.gradle",
                        "preserveFileTimestamps = true\n"
                                .getBytes(StandardCharsets.UTF_8)));

        byte[] propertiesSource =
                "org.gradle.jvmargs=-Xmx2g\n".getBytes(StandardCharsets.UTF_8);
        GradleBuildCachePropertiesRecipe.Result buildCache =
                GradleBuildCachePropertiesRecipe.apply(
                        "gradle.properties", propertiesSource);
        require(buildCache.changed()
                        && new String(buildCache.postimage(), StandardCharsets.UTF_8)
                                .endsWith("org.gradle.caching=true\n"),
                "build-cache recipe exact output");
        GradleBuildCachePropertiesRecipe.Result repeatedBuildCache =
                GradleBuildCachePropertiesRecipe.apply(
                        "gradle.properties", buildCache.postimage());
        require(!repeatedBuildCache.changed()
                        && Arrays.equals(
                                buildCache.postimage(), repeatedBuildCache.postimage()),
                "build-cache recipe idempotency");
        byte[] defensive = buildCache.postimage();
        defensive[0] ^= 1;
        require(!Arrays.equals(defensive, buildCache.postimage()),
                "build-cache recipe defensive output");
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> GradleBuildCachePropertiesRecipe.apply(
                        "gradle.properties",
                        "# org.gradle.caching=false\n"
                                .getBytes(StandardCharsets.UTF_8)));
        expectFailure(PatchFailure.Status.PROPOSED,
                () -> GradleBuildCachePropertiesRecipe.apply(
                        "sub/gradle.properties", propertiesSource));
    }

    private void assertSignerRejectsIncompleteManifest() throws Exception {
        try {
            PatchBundleSigner.sign(
                    "{}".getBytes(StandardCharsets.UTF_8),
                    KEY_ID,
                    signingKey.getPrivate());
            throw new AssertionError("incomplete unsigned manifest was signed");
        } catch (PatchFailure failure) {
            require(failure.status() == PatchFailure.Status.REJECTED,
                    "incomplete signer input status");
        }
    }

    private void assertDuplicateKeysRejected() {
        try {
            StrictJson.parse(
                    "{\"schemaVersion\":\"1.0\",\"schemaVersion\":\"1.0\"}"
                            .getBytes(StandardCharsets.UTF_8));
            throw new AssertionError("duplicate JSON key was accepted");
        } catch (IllegalArgumentException expected) {
            require(
                    expected.getMessage().contains("duplicate-key"),
                    "duplicate key diagnostic");
        }
    }

    private void writeReport(Path report, List<CaseResult> results)
            throws IOException {
        List<Object> cases = new ArrayList<>();
        for (CaseResult result : results) {
            Map<String, Object> item = new LinkedHashMap<>();
            item.put("id", result.id());
            item.put("expected", result.expected());
            item.put("actual", result.actual());
            cases.add(item);
        }
        Map<String, Object> root = new LinkedHashMap<>();
        root.put("schemaVersion", "buildopt.spikes/patcher-result/v1");
        root.put("caseCount", results.size());
        root.put("javaRuntimeFeature", Runtime.version().feature());
        root.put("allPassed", true);
        root.put("realGitWorktrees", true);
        root.put("strictJsonDuplicateKeys", true);
        root.put("ed25519JcsVerified", true);
        root.put("productionSignerUsed", true);
        root.put("signerDeterministic", true);
        root.put("signerRejectsIncomplete", true);
        root.put("signerOutputDefensive", true);
        root.put("archiveRecipeGenerated", true);
        root.put("archiveRecipeIdempotent", true);
        root.put("archiveRecipeFailClosed", true);
        root.put("candidateControlValidated", true);
        root.put("archiveArtifactAdapterValidated", true);
        root.put("exactArtifactAdapterValidated", true);
        root.put("expandedRecipeCandidateControlValidated", true);
        root.put("candidateReproducibilityValidated", true);
        root.put("candidateValidationFailClosed", true);
        root.put("fullRelevantValidationIntegrated", true);
        root.put("testOptimizationSignedResultVerified", true);
        root.put("testOptimizationPollingBounded", true);
        root.put("testOptimizationNotRequiredNoContact", true);
        root.put("postMergeEvidenceClassified", true);
        root.put("contextualImpactNotPromoted", true);
        root.put("exactInverseBundleGenerated", true);
        root.put("registryRecipeInverseBundlesValidated", true);
        root.put("draftRevertPathValidated", true);
        root.put("defaultBranchPreservedAfterRegression", true);
        root.put("bundleContentExecuted", false);
        root.put("customerCheckoutHooksExecuted", false);
        root.put("customerContentFiltersExecuted", false);
        root.put("remoteMutationPerformed", false);
        root.put("cases", cases);
        Files.createDirectories(report.getParent());
        Path temporary = Files.createTempFile(
                report.getParent(),
                ".patcher-spike-",
                ".tmp");
        Files.write(temporary, StrictJson.canonicalBytes(root));
        try {
            Files.move(temporary, report, ATOMIC_MOVE, REPLACE_EXISTING);
        } finally {
            Files.deleteIfExists(temporary);
        }
    }

    private static void expectFailure(
            PatchFailure.Status expected,
            CheckedOperation operation) throws Exception {
        try {
            operation.run();
            throw new AssertionError("expected PatchFailure " + expected);
        } catch (PatchFailure failure) {
            if (failure.status() != expected) {
                throw new AssertionError(
                        "failure status " + failure.status() + " != " + expected,
                        failure);
            }
        }
    }

    private static String sha(byte[] content) throws GeneralSecurityException {
        return PatchBundleVerifier.digestBytes(content);
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    private static Map<String, Object> object(Object value) {
        if (!(value instanceof Map<?, ?> raw)) {
            throw new IllegalArgumentException("expected JSON object");
        }
        Map<String, Object> result = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : raw.entrySet()) {
            result.put((String) entry.getKey(), entry.getValue());
        }
        return result;
    }

    private static List<Object> array(Object value) {
        if (!(value instanceof List<?> raw)) {
            throw new IllegalArgumentException("expected JSON array");
        }
        return new ArrayList<>(raw);
    }

    private static String string(Map<String, Object> object, String field) {
        Object value = object.get(field);
        if (!(value instanceof String stringValue)) {
            throw new IllegalArgumentException(field + " is not a string");
        }
        return stringValue;
    }

    private static void deleteRecursively(Path path) {
        if (!Files.exists(path, NOFOLLOW_LINKS)) {
            return;
        }
        try {
            BasicFileAttributes attributes = Files.readAttributes(
                    path,
                    BasicFileAttributes.class,
                    NOFOLLOW_LINKS);
            if (attributes.isDirectory() && !attributes.isSymbolicLink()) {
                try (DirectoryStream<Path> children = Files.newDirectoryStream(path)) {
                    for (Path child : children) {
                        deleteRecursively(child);
                    }
                }
            }
            Files.deleteIfExists(path);
        } catch (IOException exception) {
            throw new IllegalStateException("cannot remove spike path " + path, exception);
        }
    }

    private final class BundleBuilder {
        private final Fixture fixture;
        private final String actionId;
        private final String recipe;
        private final List<MutableOperation> operations = new ArrayList<>();
        private String baseTree;
        private String sourceStateDigest;
        private Instant createdAt = NOW.minus(1, ChronoUnit.MINUTES);
        private Instant expiresAt = NOW.plus(1, ChronoUnit.HOURS);

        private BundleBuilder(Fixture fixture, String actionId, String recipe) {
            this.fixture = fixture;
            this.actionId = actionId;
            this.recipe = recipe;
            this.baseTree = fixture.baseTree;
            this.sourceStateDigest = fixture.sourceStateDigest;
        }

        private void modify(
                String path,
                String preimage,
                String blobRef,
                byte[] content) {
            operations.add(new MutableOperation(
                    "MODIFY",
                    path,
                    preimage,
                    blobRef,
                    content.clone()));
        }

        private void add(String path, String blobRef, byte[] content) {
            operations.add(new MutableOperation(
                    "ADD",
                    path,
                    "",
                    blobRef,
                    content.clone()));
        }

        private BundleFile write() throws Exception {
            if (operations.isEmpty()) {
                throw new IllegalStateException("bundle has no operations");
            }
            Path root = fixture.bundles.resolve(
                    actionId + "-" + NEXT_BUNDLE.getAndIncrement());
            Files.createDirectories(root);
            List<MutableOperation> sortedBlobs = new ArrayList<>(operations);
            sortedBlobs.sort(Comparator.comparing(operation -> operation.blobRef));
            List<Object> blobValues = new ArrayList<>();
            for (MutableOperation operation : sortedBlobs) {
                Path blob = root.resolve(operation.blobRef);
                Files.createDirectories(blob.getParent());
                Files.write(blob, operation.content);
                Map<String, Object> value = new LinkedHashMap<>();
                value.put("blobRef", operation.blobRef);
                value.put("blobSha256", sha(operation.content));
                value.put("sizeBytes", operation.content.length);
                value.put("mediaType", "text/plain");
                value.put("encoding", "UTF-8");
                blobValues.add(value);
            }

            List<Object> operationValues = new ArrayList<>();
            for (int index = 0; index < operations.size(); index++) {
                MutableOperation operation = operations.get(index);
                Map<String, Object> value = new LinkedHashMap<>();
                value.put("order", index + 1);
                value.put("type", operation.type);
                value.put("path", operation.path);
                value.put("expectedMode", "100644");
                if ("MODIFY".equals(operation.type)) {
                    value.put("preimageDigest", operation.preimage);
                }
                value.put("postimageDigest", sha(operation.content));
                value.put("replacementBlob", operation.blobRef);
                operationValues.add(value);
            }

            Map<String, Object> recipeValue = new LinkedHashMap<>();
            recipeValue.put("id", recipe);
            recipeValue.put("version", "1.0");
            PatchAutopilotRecipeRegistry.Definition recipeDefinition =
                    PatchAutopilotRecipeRegistry.find(recipe, "1.0").orElseThrow();
            if (recipeDefinition.reviewedAdapterRequired()) {
                Map<String, Object> adapter = new LinkedHashMap<>();
                adapter.put("adapterId", "frontend-bundle-v2");
                adapter.put("adapterDigest", "sha256:" + "5".repeat(64));
                adapter.put("evidenceRef", "evidence-spk-004");
                recipeValue.put("reviewedAdapter", adapter);
            }

            Map<String, Object> validation = new LinkedHashMap<>();
            validation.put("mode", "FULL_RELEVANT_VALIDATION");
            validation.put("status", "PASSED");
            validation.put("requestId", "request-spk-004");
            validation.put("resultId", "result-spk-004");
            validation.put("artifactSetDigest", "sha256:" + "a".repeat(64));
            validation.put("completedAt", createdAt.minusSeconds(1).toString());
            validation.put("expiresAt", expiresAt.toString());

            Map<String, Object> delivery = new LinkedHashMap<>();
            delivery.put("branchPrefix", "buildopt/");
            delivery.put("draftPullRequest", true);
            delivery.put("autoMerge", false);
            delivery.put("forcePush", false);
            delivery.put("modifyExistingBranch", false);

            Map<String, Object> rootValue = new LinkedHashMap<>();
            rootValue.put("schemaVersion", "1.0");
            rootValue.put("recordType", "PATCH_BUNDLE");
            rootValue.put("contractVersion", "buildopt-patch-bundle/v1");
            rootValue.put("repositoryId", REPOSITORY_ID);
            rootValue.put("actionId", actionId);
            rootValue.put("baseRevision", fixture.baseRevision);
            rootValue.put("baseTree", baseTree);
            rootValue.put("sourceStateDigest", sourceStateDigest);
            rootValue.put("recipe", recipeValue);
            rootValue.put("createdAt", createdAt.toString());
            rootValue.put("expiresAt", expiresAt.toString());
            rootValue.put("operations", operationValues);
            rootValue.put("blobs", blobValues);
            rootValue.put("validation", validation);
            rootValue.put("delivery", delivery);
            byte[] unsignedManifest = StrictJson.canonicalBytes(rootValue);
            PatchBundleSigner.SignedPatchBundle signed = PatchBundleSigner.sign(
                    unsignedManifest,
                    KEY_ID,
                    signingKey.getPrivate());
            PatchBundleSigner.SignedPatchBundle repeated = PatchBundleSigner.sign(
                    unsignedManifest,
                    KEY_ID,
                    signingKey.getPrivate());
            require(signed.bundleDigest().equals(repeated.bundleDigest())
                            && Arrays.equals(
                                    signed.canonicalManifest(),
                                    repeated.canonicalManifest()),
                    "signer determinism");
            byte[] defensive = signed.canonicalManifest();
            defensive[0] ^= 1;
            require(!Arrays.equals(defensive, signed.canonicalManifest()),
                    "signer defensive output");
            Path manifest = root.resolve("manifest.json");
            Files.write(manifest, signed.canonicalManifest());
            return new BundleFile(root, manifest, actionId);
        }
    }

    private static final class MutableOperation {
        private final String type;
        private String path;
        private String preimage;
        private final String blobRef;
        private final byte[] content;

        private MutableOperation(
                String type,
                String path,
                String preimage,
                String blobRef,
                byte[] content) {
            this.type = type;
            this.path = path;
            this.preimage = preimage;
            this.blobRef = blobRef;
            this.content = content;
        }
    }

    private static final class Fixture implements AutoCloseable {
        private final Path root;
        private final Path repository;
        private final Path bundles;
        private final Path staging;
        private final MemoryPullRequests pullRequests = new MemoryPullRequests();
        private String baseRevision;
        private String baseTree;
        private String sourceStateDigest;

        private Fixture(Path root) throws Exception {
            this.root = root;
            this.repository = root.resolve("repository");
            this.bundles = root.resolve("bundles");
            this.staging = root.resolve("staging");
            Files.createDirectories(repository);
            Files.createDirectories(bundles);
            Files.createDirectories(staging);
            git("init", "-b", "main");
            Files.writeString(
                    repository.resolve("README.md"),
                    "README baseline\n",
                    StandardCharsets.UTF_8);
            Files.writeString(
                    repository.resolve("build.gradle.kts"),
                    "plugins { base }\n",
                    StandardCharsets.UTF_8);
            Files.writeString(
                    repository.resolve("build.gradle"),
                    "plugins { id 'base' }\n",
                    StandardCharsets.UTF_8);
            Files.writeString(
                    repository.resolve("gradle.properties"),
                    "org.gradle.jvmargs=-Xmx2g\n",
                    StandardCharsets.UTF_8);
            Path customTask = repository.resolve(
                    "buildSrc/src/main/java/com/example/build/BundleFrontend.java");
            Files.createDirectories(customTask.getParent());
            Files.writeString(
                    customTask,
                    "package com.example.build;\n"
                            + "public final class BundleFrontend { }\n",
                    StandardCharsets.UTF_8);
            commitAll("baseline");
            refreshBase();
        }

        private void addSymlink(String path, String target) throws Exception {
            Files.createSymbolicLink(repository.resolve(path), Path.of(target));
            commitAll("add symlink target");
            refreshBase();
        }

        private void addSymlinkParent() throws Exception {
            Path real = repository.resolve("real");
            Files.createDirectories(real);
            Files.writeString(
                    real.resolve("keep.txt"),
                    "keep\n",
                    StandardCharsets.UTF_8);
            Files.createSymbolicLink(repository.resolve("alias"), Path.of("real"));
            commitAll("add symlink parent");
            refreshBase();
        }

        private void addGitlink(String path) throws Exception {
            Path parent = repository.resolve(path).getParent();
            Files.createDirectories(parent);
            git(
                    "update-index",
                    "--add",
                    "--cacheinfo",
                    "160000," + baseRevision + "," + path);
            git(
                    "-c",
                    "core.hooksPath=/dev/null",
                    "-c",
                    "user.name=BuildOpt Fixture",
                    "-c",
                    "user.email=fixture@buildopt.invalid",
                    "commit",
                    "--no-gpg-sign",
                    "-m",
                    "add gitlink");
            refreshBase();
        }

        private void installExplosiveCheckoutMechanisms() throws Exception {
            Files.writeString(
                    repository.resolve(".gitattributes"),
                    "*.kts filter=explode\n",
                    StandardCharsets.UTF_8);
            commitAll("add inert filter declaration");
            refreshBase();

            Path hooks = root.resolve("hooks");
            Files.createDirectories(hooks);
            Path marker = root.resolve("unexpected-execution");
            Path hook = hooks.resolve("post-checkout");
            Files.writeString(
                    hook,
                    "#!/bin/sh\n"
                            + "touch '" + marker + "'\n",
                    StandardCharsets.UTF_8);
            Files.setPosixFilePermissions(
                    hook,
                    PosixFilePermissions.fromString("rwx------"));
            git("config", "core.hooksPath", hooks.toString());
            git(
                    "config",
                    "filter.explode.smudge",
                    "touch '" + marker + "'; cat");
            git("config", "filter.explode.clean", "cat");
            git("config", "filter.explode.required", "true");
        }

        private void assertNoUnexpectedExecution() {
            require(
                    !Files.exists(root.resolve("unexpected-execution"), NOFOLLOW_LINKS),
                    "customer hook or content filter executed");
        }

        private void commitAll(String message) throws Exception {
            git("add", "--all");
            git(
                    "-c",
                    "core.hooksPath=/dev/null",
                    "-c",
                    "user.name=BuildOpt Fixture",
                    "-c",
                    "user.email=fixture@buildopt.invalid",
                    "commit",
                    "--no-gpg-sign",
                    "-m",
                    message);
        }

        private void refreshBase() throws PatchFailure {
            baseRevision = git("rev-parse", "HEAD");
            baseTree = git("rev-parse", "HEAD^{tree}");
            sourceStateDigest = PatchBundleApplier.sourceStateDigest(
                    repository,
                    baseRevision);
        }

        private Snapshot snapshot() {
            return new Snapshot(
                    git("rev-parse", "HEAD"),
                    git("status", "--porcelain=v1", "--untracked-files=all"));
        }

        private void assertCheckoutUnchanged(Snapshot before) {
            Snapshot after = snapshot();
            require(before.equals(after), "customer checkout or index changed");
        }

        private void assertOnlyCustomerWorktree() {
            long count = git("worktree", "list", "--porcelain").lines()
                    .filter(line -> line.startsWith("worktree "))
                    .count();
            require(count == 1, "private staging worktree leaked");
            try (DirectoryStream<Path> children = Files.newDirectoryStream(staging)) {
                require(
                        !children.iterator().hasNext(),
                        "private staging directory leaked");
            } catch (IOException exception) {
                throw new IllegalStateException(
                        "cannot inspect private staging cleanup",
                        exception);
            }
        }

        private void assertNoActionBranch(String actionId) {
            require(readActionBranch(actionId).isEmpty(), "unexpected action branch");
        }

        private String readActionBranch(String actionId) {
            GitCommand result = gitResult(
                    "rev-parse",
                    "--verify",
                    "--quiet",
                    "refs/heads/buildopt/" + actionId);
            if (result.exitCode == 1) {
                return "";
            }
            require(result.exitCode == 0, "cannot read action branch");
            return result.output.trim();
        }

        private void createActionBranch(String actionId, String commit) {
            git("update-ref", "refs/heads/buildopt/" + actionId, commit);
        }

        private String show(String branch, String path) {
            GitCommand result = gitResult("show", branch + ":" + path);
            if (result.exitCode != 0) {
                throw new IllegalStateException(
                        "cannot read " + branch + ":" + path + ": " + result.output);
            }
            return result.output;
        }

        private String git(String... arguments) {
            GitCommand result = gitResult(arguments);
            if (result.exitCode != 0) {
                throw new IllegalStateException(
                        "git " + String.join(" ", arguments)
                                + " failed: " + result.output);
            }
            return result.output.trim();
        }

        private GitCommand gitResult(String... arguments) {
            List<String> command = new ArrayList<>();
            command.add("git");
            command.add("-C");
            command.add(repository.toString());
            command.addAll(List.of(arguments));
            ProcessBuilder builder = new ProcessBuilder(command);
            builder.redirectErrorStream(true);
            try {
                Process process = builder.start();
                process.getOutputStream().close();
                ByteArrayOutputStream output = new ByteArrayOutputStream();
                process.getInputStream().transferTo(output);
                int exitCode = process.waitFor();
                return new GitCommand(
                        exitCode,
                        output.toString(StandardCharsets.UTF_8));
            } catch (IOException exception) {
                throw new IllegalStateException("cannot execute Git", exception);
            } catch (InterruptedException exception) {
                Thread.currentThread().interrupt();
                throw new IllegalStateException("Git interrupted", exception);
            }
        }

        @Override
        public void close() {
            deleteRecursively(root);
        }
    }

    private static final class MemoryPullRequests implements DraftPullRequests {
        private final Map<Identity, DraftPullRequest> entries = new HashMap<>();
        private boolean failNextCreate;

        @Override
        public Optional<DraftPullRequest> find(Identity identity) {
            return Optional.ofNullable(entries.get(identity));
        }

        @Override
        public void create(DraftPullRequest pullRequest) throws IOException {
            if (failNextCreate) {
                failNextCreate = false;
                throw new IOException("injected draft PR failure");
            }
            DraftPullRequest prior = entries.putIfAbsent(
                    pullRequest.identity(),
                    pullRequest);
            if (prior != null && !prior.equals(pullRequest)) {
                throw new IOException("conflicting draft PR");
            }
        }
    }

    @FunctionalInterface
    private interface CheckedOperation {
        void run() throws Exception;
    }

    private record CaseDefinition(String id, String expected) {
    }

    private record CaseResult(String id, String expected, String actual) {
    }

    private record BundleFile(Path root, Path manifest, String actionId) {
    }

    private record Snapshot(String head, String status) {
    }

    private record GitCommand(int exitCode, String output) {
    }
}
