"""One bounded host-only observation; no builds, process control or disk walk."""
from pathlib import Path
import datetime
import hashlib
import json
import os
import time

ROOT = Path(__file__).resolve().parent
POLICY_SHA = '1c1d2080051f44929de62b0e0447fa1ccd63e490f8762c5c6619b33c5e2d22bb'
OLD_PIDS = {2695349: 38698413, 2696683: 38709345, 2696684: 38709349}
OLD_CGROUP = Path('/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/buildopt-replay-5dc9f073af206d2d-r1-N-1.service')


def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def boot_ns():
    return time.clock_gettime_ns(time.CLOCK_BOOTTIME)


def processes():
    indexers, old, unavailable = [], [], []
    for entry in Path('/proc').iterdir():
        if not entry.name.isdigit():
            continue
        try:
            raw = (entry / 'stat').read_text()
        except FileNotFoundError:
            continue
        except OSError as error:
            unavailable.append({'pid': int(entry.name), 'errno': error.errno})
            continue
        comm = raw.split('(', 1)[1].rsplit(')', 1)[0]
        fields = raw.rsplit(')', 1)[1].split()
        row = {'pid': int(entry.name), 'name': comm, 'startTicks': int(fields[19]), 'state': fields[0]}
        if comm in ('updatedb', 'updatedb.mlocate', 'updatedb.plocat') or comm.startswith('updatedb.'):
            indexers.append(row)
        if int(entry.name) in OLD_PIDS:
            row['sameIdentity'] = int(fields[19]) == OLD_PIDS[int(entry.name)]
            old.append(row)
    return {'atUTC': utc(), 'indexers': indexers, 'oldProcesses': old,
            'unavailable': unavailable, 'oldCgroupExists': OLD_CGROUP.exists()}


def storage():
    fields = {}
    for line in Path('/proc/meminfo').read_text().splitlines():
        if line.split(':')[0] in ('Dirty', 'Writeback'):
            fields[line.split(':')[0]] = line.split(':')[1].strip()
    return {'atUTC': utc(), 'pendingWrites': fields,
            'diskstats': Path('/proc/diskstats').read_text()}


def main():
    assert hashlib.sha256((ROOT / 'policy.json').read_bytes()).hexdigest() == POLICY_SHA
    allocation = json.loads((ROOT / 'allocation.json').read_text())
    assert hashlib.sha256(Path(__file__).read_bytes()).hexdigest() == allocation['observerSHA256']
    before = processes()
    assert not before['indexers'] and not before['unavailable']
    assert not any(row['sameIdentity'] for row in before['oldProcesses'])
    assert not before['oldCgroupExists']
    boot_id = Path('/proc/sys/kernel/random/boot_id').read_text().strip()
    start, cpu_start = boot_ns(), time.process_time_ns()
    record = {'schema': 'buildopt.host-only-observation/v1', 'startedUTC': utc(),
              'boot': boot_id, 'startBootNS': start, 'samples': [],
              'processChecks': [before], 'storageBefore': storage()}
    stat = Path('/proc/self/stat').read_text().rsplit(')', 1)[1].split()
    (ROOT / 'process.json').write_text(json.dumps({'pid': os.getpid(), 'startTicks': int(stat[19]), 'atUTC': utc()})+'\n')
    try:
        for index in range(allocation['sampleCount']):
            target = start + index * allocation['sampleNS']
            time.sleep(max(0, (target - boot_ns()) / 1e9))
            if boot_ns() - start > allocation['hardMaximumNS']:
                raise RuntimeError('observation deadline exceeded')
            begin = boot_ns()
            totals, raw = {}, {}
            for resource in ('cpu', 'io', 'memory'):
                raw[resource] = Path('/proc/pressure', resource).read_text()
                line = next(line for line in raw[resource].splitlines() if line.startswith('some '))
                totals[resource] = int(dict(field.split('=') for field in line.split()[1:])['total'])
            end = boot_ns()
            record['samples'].append({'begin': {'boot': boot_id, 'ns': begin},
                                      'end': {'boot': boot_id, 'ns': end},
                                      'totalsUS': totals, 'raw': raw})
            if index and index % 10 == 0:
                record['processChecks'].append(processes())
            if index and index % 60 == 0:
                print(json.dumps({'samples': len(record['samples']), 'elapsedSeconds': (end-start)/1e9}), flush=True)
        time.sleep(max(0, (start + allocation['durationNS'] - boot_ns()) / 1e9))
        assert Path('/proc/sys/kernel/random/boot_id').read_text().strip() == boot_id
        record['status'] = 'COMPLETE'
    except BaseException as error:
        record['status'] = 'INCOMPLETE'
        record['error'] = repr(error)
        raise
    finally:
        record['endBootNS'] = boot_ns()
        record['endedUTC'] = utc()
        record['observerCPUSeconds'] = (time.process_time_ns()-cpu_start)/1e9
        record['storageAfter'] = storage()
        record['processChecks'].append(processes())
        data = json.dumps(record, indent=2)+'\n'
        assert len(data.encode()) < allocation['maximumDiagnosticBytes']
        (ROOT / 'observation.json').write_text(data)
        print(json.dumps({'status': record['status'], 'samples': len(record['samples']),
                          'seconds': (record['endBootNS']-start)/1e9,
                          'observerCPUSeconds': record['observerCPUSeconds']}), flush=True)


if __name__ == '__main__':
    main()
