package dev.buildopt.patcher;

import java.util.ArrayList;
import java.util.List;

import dev.buildopt.patcher.PostMergePatchMonitor.Action;
import dev.buildopt.patcher.PostMergePatchMonitor.Arm;
import dev.buildopt.patcher.PostMergePatchMonitor.Context;
import dev.buildopt.patcher.PostMergePatchMonitor.Decision;
import dev.buildopt.patcher.PostMergePatchMonitor.Observation;
import dev.buildopt.patcher.PostMergePatchMonitor.Outcome;
import dev.buildopt.patcher.PostMergePatchMonitor.Status;

/** Focused C4-009 post-merge classification conformance. */
final class PostMergePatchMonitorSpike {
    private static final String ARTIFACT = digest('a');
    private static final Context CONTEXT = new Context(
            "tenant-7/repo-42",
            "archive-action",
            "1".repeat(40),
            "epoch-9",
            digest('2'),
            "linux-amd64-4cpu-16gib",
            digest('3'));

    private PostMergePatchMonitorSpike() {
    }

    static void assertConformance() {
        Decision regression = PostMergePatchMonitor.evaluate(
                pairs(List.of(300L, 280L, 320L, 310L), List.of(100L, 100L, 110L, 90L)),
                true,
                true);
        require(regression.status() == Status.REGRESSION
                        && regression.action() == Action.CREATE_DRAFT_REVERT_PR
                        && regression.lower95Ms() > 0,
                "causal regression creates draft revert");

        Decision improvement = PostMergePatchMonitor.evaluate(
                pairs(List.of(80L, 90L, 70L, 85L), List.of(200L, 190L, 210L, 205L)),
                true,
                true);
        require(improvement.status() == Status.IMPROVEMENT
                        && improvement.action() == Action.NONE
                        && improvement.upper95Ms() < 0,
                "causal improvement remains active");

        Decision inconclusive = PostMergePatchMonitor.evaluate(
                pairs(List.of(90L, 110L, 90L, 110L), List.of(100L, 100L, 100L, 100L)),
                true,
                true);
        require(inconclusive.status() == Status.INCONCLUSIVE
                        && inconclusive.action() == Action.NONE,
                "crossing interval is inconclusive");

        Decision contextual = PostMergePatchMonitor.evaluate(
                pairs(List.of(300L, 280L, 320L, 310L), List.of(100L, 100L, 110L, 90L)),
                false,
                true);
        require(contextual.status() == Status.CONTEXTUAL
                        && contextual.action() == Action.NONE,
                "missing control budget is contextual");

        Decision noControl = PostMergePatchMonitor.evaluate(
                List.of(observation("unpaired", Arm.PATCHED_NATURAL, 300L, "natural")),
                true,
                true);
        require(noControl.status() == Status.CONTEXTUAL
                        && "NO_COMPARABLE_CONTROL".equals(noControl.reason()),
                "missing inverse observation is contextual");

        Decision instruction = PostMergePatchMonitor.evaluate(
                pairs(List.of(300L, 280L, 320L, 310L), List.of(100L, 100L, 110L, 90L)),
                true,
                false);
        require(instruction.status() == Status.CONTEXTUAL
                        && instruction.action() == Action.PRECISE_REVERT_INSTRUCTION,
                "non-invertible patch yields instruction");

        List<Observation> divergent = new ArrayList<>(
                pairs(List.of(300L, 280L, 320L, 310L), List.of(100L, 100L, 110L, 90L)));
        Observation changed = divergent.get(1);
        divergent.set(1, new Observation(
                changed.observationId(),
                changed.pairId(),
                changed.arm(),
                changed.context(),
                changed.isolationKey(),
                changed.outcome(),
                changed.customerVisibleBuildMs(),
                digest('b'),
                changed.inverseAppliedExactly()));
        Decision divergence = PostMergePatchMonitor.evaluate(divergent, true, true);
        require(divergence.status() == Status.REGRESSION
                        && divergence.action() == Action.CREATE_DRAFT_REVERT_PR
                        && "REQUIRED_ARTIFACT_DIVERGENCE".equals(divergence.reason()),
                "artifact divergence is non-compensable");

        List<Observation> failed = new ArrayList<>(
                pairs(List.of(300L, 280L, 320L, 310L), List.of(100L, 100L, 110L, 90L)));
        Observation patched = failed.get(0);
        failed.set(0, new Observation(
                patched.observationId(),
                patched.pairId(),
                patched.arm(),
                patched.context(),
                patched.isolationKey(),
                Outcome.BUILD_FAILURE,
                0,
                "",
                patched.inverseAppliedExactly()));
        Decision failure = PostMergePatchMonitor.evaluate(failed, true, true);
        require(failure.status() == Status.REGRESSION
                        && failure.action() == Action.CREATE_DRAFT_REVERT_PR,
                "patched build failure is non-compensable");

        List<Observation> cancelled = new ArrayList<>(
                pairs(List.of(300L, 280L, 320L, 310L), List.of(100L, 100L, 110L, 90L)));
        Observation cancelledPatched = cancelled.get(0);
        cancelled.set(0, new Observation(
                cancelledPatched.observationId(),
                cancelledPatched.pairId(),
                cancelledPatched.arm(),
                cancelledPatched.context(),
                cancelledPatched.isolationKey(),
                Outcome.CANCELLED,
                0,
                "",
                cancelledPatched.inverseAppliedExactly()));
        Decision cancellation = PostMergePatchMonitor.evaluate(cancelled, true, true);
        require(cancellation.status() == Status.REGRESSION
                        && cancellation.action() == Action.CREATE_DRAFT_REVERT_PR,
                "patched cancellation is non-compensable");

        List<Observation> inexact = new ArrayList<>(
                pairs(List.of(300L, 280L, 320L, 310L), List.of(100L, 100L, 110L, 90L)));
        Observation inverse = inexact.get(1);
        inexact.set(1, new Observation(
                inverse.observationId(),
                inverse.pairId(),
                inverse.arm(),
                inverse.context(),
                inverse.isolationKey(),
                inverse.outcome(),
                inverse.customerVisibleBuildMs(),
                inverse.requiredArtifactDigest(),
                false));
        Decision notExact = PostMergePatchMonitor.evaluate(inexact, true, true);
        require(notExact.status() == Status.CONTEXTUAL
                        && "INVERSE_NOT_EXACT".equals(notExact.reason()),
                "inexact control cannot be causal");

        List<Observation> insufficient = pairs(
                List.of(200L, 210L, 190L),
                List.of(100L, 100L, 100L));
        require(PostMergePatchMonitor.evaluate(insufficient, true, true).status()
                        == Status.INCONCLUSIVE,
                "fewer than four pairs are inconclusive");

        require(regression.equals(PostMergePatchMonitor.evaluate(
                        pairs(List.of(300L, 280L, 320L, 310L),
                                List.of(100L, 100L, 110L, 90L)),
                        true,
                        true)),
                "identical evidence yields an identical decision");
    }

    private static List<Observation> pairs(List<Long> patched, List<Long> inverse) {
        List<Observation> values = new ArrayList<>();
        for (int index = 0; index < patched.size(); index++) {
            String pair = "pair-" + (index + 1);
            values.add(observation(pair, Arm.PATCHED_NATURAL, patched.get(index), "natural"));
            values.add(observation(pair, Arm.INVERSE_CONTROL, inverse.get(index), "inverse"));
        }
        return values;
    }

    private static Observation observation(
            String pair,
            Arm arm,
            long duration,
            String isolationPrefix) {
        return new Observation(
                pair + "-" + arm.name().toLowerCase(java.util.Locale.ROOT),
                pair,
                arm,
                CONTEXT,
                isolationPrefix + "-" + pair,
                Outcome.SUCCESS,
                duration,
                ARTIFACT,
                true);
    }

    private static String digest(char value) {
        return "sha256:" + String.valueOf(value).repeat(64);
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }
}
