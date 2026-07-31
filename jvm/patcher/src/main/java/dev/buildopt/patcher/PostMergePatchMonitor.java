package dev.buildopt.patcher;

import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;
import java.util.Set;
import java.util.SplittableRandom;
import java.util.regex.Pattern;

/** Fail-closed post-merge classifier for exact inverse-patch controls. */
public final class PostMergePatchMonitor {
    private static final int BOOTSTRAP_REPLICATES = 4096;
    private static final int MINIMUM_EXACT_PAIRS = 4;
    private static final Pattern IDENTIFIER = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$");
    private static final Pattern DIGEST = Pattern.compile(
            "^(?:sha256|hmac-sha256):[0-9a-f]{64}$");
    private static final Pattern GIT_OBJECT = Pattern.compile("^(?:[0-9a-f]{40}|[0-9a-f]{64})$");

    private PostMergePatchMonitor() {
    }

    /** Observation treatment. */
    public enum Arm {
        PATCHED_NATURAL,
        INVERSE_CONTROL
    }

    /** Retained build outcome. */
    public enum Outcome {
        SUCCESS,
        BUILD_FAILURE,
        INFRA_FAILURE,
        CANCELLED
    }

    /** Evidence classification. */
    public enum Status {
        IMPROVEMENT,
        REGRESSION,
        CONTEXTUAL,
        INCONCLUSIVE
    }

    /** Safe follow-up selected by the classifier. */
    public enum Action {
        NONE,
        CREATE_DRAFT_REVERT_PR,
        PRECISE_REVERT_INSTRUCTION
    }

    /** Immutable context that must be identical across every retained observation. */
    public record Context(
            String repositoryId,
            String actionId,
            String mergedRevision,
            String measurementEpoch,
            String workUnitsFingerprint,
            String runnerClass,
            String policyDigest) {
    }

    /** One natural or inverse-control observation. */
    public record Observation(
            String observationId,
            String pairId,
            Arm arm,
            Context context,
            String isolationKey,
            Outcome outcome,
            long customerVisibleBuildMs,
            String requiredArtifactDigest,
            boolean inverseAppliedExactly) {
    }

    /** Terminal post-merge decision. */
    public record Decision(
            Status status,
            Action action,
            String reason,
            int assignedPairs,
            int analyzedPairs,
            long meanPatchedMinusInverseMs,
            long lower95Ms,
            long upper95Ms,
            long patchedMinusInverseP95Ms) {
    }

    /**
     * Evaluates retained natural builds and budgeted exact inverse controls.
     * Only exact comparable evidence can request a draft revert PR.
     */
    public static Decision evaluate(
            List<Observation> observations,
            boolean controlBudgetAuthorized,
            boolean exactRevertBundleAvailable) {
        if (observations == null || observations.isEmpty() || observations.size() > 4096) {
            return inconclusive("NO_OBSERVATIONS", 0);
        }
        Context context = observations.get(0) == null
                ? null
                : observations.get(0).context();
        if (!validContext(context)) {
            return inconclusive("INVALID_CONTEXT", 0);
        }

        Set<String> observationIds = new HashSet<>();
        Set<String> isolationKeys = new HashSet<>();
        Map<String, Pair> pairs = new TreeMap<>();
        int naturalCount = 0;
        int inverseCount = 0;
        for (Observation observation : observations) {
            if (!validObservation(observation, context)
                    || !observationIds.add(observation.observationId())
                    || !isolationKeys.add(observation.isolationKey())) {
                return inconclusive("INVALID_OR_DUPLICATE_OBSERVATION", pairs.size());
            }
            Pair pair = pairs.computeIfAbsent(observation.pairId(), ignored -> new Pair());
            if (observation.arm() == Arm.PATCHED_NATURAL) {
                naturalCount++;
                if (pair.patched != null) {
                    return inconclusive("DUPLICATE_PAIR_ARM", pairs.size());
                }
                pair.patched = observation;
            } else {
                inverseCount++;
                if (pair.inverse != null) {
                    return inconclusive("DUPLICATE_PAIR_ARM", pairs.size());
                }
                pair.inverse = observation;
            }
        }
        if (naturalCount == 0) {
            return inconclusive("NO_NATURAL_BUILD", pairs.size());
        }
        if (inverseCount == 0) {
            return contextual("NO_COMPARABLE_CONTROL", pairs.size());
        }
        if (!controlBudgetAuthorized) {
            return contextual("CONTROL_BUDGET_UNAVAILABLE", pairs.size());
        }
        if (!exactRevertBundleAvailable) {
            return new Decision(
                    Status.CONTEXTUAL,
                    Action.PRECISE_REVERT_INSTRUCTION,
                    "EXACT_REVERT_UNAVAILABLE",
                    pairs.size(),
                    0,
                    0,
                    0,
                    0,
                    0);
        }

        List<Long> deltas = new ArrayList<>();
        List<Long> patchedDurations = new ArrayList<>();
        List<Long> inverseDurations = new ArrayList<>();
        for (Map.Entry<String, Pair> entry : pairs.entrySet()) {
            Pair pair = entry.getValue();
            if (pair.patched == null || pair.inverse == null) {
                continue;
            }
            Observation patched = pair.patched;
            Observation inverse = pair.inverse;
            if (!inverse.inverseAppliedExactly()) {
                return contextual("INVERSE_NOT_EXACT", pairs.size());
            }
            if ((patched.outcome() == Outcome.BUILD_FAILURE
                            || patched.outcome() == Outcome.CANCELLED)
                    && inverse.outcome() == Outcome.SUCCESS) {
                return regression(
                        patched.outcome() == Outcome.CANCELLED
                                ? "PATCHED_BUILD_CANCELLED"
                                : "PATCHED_BUILD_FAILURE",
                        pairs.size(),
                        deltas.size());
            }
            if (patched.outcome() != Outcome.SUCCESS
                    || inverse.outcome() != Outcome.SUCCESS) {
                continue;
            }
            if (!patched.requiredArtifactDigest().equals(
                    inverse.requiredArtifactDigest())) {
                return regression("REQUIRED_ARTIFACT_DIVERGENCE", pairs.size(), deltas.size());
            }
            deltas.add(patched.customerVisibleBuildMs() - inverse.customerVisibleBuildMs());
            patchedDurations.add(patched.customerVisibleBuildMs());
            inverseDurations.add(inverse.customerVisibleBuildMs());
        }
        if (deltas.size() < MINIMUM_EXACT_PAIRS) {
            return inconclusive("INSUFFICIENT_EXACT_PAIRS", pairs.size());
        }

        long mean = roundedMean(deltas);
        long[] interval = bootstrapInterval(context, deltas);
        long p95Delta = nearestRank(patchedDurations) - nearestRank(inverseDurations);
        if (interval[0] > 0 || (p95Delta > 0 && mean > 0)) {
            return new Decision(
                    Status.REGRESSION,
                    Action.CREATE_DRAFT_REVERT_PR,
                    "CAUSAL_REGRESSION",
                    pairs.size(),
                    deltas.size(),
                    mean,
                    interval[0],
                    interval[1],
                    p95Delta);
        }
        if (interval[1] < 0 && p95Delta <= 0) {
            return new Decision(
                    Status.IMPROVEMENT,
                    Action.NONE,
                    "CAUSAL_IMPROVEMENT",
                    pairs.size(),
                    deltas.size(),
                    mean,
                    interval[0],
                    interval[1],
                    p95Delta);
        }
        return new Decision(
                Status.INCONCLUSIVE,
                Action.NONE,
                "INTERVAL_CROSSES_ZERO",
                pairs.size(),
                deltas.size(),
                mean,
                interval[0],
                interval[1],
                p95Delta);
    }

    private static boolean validContext(Context context) {
        return context != null
                && validIdentifier(context.repositoryId())
                && validIdentifier(context.actionId())
                && context.mergedRevision() != null
                && GIT_OBJECT.matcher(context.mergedRevision()).matches()
                && validIdentifier(context.measurementEpoch())
                && validDigest(context.workUnitsFingerprint())
                && validIdentifier(context.runnerClass())
                && validDigest(context.policyDigest());
    }

    private static boolean validObservation(Observation observation, Context context) {
        if (observation == null
                || !context.equals(observation.context())
                || !validIdentifier(observation.observationId())
                || !validIdentifier(observation.pairId())
                || observation.arm() == null
                || !validIdentifier(observation.isolationKey())
                || observation.outcome() == null) {
            return false;
        }
        if (observation.outcome() == Outcome.SUCCESS) {
            return observation.customerVisibleBuildMs() > 0
                    && observation.customerVisibleBuildMs() <= 86400000L
                    && validDigest(observation.requiredArtifactDigest());
        }
        return observation.customerVisibleBuildMs() >= 0
                && observation.customerVisibleBuildMs() <= 86400000L
                && (observation.requiredArtifactDigest() == null
                        || observation.requiredArtifactDigest().isEmpty());
    }

    private static long[] bootstrapInterval(Context context, List<Long> deltas) {
        long[] means = new long[BOOTSTRAP_REPLICATES];
        SplittableRandom random = new SplittableRandom(seed(context, deltas));
        for (int replicate = 0; replicate < BOOTSTRAP_REPLICATES; replicate++) {
            long total = 0;
            for (int index = 0; index < deltas.size(); index++) {
                total += deltas.get(random.nextInt(deltas.size()));
            }
            means[replicate] = Math.round((double) total / deltas.size());
        }
        java.util.Arrays.sort(means);
        return new long[] {
            means[(int) Math.floor(0.025 * (means.length - 1))],
            means[(int) Math.ceil(0.975 * (means.length - 1))]
        };
    }

    private static long seed(Context context, List<Long> deltas) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            digest.update("buildopt-post-merge-bootstrap-v1".getBytes(StandardCharsets.UTF_8));
            digest.update(StrictJson.canonicalBytes(Map.of(
                    "context", Map.of(
                            "repositoryId", context.repositoryId(),
                            "actionId", context.actionId(),
                            "mergedRevision", context.mergedRevision(),
                            "measurementEpoch", context.measurementEpoch(),
                            "workUnitsFingerprint", context.workUnitsFingerprint(),
                            "runnerClass", context.runnerClass(),
                            "policyDigest", context.policyDigest()),
                    "deltas", deltas)));
            byte[] value = digest.digest();
            long seed = 0;
            for (int index = 0; index < Long.BYTES; index++) {
                seed = (seed << 8) | Byte.toUnsignedLong(value[index]);
            }
            return seed;
        } catch (GeneralSecurityException exception) {
            throw new IllegalStateException("SHA-256 is unavailable", exception);
        }
    }

    private static long roundedMean(List<Long> values) {
        long total = 0;
        for (long value : values) {
            total += value;
        }
        return Math.round((double) total / values.size());
    }

    private static long nearestRank(List<Long> values) {
        List<Long> sorted = new ArrayList<>(values);
        sorted.sort(Comparator.naturalOrder());
        return sorted.get(Math.max(0, (int) Math.ceil(0.95 * sorted.size()) - 1));
    }

    private static Decision contextual(String reason, int pairs) {
        return new Decision(Status.CONTEXTUAL, Action.NONE, reason, pairs, 0, 0, 0, 0, 0);
    }

    private static Decision inconclusive(String reason, int pairs) {
        return new Decision(Status.INCONCLUSIVE, Action.NONE, reason, pairs, 0, 0, 0, 0, 0);
    }

    private static Decision regression(String reason, int pairs, int analyzed) {
        return new Decision(
                Status.REGRESSION,
                Action.CREATE_DRAFT_REVERT_PR,
                reason,
                pairs,
                analyzed,
                0,
                0,
                0,
                0);
    }

    private static boolean validIdentifier(String value) {
        return value != null && IDENTIFIER.matcher(value).matches();
    }

    private static boolean validDigest(String value) {
        return value != null && DIGEST.matcher(value).matches();
    }

    private static final class Pair {
        private Observation patched;
        private Observation inverse;
    }
}
