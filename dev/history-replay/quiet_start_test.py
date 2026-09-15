"""Exercise admission decisions without starting a build or waiting in real time."""
import copy
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("quiet_start", Path(__file__).with_name("quiet_start.py"))
quiet = importlib.util.module_from_spec(spec)
spec.loader.exec_module(quiet)


def samples(count=31, pressure=0.0):
    return [{"startNS": i * 1_000_000_000, "endNS": i * 1_000_000_000,
             "totalUS": {key: int(i * pressure * 1_000_000) for key in quiet.RESOURCES}}
            for i in range(count)]


class QuietStartTests(unittest.TestCase):
    def test_full_window_required(self):
        self.assertFalse(quiet.assess([])["ready"])
        self.assertFalse(quiet.assess(samples(30))["ready"])
        self.assertTrue(quiet.assess(samples())["ready"])

    def test_threshold_boundary(self):
        self.assertTrue(quiet.assess(samples(pressure=0.1))["ready"])
        self.assertFalse(quiet.assess(samples(pressure=0.100001))["ready"])

    def test_single_recent_burst_blocks_each_resource(self):
        for resource in quiet.RESOURCES:
            with self.subTest(resource=resource):
                rows = samples()
                rows[-1]["totalUS"][resource] = 900_000
                self.assertFalse(quiet.assess(rows)["ready"])

    def test_old_burst_can_be_followed_by_complete_quiet_window(self):
        rows = samples(62)
        for row in rows[1:]:
            row["totalUS"]["io"] = 900_000
        self.assertTrue(quiet.assess(rows)["ready"])
        self.assertEqual(len(rows), 62)

    def test_delayed_read_and_gap_are_not_quiet(self):
        rows = samples()
        rows[-1]["startNS"] -= 100_000_001
        self.assertFalse(quiet.assess(rows)["ready"])
        rows = samples()
        del rows[10:14]
        self.assertFalse(quiet.assess(rows)["ready"])

    def test_reset_and_overlapping_read_are_not_quiet(self):
        rows = samples(pressure=0.01)
        rows[-1]["totalUS"]["io"] = 0
        self.assertFalse(quiet.assess(rows)["ready"])
        rows = samples()
        rows[15], rows[16] = rows[16], rows[15]
        self.assertFalse(quiet.assess(rows)["ready"])

    def test_assessment_preserves_input(self):
        rows = samples()
        before = copy.deepcopy(rows)
        quiet.assess(rows)
        self.assertEqual(rows, before)

    def test_observer_ready_timeout_and_failure(self):
        for pressure, expected in [(0.0, "QUIET_START"), (0.3, "NOT_QUIET_TIMEOUT")]:
            with self.subTest(pressure=pressure):
                now = [0]
                def sleep(seconds):
                    now[0] += int(seconds * 1e9)
                def read():
                    return {"startNS": now[0], "endNS": now[0],
                            "totalUS": {key: int(now[0] / 1000 * pressure) for key in quiet.RESOURCES}}
                result = quiet.observe(35, read, lambda: now[0], sleep)
                self.assertEqual(result["decision"], expected)
                self.assertGreaterEqual(len(result["samples"]), 31)
        def failed_read():
            raise PermissionError("pressure unavailable")
        self.assertEqual(quiet.observe(35, failed_read)["decision"], "OBSERVATION_FAILED")

    def test_deadline_and_cancellation_do_not_admit(self):
        now = [0]
        def late_read():
            now[0] = 36_000_000_000
            return samples()[0]
        self.assertEqual(quiet.observe(35, late_read, lambda: now[0])["decision"], "NOT_QUIET_TIMEOUT")
        def cancelled_read():
            raise KeyboardInterrupt()
        self.assertEqual(quiet.observe(35, cancelled_read)["decision"], "INTERRUPTED")


if __name__ == "__main__":
    unittest.main()
