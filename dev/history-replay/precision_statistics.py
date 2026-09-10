"""Statistics for the prospectively registered BV-006 owner precision design.

The pilot only selects a confirmation size or stops. Confirmation data never
change that size. Both original point gates and both block intervals must pass.
Native provenance, ordering and outcomes must be checked by the caller first.
"""
import math
import random
import statistics

ARM_LIMIT_NS = 100_000_000
WRAPPER_LIMIT_NS = 10_000_000
BLOCK_SIZES = (4, 8)
RESAMPLES = 20_000
SEED = 4_347_446


def _differences(native, candidate, count=None):
    if len(native) != len(candidate) or (count is not None and len(native) != count):
        raise ValueError("Incomplete paired arm samples")
    if any(type(value) is not int for value in [*native, *candidate]):
        raise ValueError("Native differences must be integer nanoseconds")


def nearest_rank(values, percent):
    if not values or not 0 < percent <= 100:
        raise ValueError("Invalid nearest-rank population or percentile")
    return sorted(values)[math.ceil(len(values) * percent / 100) - 1]


def size_confirmation(native, candidate):
    _differences(native, candidate, 16)
    variances = [statistics.variance(values) for values in (native, candidate)]
    estimate = (math.pi / 2) * 2 * sum(variances) * (1.96 + 1.645) ** 2 / ARM_LIMIT_NS**2
    required = 8 * math.ceil(max(64, estimate) / 8)
    admitted = required <= 512
    return {
        "decision": "FREEZE_ONE_CONFIRMATION" if admitted else "INSUFFICIENT_PRECISION_NO_CONFIRMATION",
        "sampleVariancesNS2": variances,
        "unroundedRequiredPairs": estimate,
        "requiredPairsPerArm": required,
        "selectedPairsPerArm": required if admitted else None,
        "confirmationGradleStarts": 16 + 4 * required if admitted else 0,
        "readinessQualified": False,
    }


def confirmation_intervals(native, candidate):
    _differences(native, candidate)
    count = len(native)
    if count < 64 or count > 512 or count % 8:
        raise ValueError("Confirmation count is outside the frozen design")
    intervals = []
    for block_size in BLOCK_SIZES:
        rng = random.Random(SEED)
        resampled = []
        for _ in range(RESAMPLES):
            indices = []
            for _ in range(count // block_size):
                start = rng.randrange(count)
                indices.extend((start + offset) % count for offset in range(block_size))
            difference = nearest_rank([native[index] for index in indices], 50) - nearest_rank(
                [candidate[index] for index in indices], 50
            )
            resampled.append(difference)
        low, high = nearest_rank(resampled, 2.5), nearest_rank(resampled, 97.5)
        intervals.append({
            "blockSize": block_size, "replicates": RESAMPLES, "seed": SEED,
            "lowerNS": low, "upperNS": high,
            "withinEnvelope": low >= -ARM_LIMIT_NS and high <= ARM_LIMIT_NS,
        })
    return intervals


def assess_confirmation(native, candidate, wrappers, frozen_pairs):
    _differences(native, candidate, frozen_pairs)
    if len(wrappers) != 4 * frozen_pairs or any(type(value) is not int or value < 0 for value in wrappers):
        raise ValueError("Incomplete or invalid wrapper measurements")
    signed_difference = nearest_rank(native, 50) - nearest_rank(candidate, 50)
    wrapper_p95 = nearest_rank(wrappers, 95)
    intervals = confirmation_intervals(native, candidate)
    point_pass = abs(signed_difference) <= ARM_LIMIT_NS and wrapper_p95 <= WRAPPER_LIMIT_NS
    interval_pass = all(interval["withinEnvelope"] for interval in intervals)
    qualified = point_pass and interval_pass
    return {
        "decision": "QUALIFIED_OWNER" if qualified else "INSUFFICIENT_READINESS",
        "readinessQualified": qualified, "signedArmDifferenceNS": signed_difference,
        "armImbalanceNS": abs(signed_difference), "wrapperP95NS": wrapper_p95,
        "pointGatesPass": point_pass, "intervalGatesPass": interval_pass,
        "intervals": intervals,
    }
