"""Counter contracts; temporary proc files and live self-only observations."""

import importlib.util
import os
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("storage", Path(__file__).with_name("storage_observation.py"))
storage = importlib.util.module_from_spec(spec)
spec.loader.exec_module(storage)


class StorageObservationTests(unittest.TestCase):
    def test_pending_writes_keep_units_and_optional_fields(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "meminfo").write_text("Dirty: 12 kB\nWriteback: 7 kB\n")
            (root / "diskstats").write_text("259 0 nvme0n1 1 2 3 4\n")
            row = storage.host_storage(root)
            self.assertEqual(row["pendingWrites"], {"available": True, "unit": "KiB", "value": {"Dirty": 12, "Writeback": 7}})
            self.assertIn("nvme0n1", row["deviceCounters"]["value"])
            self.assertLessEqual(row["beginBootNS"], row["endBootNS"])
            (root / "diskstats").unlink()
            self.assertFalse(storage.host_storage(root)["deviceCounters"]["available"])

    def test_missing_or_malformed_is_not_zero(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for raw in ("Dirty: 1 kB\n", "Dirty: -1 kB\nWriteback: 0 kB\n", "Dirty: 1 bytes\nWriteback: 0 kB\n", "\n"):
                (root / "meminfo").write_text(raw)
                pending = storage.host_storage(root)["pendingWrites"]
                self.assertFalse(pending["available"])
                self.assertNotIn("value", pending)
            row = storage.group_storage(root)
            self.assertFalse(row["ioPressure"]["available"])
            self.assertFalse(row["ioCounters"]["available"])
            observer = storage.observer_resources(123, root)
            self.assertFalse(observer["available"])

    def test_group_counters_and_pressure(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "io.pressure").write_text("some avg10=0.00 total=17\n")
            (root / "io.stat").write_text("259:0 rbytes=7 wbytes=12\n")
            row = storage.group_storage(root)
            self.assertEqual(row["ioPressure"]["value"], "some avg10=0.00 total=17\n")
            self.assertEqual(row["ioCounters"]["value"], "259:0 rbytes=7 wbytes=12\n")

    @unittest.skipUnless(Path("/proc/self/io").exists(), "Linux process counters")
    def test_live_cost_covers_all_reads(self):
        row = storage.storage_sample(observers=[os.getpid()])
        before, after = row["samplerBefore"], row["samplerAfter"]
        self.assertTrue(before["available"])
        self.assertTrue(after["available"])
        self.assertEqual(before["identity"]["startTicks"], after["identity"]["startTicks"])
        self.assertGreaterEqual(after["selfCPUNS"], before["selfCPUNS"])
        self.assertLessEqual(row["beginBootNS"], row["host"]["beginBootNS"])
        self.assertGreaterEqual(row["endBootNS"], after["endBootNS"])
        self.assertEqual(set(row["observers"]), {str(os.getpid())})


if __name__ == "__main__":
    unittest.main()
