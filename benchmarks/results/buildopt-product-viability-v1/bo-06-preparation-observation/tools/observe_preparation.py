"""Observe one preparation pass, with no workflow invocation or retries."""
from pathlib import Path
import datetime
import hashlib
import json
import os
import shutil
import signal
import subprocess
import threading
import time
from storage_observation import storage_sample, observer_resources
from host_probe import processes

ROOT = Path(__file__).resolve().parent.parent
PROTOCOL = json.loads((ROOT / 'inputs/observation-protocol.json').read_text())
stop_reason = None


def boot_ns():
    return time.clock_gettime_ns(time.CLOCK_BOOTTIME)


def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def save(name, value):
    path = ROOT / name
    temp = path.with_suffix('.writing')
    temp.write_text(json.dumps(value, indent=2) + '\n')
    temp.replace(path)


def pressure(boot):
    begin = boot_ns()
    raw = {key: Path('/proc/pressure', key).read_text() for key in ('cpu', 'io', 'memory')}
    totals = {}
    for key, value in raw.items():
        line = next(line for line in value.splitlines() if line.startswith('some '))
        totals[key] = int(dict(field.split('=') for field in line.split()[1:])['total'])
    return {'begin': {'boot': boot, 'ns': begin}, 'end': {'boot': boot, 'ns': boot_ns()},
            'totalsUS': totals, 'raw': raw}


def process_state(pid):
    try:
        raw = Path('/proc', str(pid), 'stat').read_text()
        fields = raw.rsplit(')', 1)[1].split()
        result = {'pid': pid, 'startTicks': int(fields[19]), 'state': fields[0],
                  'children': Path('/proc', str(pid), 'task', str(pid), 'children').read_text().strip()}
        try:
            result['waitChannel'] = Path('/proc', str(pid), 'wchan').read_text().strip()
        except OSError as error:
            result['waitChannelUnavailable'] = error.errno
        return result
    except (FileNotFoundError, ProcessLookupError):
        return None


def register_worktrees():
    state = json.loads((ROOT / 'task-state.json').read_text())
    original = json.loads((ROOT / 'inputs/original-manifest.json').read_text())
    records = []
    for arm in ('N', 'I'):
        repo = ROOT / 'run/r1' / arm / 'repo'
        record = {'repository': str(repo), 'commonGit': original['commonGit'], 'arm': arm,
                  'expectedSHA': original['history'][0]['commit'], 'status': 'not observed',
                  'remoteTarget': 'local subject worktree; no publication', 'locator': str(ROOT / 'task-state.json')}
        if (repo / '.git').is_file():
            metadata = Path((repo / '.git').read_text().strip().removeprefix('gitdir: '))
            assert metadata.is_relative_to(Path(original['commonGit']) / 'worktrees')
            head = (metadata / 'HEAD').read_text().strip()
            assert head.startswith('ref: refs/heads/bv-replay-')
            branch = head.removeprefix('ref: refs/heads/')
            sha = (Path(original['commonGit']) / 'refs/heads' / branch).read_text().strip()
            assert sha == record['expectedSHA']
            record.update(status='verified', branch=branch, head=sha)
        records.append(record)
    if state.get('worktrees') != records:
        state['worktrees'] = records
        save('task-state.json', state)
    return records


def main():
    global stop_reason
    for item in json.loads((ROOT / 'inputs/diagnostic-freeze.json').read_text())['files']:
        assert hashlib.sha256((ROOT / item['path']).read_bytes()).hexdigest() == item['sha256'], item['path']
    assert not (ROOT / 'run').exists() and not (ROOT / 'receipts/start.json').exists()
    os.sched_setaffinity(0, {PROTOCOL['observerCPU']})
    start = boot_ns()
    boot = Path('/proc/sys/kernel/random/boot_id').read_text().strip()
    free_start = shutil.disk_usage(ROOT).free
    assert free_start >= PROTOCOL['minimumFreeBytes']
    config = {'schema': 'buildopt.preparation-observation/config/v1',
              'original': {'path': str(ROOT / 'inputs/original-manifest.json'), 'sha256': hashlib.sha256((ROOT / 'inputs/original-manifest.json').read_bytes()).hexdigest()},
              'runRoot': str(ROOT / 'run'), 'deadlineUTC': (datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(seconds=PROTOCOL['maxObservationSeconds'])).isoformat(),
              'maxRunNS': PROTOCOL['maxObservationSeconds'] * 10**9,
              'maxBytes': 16 * 2**30, 'minimumFreeBytes': PROTOCOL['minimumFreeBytes']}
    save('inputs/diagnostic-config.json', config)
    identity = process_state(os.getpid())
    save('receipts/start.json', {'atUTC': utc(), 'bootNS': start, 'observer': identity, 'freeBytes': free_start, 'protocol': PROTOCOL})
    state = json.loads((ROOT / 'task-state.json').read_text())
    state['runningProcesses'] = [{'role': 'observer', **identity}]
    state['steps']['one preparation observation'] = 'in progress'
    save('task-state.json', state)
    samples, sample_lines, trace_lines, process_checks = [], [], [], []
    trace_bytes = 0
    sample_bytes = 0
    stream_error = []
    child = None
    reader = None
    sampler_begin = observer_resources()
    phase = 'baseline'
    boundaries = {'baselineBeginNS': start}
    child_exit = None
    completed = False

    def interrupt(signum, frame):
        global stop_reason
        stop_reason = f'signal {signum}'
    signal.signal(signal.SIGTERM, interrupt)
    signal.signal(signal.SIGINT, interrupt)

    def read_trace():
        nonlocal trace_bytes
        try:
            for line in iter(child.stdout.readline, b''):
                if len(line) > 1024*1024:
                    raise RuntimeError('oversized trace row')
                json.loads(line)
                trace_bytes += len(line)
                if trace_bytes + sample_bytes > PROTOCOL['maxDiagnosticBytes']:
                    raise RuntimeError('diagnostic byte limit')
                trace_lines.append(line)
        except BaseException as error:
            stream_error.append(repr(error))

    stderr = (ROOT / 'logs/preparation-stderr.txt').open('xb')
    try:
        next_sample = start
        while True:
            time.sleep(max(0, (next_sample - boot_ns())/1e9))
            now = boot_ns()
            if stop_reason or stream_error:
                raise RuntimeError(stop_reason or stream_error[0])
            if now - start >= PROTOCOL['maxObservationSeconds'] * 10**9:
                raise RuntimeError('observation time limit')
            free = shutil.disk_usage(ROOT).free
            if free < PROTOCOL['minimumFreeBytes'] or free_start-free > PROTOCOL['maxFreeSpaceDecreaseBytes']:
                raise RuntimeError('free-space limit')
            row = {'sequence': len(samples), 'phase': phase, 'pressure': pressure(boot),
                   'storage': storage_sample(observers=[] if child is None or child.poll() is not None else [child.pid]),
                   'process': None if child is None else process_state(child.pid), 'freeBytes': free}
            line = (json.dumps(row, separators=(',', ':'))+'\n').encode()
            sample_bytes += len(line)
            if sample_bytes + trace_bytes + os.fstat(stderr.fileno()).st_size > PROTOCOL['maxDiagnosticBytes']:
                raise RuntimeError('diagnostic byte limit')
            samples.append(row); sample_lines.append(line)
            if len(samples) % 10 == 1:
                process_checks.append(processes())
                register_worktrees()
            if phase == 'baseline' and now-start >= PROTOCOL['baselineSeconds']*10**9:
                boundaries['baselineEndNS'] = boot_ns()
                boundaries['preparationLaunchNS'] = boot_ns()
                child = subprocess.Popen([str(ROOT / 'preparation-observer'), 'prepare-only', str(ROOT / 'inputs/diagnostic-config.json')], cwd=ROOT.parents[3], stdout=subprocess.PIPE, stderr=stderr, start_new_session=True)
                child_identity = process_state(child.pid)
                save('receipts/preparation-start.json', {'atUTC': utc(), 'bootNS': boundaries['preparationLaunchNS'], 'process': child_identity})
                state = json.loads((ROOT / 'task-state.json').read_text())
                state['runningProcesses'].append({'role': 'preparation', **child_identity})
                save('task-state.json', state)
                reader = threading.Thread(target=read_trace, daemon=True); reader.start()
                phase = 'preparation'
                print('Preparation started after the complete baseline.', flush=True)
            elif phase == 'preparation' and child.poll() is not None:
                child_exit = child.wait()
                boundaries['preparationExitObservedNS'] = boot_ns()
                boundaries['recoveryBeginNS'] = boot_ns()
                phase = 'recovery'
                print(json.dumps({'preparationExitCode': child_exit, 'recoveryStarted': True}), flush=True)
            elif phase == 'recovery' and now-boundaries['recoveryBeginNS'] >= PROTOCOL['recoverySeconds']*10**9:
                boundaries['recoveryEndNS'] = boot_ns()
                completed = True
                break
            if len(samples) % 60 == 0:
                print(json.dumps({'phase': phase, 'samples': len(samples), 'elapsedSeconds': round((boot_ns()-start)/1e9, 1)}), flush=True)
            next_sample = max(next_sample+10**9, boot_ns())
    except BaseException as error:
        stop_reason = repr(error)
    finally:
        if child is not None and child.poll() is None:
            os.killpg(child.pid, signal.SIGTERM)
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL); child.wait(timeout=5)
        if reader is not None:
            reader.join(timeout=5)
            if reader.is_alive():
                stop_reason = 'trace reader did not close'
        stderr.close()
        if child is not None:
            child_exit = child.returncode
        finish = boot_ns()
        sampler_end = observer_resources()
        (ROOT / 'samples.jsonl').write_bytes(b''.join(sample_lines))
        (ROOT / 'trace.jsonl').write_bytes(b''.join(trace_lines))
        save('process-checks.json', process_checks)
        worktrees = register_worktrees()
        result = {'status': 'COMPLETE' if completed and child_exit == 0 and not stop_reason and not stream_error else 'INCOMPLETE',
                  'startedBootNS': start, 'endedBootNS': finish, 'endedUTC': utc(), 'boundaries': boundaries,
                  'preparationExitCode': child_exit, 'error': stop_reason, 'streamErrors': stream_error,
                  'samples': len(samples), 'traceRows': len(trace_lines), 'diagnosticBytes': sample_bytes+trace_bytes,
                  'observerBefore': sampler_begin, 'observerAfter': sampler_end, 'freeBytesBefore': free_start,
                  'freeBytesAfter': shutil.disk_usage(ROOT).free, 'worktrees': worktrees,
                  'projectBuilds': 0, 'comparisonJVMs': 0, 'retries': 0}
        save('receipts/end.json', result)
        state = json.loads((ROOT / 'task-state.json').read_text())
        state['runningProcesses'] = []
        state['observationStatus'] = result['status']
        save('task-state.json', state)
        print(json.dumps({k: result[k] for k in ('status','preparationExitCode','samples','traceRows','error')}), flush=True)
        if result['status'] != 'COMPLETE':
            raise SystemExit(1)


if __name__ == '__main__':
    main()
