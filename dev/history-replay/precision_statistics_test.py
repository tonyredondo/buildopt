"""Behavioral rejection checks for the owner precision design; no native starts."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("precision_statistics", Path(__file__).with_name("precision_statistics.py"))
precision = importlib.util.module_from_spec(spec)
spec.loader.exec_module(precision)


class OwnerPrecisionTest(unittest.TestCase):
    def test_pilot_cannot_qualify_even_with_perfect_balance(self):
        result = precision.size_confirmation([0] * 16, [0] * 16)
        self.assertEqual(result["selectedPairsPerArm"], 64)
        self.assertFalse(result["readinessQualified"])

    def test_size_uses_variance_not_observed_arm_effect(self):
        native = [index * 5_000_000 for index in range(16)]
        candidate = list(reversed(native))
        original = precision.size_confirmation(native, candidate)
        shifted = precision.size_confirmation([value + 9_000_000_000 for value in native], candidate)
        self.assertEqual(original, shifted)

    def test_no_small_confirmation_when_required_count_exceeds_limit(self):
        result = precision.size_confirmation([0, 5_000_000_000] * 8, [0, 5_000_000_000] * 8)
        self.assertEqual(result["decision"], "INSUFFICIENT_PRECISION_NO_CONFIRMATION")
        self.assertIsNone(result["selectedPairsPerArm"])
        self.assertEqual(result["confirmationGradleStarts"], 0)

    def test_incomplete_or_non_integer_pilot_rejected(self):
        for native in ([0] * 15, [0.0] * 16, [False] * 16):
            with self.assertRaises(ValueError):
                precision.size_confirmation(native, [0] * 16)

    def test_point_gate_alone_cannot_admit_noisy_balanced_medians(self):
        native = [-1_000_000_000] * 16 + [0] * 32 + [1_000_000_000] * 16
        candidate = [0] * 64
        result = precision.assess_confirmation(native, candidate, [0] * 256, 64)
        self.assertTrue(result["pointGatesPass"])
        self.assertFalse(result["intervalGatesPass"])
        self.assertFalse(result["readinessQualified"])

    def test_exact_envelope_is_inclusive_but_next_nanosecond_fails(self):
        for effect, expected in ((100_000_000, True), (100_000_001, False)):
            result = precision.assess_confirmation([effect] * 64, [0] * 64, [10_000_000] * 256, 64)
            self.assertEqual(result["readinessQualified"], expected)

    def test_wrapper_limit_still_rejects_perfect_arm_balance(self):
        result = precision.assess_confirmation([0] * 64, [0] * 64, [10_000_001] * 256, 64)
        self.assertTrue(result["intervalGatesPass"])
        self.assertFalse(result["readinessQualified"])

    def test_joint_blocks_preserve_shared_temporal_noise(self):
        native = [index * 1_000_000_000 for index in range(64)]
        candidate = [value - 20_000_000 for value in native]
        result = precision.assess_confirmation(native, candidate, [0] * 256, 64)
        self.assertTrue(result["readinessQualified"])
        self.assertEqual([(row["lowerNS"], row["upperNS"]) for row in result["intervals"]], [(20_000_000, 20_000_000)] * 2)

    def test_confirmation_cannot_resize_or_lose_wrappers(self):
        for native, candidate, wrappers, count in (([0] * 64, [0] * 64, [0] * 256, 72),
                                                   ([0] * 64, [0] * 64, [0] * 255, 64)):
            with self.assertRaises(ValueError):
                precision.assess_confirmation(native, candidate, wrappers, count)


if __name__ == "__main__":
    unittest.main()
