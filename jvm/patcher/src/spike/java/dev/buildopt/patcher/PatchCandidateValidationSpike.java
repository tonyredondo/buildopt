package dev.buildopt.patcher;

import java.io.ByteArrayOutputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

import dev.buildopt.patcher.PatchCandidateValidator.Arm;
import dev.buildopt.patcher.PatchCandidateValidator.Artifact;
import dev.buildopt.patcher.PatchCandidateValidator.ArtifactAdapter;
import dev.buildopt.patcher.PatchCandidateValidator.Context;
import dev.buildopt.patcher.PatchCandidateValidator.Phase;
import dev.buildopt.patcher.PatchCandidateValidator.Request;
import dev.buildopt.patcher.PatchCandidateValidator.Result;
import dev.buildopt.patcher.PatchCandidateValidator.Run;
import dev.buildopt.patcher.PatchCandidateValidator.Status;

/** Focused C4-005 conformance cases for the production validator. */
final class PatchCandidateValidationSpike {
    private static final List<String> REQUIRED =
            List.of("build/distributions/app.zip", "build/reports/manifest.txt");
    private static final byte[] REPORT =
            "artifact manifest v1\n".getBytes(StandardCharsets.UTF_8);

    private PatchCandidateValidationSpike() {
    }

    static void assertConformance() throws Exception {
        byte[] candidate = archive(0L, false, "payload\n");
        byte[] controlClean = archive(1700000000000L, true, "payload\n");
        byte[] controlIncremental = archive(1700000004000L, false, "payload\n");
        byte[] controlRelocated = archive(1700000008000L, true, "payload\n");

        List<Run> passingRuns = runs(
                candidate,
                candidate,
                candidate,
                controlClean,
                controlIncremental,
                controlRelocated);
        Result passed = PatchCandidateValidator.validate(request(passingRuns));
        require(passed.status() == Status.PASSED
                        && "PASSED".equals(passed.reason())
                        && passed.candidateArtifactSetDigest() != null
                        && passed.controlArtifactSetDigest() != null
                        && passed.logicalArtifactSetDigest() != null
                        && !passed.candidateArtifactSetDigest().equals(
                                passed.controlArtifactSetDigest()),
                "candidate/control archive validation");

        List<Run> nonReproducible = new ArrayList<>(passingRuns);
        replace(
                nonReproducible,
                Arm.CANDIDATE,
                Phase.INCREMENTAL,
                copy(
                        find(nonReproducible, Arm.CANDIDATE, Phase.INCREMENTAL),
                        artifacts(archive(1700000012000L, true, "payload\n"))));
        requireResult(
                request(nonReproducible),
                Status.FAILED,
                "CANDIDATE_NOT_REPRODUCIBLE");

        List<Run> divergent = new ArrayList<>(passingRuns);
        replace(
                divergent,
                Arm.CANDIDATE,
                Phase.RELOCATED,
                copy(
                        find(divergent, Arm.CANDIDATE, Phase.RELOCATED),
                        artifacts(archive(0L, false, "different\n"))));
        requireResult(request(divergent), Status.FAILED, "ARTIFACT_DIVERGENCE");

        List<Run> missingRun = new ArrayList<>(passingRuns);
        missingRun.remove(missingRun.size() - 1);
        requireResult(request(missingRun), Status.INCONCLUSIVE, "MISSING_RUN");

        List<Run> sharedIsolation = new ArrayList<>(passingRuns);
        Run last = sharedIsolation.get(sharedIsolation.size() - 1);
        sharedIsolation.set(
                sharedIsolation.size() - 1,
                new Run(
                        last.arm(),
                        last.phase(),
                        last.context(),
                        passingRuns.get(0).isolationKey(),
                        last.configurationCacheState(),
                        last.exitCode(),
                        last.artifacts()));
        requireResult(
                request(sharedIsolation),
                Status.INCONCLUSIVE,
                "ISOLATION_NOT_DISTINCT");

        List<Run> contextDrift = new ArrayList<>(passingRuns);
        Run drifted = contextDrift.get(contextDrift.size() - 1);
        Context context = drifted.context();
        Context otherPolicy = new Context(
                context.repositoryId(),
                context.actionId(),
                context.revision(),
                context.sourceStateDigest(),
                context.workUnitsFingerprint(),
                context.requiredDeliverablesManifestDigest(),
                digest('9'),
                context.toolchainId(),
                context.runnerClass());
        contextDrift.set(
                contextDrift.size() - 1,
                new Run(
                        drifted.arm(),
                        drifted.phase(),
                        otherPolicy,
                        drifted.isolationKey(),
                        drifted.configurationCacheState(),
                        drifted.exitCode(),
                        drifted.artifacts()));
        requireResult(request(contextDrift), Status.INCONCLUSIVE, "CONTEXT_DRIFT");

        List<Run> cacheFailure = new ArrayList<>(passingRuns);
        Run incremental = find(cacheFailure, Arm.CANDIDATE, Phase.INCREMENTAL);
        replace(
                cacheFailure,
                Arm.CANDIDATE,
                Phase.INCREMENTAL,
                new Run(
                        incremental.arm(),
                        incremental.phase(),
                        incremental.context(),
                        incremental.isolationKey(),
                        "DISABLED",
                        incremental.exitCode(),
                        incremental.artifacts()));
        requireResult(
                request(cacheFailure),
                Status.FAILED,
                "CONFIGURATION_CACHE_FAILED");

        List<Run> armFailure = new ArrayList<>(passingRuns);
        Run failed = find(armFailure, Arm.CONTROL, Phase.CLEAN);
        replace(
                armFailure,
                Arm.CONTROL,
                Phase.CLEAN,
                new Run(
                        failed.arm(),
                        failed.phase(),
                        failed.context(),
                        failed.isolationKey(),
                        failed.configurationCacheState(),
                        1,
                        failed.artifacts()));
        requireResult(request(armFailure), Status.FAILED, "ARM_FAILED");

        List<Run> missingArtifact = new ArrayList<>(passingRuns);
        Run incomplete = find(missingArtifact, Arm.CANDIDATE, Phase.CLEAN);
        replace(
                missingArtifact,
                Arm.CANDIDATE,
                Phase.CLEAN,
                copy(incomplete, List.of(incomplete.artifacts().get(0))));
        requireResult(
                request(missingArtifact),
                Status.FAILED,
                "INVALID_ARTIFACT");

        List<Run> unsafeArchive = new ArrayList<>(passingRuns);
        Run unsafe = find(unsafeArchive, Arm.CANDIDATE, Phase.CLEAN);
        replace(
                unsafeArchive,
                Arm.CANDIDATE,
                Phase.CLEAN,
                copy(unsafe, artifacts(archiveWithUnsafeEntry())));
        requireResult(request(unsafeArchive), Status.FAILED, "INVALID_ARTIFACT");


        List<Run> exactRuns = runs(
                candidate,
                candidate,
                candidate,
                candidate,
                candidate,
                candidate);
        Request exactAdapter = new Request(
                "CUSTOM_TASK_CONTRACT_JAVA_V1",
                "1.0",
                ArtifactAdapter.EXACT_BYTES,
                REQUIRED,
                exactRuns);
        requireResult(exactAdapter, Status.PASSED, "PASSED");

        Request groovyRecipe = new Request(
                ArchiveReproducibilityGroovyDslRecipe.RECIPE_ID,
                ArchiveReproducibilityGroovyDslRecipe.RECIPE_VERSION,
                ArtifactAdapter.ARCHIVE_CONTENTS_V1,
                REQUIRED,
                passingRuns);
        requireResult(groovyRecipe, Status.PASSED, "PASSED");

        Request buildCacheRecipe = new Request(
                GradleBuildCachePropertiesRecipe.RECIPE_ID,
                GradleBuildCachePropertiesRecipe.RECIPE_VERSION,
                ArtifactAdapter.ARCHIVE_CONTENTS_V1,
                REQUIRED,
                passingRuns);
        requireResult(buildCacheRecipe, Status.PASSED, "PASSED");

        Request wrongAdapter = new Request(
                ArchiveReproducibilityRecipe.RECIPE_ID,
                ArchiveReproducibilityRecipe.RECIPE_VERSION,
                ArtifactAdapter.EXACT_BYTES,
                REQUIRED,
                passingRuns);
        requireResult(wrongAdapter, Status.INCONCLUSIVE, "INVALID_REQUEST");

        byte[] defensive = candidate.clone();
        List<Artifact> copied = artifacts(defensive);
        defensive[0] ^= 1;
        require(
                !Arrays.equals(defensive, copied.get(0).content()),
                "artifact bytes are defensive");
    }

    private static List<Run> runs(
            byte[] candidateClean,
            byte[] candidateIncremental,
            byte[] candidateRelocated,
            byte[] controlClean,
            byte[] controlIncremental,
            byte[] controlRelocated) {
        Context context = context();
        return List.of(
                run(Arm.CANDIDATE, Phase.CLEAN, context, candidateClean),
                run(Arm.CONTROL, Phase.CLEAN, context, controlClean),
                run(Arm.CANDIDATE, Phase.INCREMENTAL, context, candidateIncremental),
                run(Arm.CONTROL, Phase.INCREMENTAL, context, controlIncremental),
                run(Arm.CANDIDATE, Phase.RELOCATED, context, candidateRelocated),
                run(Arm.CONTROL, Phase.RELOCATED, context, controlRelocated));
    }

    private static Run run(Arm arm, Phase phase, Context context, byte[] archive) {
        String cacheState = phase == Phase.INCREMENTAL ? "REUSED" : "STORED";
        return new Run(
                arm,
                phase,
                context,
                arm.name().toLowerCase(java.util.Locale.ROOT)
                        + "-"
                        + phase.name().toLowerCase(java.util.Locale.ROOT),
                cacheState,
                0,
                artifacts(archive));
    }

    private static List<Artifact> artifacts(byte[] archive) {
        return List.of(
                new Artifact("build/distributions/app.zip", archive),
                new Artifact("build/reports/manifest.txt", REPORT));
    }

    private static Request request(List<Run> runs) {
        return new Request(
                ArchiveReproducibilityRecipe.RECIPE_ID,
                ArchiveReproducibilityRecipe.RECIPE_VERSION,
                ArtifactAdapter.ARCHIVE_CONTENTS_V1,
                REQUIRED,
                runs);
    }

    private static Context context() {
        return new Context(
                "tenant-7/repo-42",
                "archive-validation",
                "a".repeat(40),
                digest('1'),
                digest('2'),
                digest('3'),
                digest('4'),
                "temurin-21-gradle-9.6.1",
                "linux-amd64-4cpu-16gib");
    }

    private static String digest(char value) {
        return "sha256:" + String.valueOf(value).repeat(64);
    }

    private static Run copy(Run run, List<Artifact> artifacts) {
        return new Run(
                run.arm(),
                run.phase(),
                run.context(),
                run.isolationKey(),
                run.configurationCacheState(),
                run.exitCode(),
                artifacts);
    }

    private static Run find(List<Run> runs, Arm arm, Phase phase) {
        for (Run run : runs) {
            if (run.arm() == arm && run.phase() == phase) {
                return run;
            }
        }
        throw new AssertionError("missing fixture run");
    }

    private static void replace(List<Run> runs, Arm arm, Phase phase, Run replacement) {
        for (int index = 0; index < runs.size(); index++) {
            Run run = runs.get(index);
            if (run.arm() == arm && run.phase() == phase) {
                runs.set(index, replacement);
                return;
            }
        }
        throw new AssertionError("missing fixture run");
    }

    private static byte[] archive(long timestamp, boolean reverse, String payload)
            throws Exception {
        List<Entry> entries = new ArrayList<>(List.of(
                new Entry("META-INF/MANIFEST.MF", "Manifest-Version: 1.0\n"),
                new Entry("data/value.txt", payload)));
        if (reverse) {
            java.util.Collections.reverse(entries);
        }
        return writeArchive(entries, timestamp);
    }

    private static byte[] archiveWithUnsafeEntry() throws Exception {
        return writeArchive(
                List.of(new Entry("../escape.txt", "unsafe\n")),
                0L);
    }

    private static byte[] writeArchive(List<Entry> entries, long timestamp)
            throws Exception {
        ByteArrayOutputStream output = new ByteArrayOutputStream();
        try (ZipOutputStream archive = new ZipOutputStream(output)) {
            for (Entry entry : entries) {
                ZipEntry zipEntry = new ZipEntry(entry.path());
                zipEntry.setTime(timestamp);
                archive.putNextEntry(zipEntry);
                archive.write(entry.content().getBytes(StandardCharsets.UTF_8));
                archive.closeEntry();
            }
        }
        return output.toByteArray();
    }

    private static void requireResult(Request request, Status status, String reason) {
        Result result = PatchCandidateValidator.validate(request);
        require(result.status() == status && reason.equals(result.reason()),
                "expected " + status + "/" + reason
                        + ", got " + result.status() + "/" + result.reason());
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    private record Entry(String path, String content) {
    }
}
