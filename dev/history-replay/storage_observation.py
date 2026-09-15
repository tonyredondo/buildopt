"""Read storage diagnostics for the host and explicitly owned observers.

These counters describe work and waiting; they never relax quiet admission or
attribute host I/O to a project. Missing optional counters remain unavailable.
No process discovery, subprocesses, controller changes or privileged reads.
"""

import os
from pathlib import Path
import time


def boot_ns():
    return time.clock_gettime_ns(time.CLOCK_BOOTTIME)


def optional_text(path):
    try:
        return {"available": True, "value": path.read_text()}
    except OSError as error:
        return {"available": False, "errno": error.errno, "reason": error.strerror}


def host_storage(proc=Path("/proc")):
    start = boot_ns()
    memory = optional_text(proc / "meminfo")
    pending = memory
    if memory["available"]:
        try:
            values = {}
            for line in memory["value"].splitlines():
                fields = line.split()
                if fields[0] in ("Dirty:", "Writeback:"):
                    if len(fields) != 3 or fields[2] != "kB" or int(fields[1]) < 0:
                        raise ValueError("invalid pending-write counter")
                    key = fields[0][:-1]
                    if key in values:
                        raise ValueError("duplicate pending-write counter")
                    values[key] = int(fields[1])
            if set(values) != {"Dirty", "Writeback"}:
                raise ValueError("missing pending-write counter")
            pending = {"available": True, "unit": "KiB", "value": values}
        except (ValueError, IndexError) as error:
            pending = {"available": False, "reason": str(error)}
    devices = optional_text(proc / "diskstats")
    return {"beginBootNS": start, "endBootNS": boot_ns(),
            "deviceCounters": devices, "pendingWrites": pending}


def group_storage(group):
    start = boot_ns()
    pressure = optional_text(group / "io.pressure")
    counters = optional_text(group / "io.stat")
    return {"beginBootNS": start, "endBootNS": boot_ns(),
            "ioPressure": pressure, "ioCounters": counters}


def _identity(raw):
    fields = raw.rsplit(")", 1)[1].split()
    return {"startTicks": int(fields[19]), "cpuTicks": int(fields[11]) + int(fields[12])}


def observer_resources(pid=None, proc=Path("/proc")):
    """Read a supplied process only and reject PID reuse across the reads."""
    pid = os.getpid() if pid is None else pid
    start = boot_ns()
    path = proc / str(pid)
    result = {"pid": pid, "beginBootNS": start}
    try:
        before = _identity((path / "stat").read_text())
        io = optional_text(path / "io")
        after = _identity((path / "stat").read_text())
        if before["startTicks"] != after["startTicks"]:
            raise ValueError("observer identity changed during observation")
        result.update(available=True, identity=after, io=io,
                      ticksPerSecond=os.sysconf("SC_CLK_TCK"))
        if pid == os.getpid():
            result["selfCPUNS"] = time.process_time_ns()
    except (OSError, ValueError, IndexError) as error:
        result.update(available=False, reason=str(error))
    result["endBootNS"] = boot_ns()
    return result


def storage_sample(groups=(), observers=()):
    """Keep the collection interval and its own CPU cost around every read."""
    begin = boot_ns()
    before = observer_resources()
    host = host_storage()
    group_rows = {str(group): group_storage(Path(group)) for group in groups}
    observer_rows = {str(pid): observer_resources(pid) for pid in observers}
    after = observer_resources()
    return {"schema": "buildopt.storage-sample/v1", "beginBootNS": begin,
            "endBootNS": boot_ns(), "host": host, "groups": group_rows,
            "observers": observer_rows, "samplerBefore": before, "samplerAfter": after}
