"""Join closed native requests, exact phase records, clocks and sampled resources."""
from pathlib import Path
import collections
import datetime
import hashlib
import json
import os
import re

D = Path(__file__).resolve().parent
PROFILE = D / 'profiles/supported'
RUN = PROFILE / 'run'

def read(path):
    return json.loads(path.read_text())

def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        for block in iter(lambda: f.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()

def bound(binding):
    p = Path(binding['path'])
    assert digest(p) == binding['sha256'], p
    return read(p)

def epoch_ns(text):
    # Microsecond datetime rounding is far below one-second process sampling.
    return int(datetime.datetime.fromisoformat(text.replace('Z', '+00:00')).timestamp() * 1e9)

def iso(value):
    return datetime.datetime.fromtimestamp(value / 1e9, datetime.timezone.utc).isoformat().replace('+00:00', 'Z')

samples = [json.loads(line) for line in (PROFILE / 'process-samples.jsonl').read_text().splitlines()]
assert samples
offsets = [s['epochNS'] - s['monotonicNS'] for s in samples]
hz = os.sysconf('SC_CLK_TCK')
rows = []
windows = []
for attempt in sorted((RUN / 'attempts').iterdir()):
    if not attempt.is_dir():
        continue
    start = read(attempt / 'start.json')
    end = read(attempt / 'end.json')
    native = bound(end['native'])
    capture = bound(end['capture'])
    assert native['exitCode'] == 0 and end['class'] == 'COMPARABLE'
    selected = [s for s in samples if native['start']['ns'] <= s['endBootNS'] <= native['end']['ns']]
    assert selected
    near = min(samples, key=lambda s: abs(s['endBootNS'] - native['start']['ns']))
    boot_offset = near['epochNS'] - near['endBootNS']
    begin_epoch = native['start']['ns'] + boot_offset
    end_epoch = native['end']['ns'] + boot_offset
    name = attempt.name
    windows.append((name, iso(begin_epoch), iso(end_epoch)))
    tasks = {}
    for b in capture['graph']:
        p = Path(b['path'])
        assert digest(p) == b['sha256']
        for line in p.read_text().splitlines():
            event = json.loads(line)
            if event['kind'] == 'task':
                task = dict(event['task'])
                task.update(startUTC=event['startedUTC'], endUTC=event['utc'],
                            seconds=(epoch_ns(event['utc']) - epoch_ns(event['startedUTC'])) / 1e9)
                assert task['identity'] not in tasks
                tasks[task['identity']] = task
    phases = []
    for log in ('stdout.log', 'stderr.log'):
        assert digest(attempt / log) == native[log.split('.')[0] + 'SHA256']
        for line in (attempt / log).read_text(errors='replace').splitlines():
            assert '[BUILDOPT-REG-ERROR]' not in line
            if '[BUILDOPT-REG] phase=' not in line:
                continue
            phase = dict(re.findall(r'(\w+)=([^\s]+)', line))
            for key in ('pid', 'startNs', 'endNs', 'elapsedNs', 'exitOpcode', 'inputCount', 'reused', 'hashed'):
                if key in phase:
                    phase[key] = int(phase[key])
            assert phase['elapsedNs'] == phase['endNs'] - phase['startNs']
            clock = min(selected, key=lambda s: abs(s['monotonicNS'] - phase['startNs']))
            offset = clock['epochNS'] - clock['monotonicNS']
            phase.update(startUTC=iso(phase['startNs'] + offset), endUTC=iso(phase['endNs'] + offset),
                         startRelativeSeconds=(phase['startNs'] + offset - begin_epoch) / 1e9,
                         endRelativeSeconds=(phase['endNs'] + offset - begin_epoch) / 1e9)
            phases.append(phase)
    resources = {}
    bins = collections.defaultdict(lambda: collections.defaultdict(float))
    for sample in selected:
        for group in sample['groups']:
            if group['unit'] != native['unit']:
                continue
            for p in group['processes']:
                ident = p['identity']
                key = f"{ident['pid']}:{ident['startTicks']}"
                command = p['command']
                role = ('daemon' if 'GradleDaemon' in command else 'checkstyle-worker' if any(
                    phase['pid'] == ident['pid'] and 'Checker.process' in phase['phase'] for phase in phases)
                    else 'other-worker' if 'GradleWorkerMain' in command else 'supervisor' if 'history-replay' in command
                    else 'wrapper' if 'GradleWrapperMain' in command else 'other')
                value = resources.setdefault(key, dict(pid=ident['pid'], startTicks=ident['startTicks'], role=role,
                    first=ident, last=ident, firstIO=p['io'], lastIO=p['io'], firstBootNS=sample['endBootNS'],
                    lastBootNS=sample['endBootNS'], samples=0, states=collections.Counter()))
                delta = max(0, ident['cpuTicks'] - value['last']['cpuTicks']) / hz
                bins[int((sample['endBootNS'] - native['start']['ns']) // (5 * 10**9))][role] += delta
                value.update(last=ident, lastIO=p['io'], lastBootNS=sample['endBootNS'], samples=value['samples'] + 1)
                value['states'][ident['state']] += 1
    for value in resources.values():
        value['observedCPUSeconds'] = (value['last']['cpuTicks'] - value['first']['cpuTicks']) / hz
        value['observedReadBytes'] = value['lastIO']['read_bytes'] - value['firstIO']['read_bytes']
        value['observedWriteBytes'] = value['lastIO']['write_bytes'] - value['firstIO']['write_bytes']
        value['observedMajorFaults'] = value['last']['majorFaults'] - value['first']['majorFaults']
    adapter_tasks = []
    for phase in phases:
        if phase['phase'].endswith('ContentAwareChecker.process'):
            task = ':server:' + Path(phase['state']).name
            commit = [x for x in phases if x.get('task') == task and x['phase'].endswith('$Commit.execute')]
            prepare = [x for x in phases if x.get('task') == task and x['phase'].endswith('$Prepare.execute')]
            assert len(commit) == len(prepare) == 1
            inner = [x for x in phases if x['pid'] == phase['pid'] and x['phase'] == 'com/puppycrawl/tools/checkstyle/Checker.process'
                     and phase['startNs'] <= x['startNs'] <= x['endNs'] <= phase['endNs']]
            assert len(inner) == 1
            assert inner[0]['inputCount'] + phase['reused'] == phase['inputCount']
            adapter_tasks.append(dict(task=task, inputFiles=phase['inputCount'], reusedFiles=phase['reused'],
                processedFiles=inner[0]['inputCount'], adapterSeconds=phase['elapsedNs'] / 1e9,
                engineSeconds=inner[0]['elapsedNs'] / 1e9, prepareSeconds=prepare[0]['elapsedNs'] / 1e9,
                commitSeconds=commit[0]['elapsedNs'] / 1e9,
                afterAdapterBeforeCommitSeconds=(commit[0]['startNs'] - phase['endNs']) / 1e9,
                afterAdapterBeforeTaskCompletionSeconds=(epoch_ns(tasks[task]['endUTC']) - epoch_ns(phase['endUTC'])) / 1e9,
                commitStartAfterTaskCompletionSeconds=(epoch_ns(commit[0]['startUTC']) - epoch_ns(tasks[task]['endUTC'])) / 1e9,
                taskSeconds=tasks[task]['seconds'], workerPID=phase['pid'],
                adapterEndUTC=phase['endUTC'], commitStartUTC=commit[0]['startUTC']))
    row = dict(attempt=name, replication=start['replication'], originalOrdinal=start['ordinal'] + 17, arm=start['arm'],
               nativeSeconds=(native['end']['ns'] - native['start']['ns']) / 1e9,
               customerEnvelopeSeconds=end['durationNS'] / 1e9, startUTC=iso(begin_epoch), endUTC=iso(end_epoch),
               tasks=tasks, phases=phases, adapterTasks=adapter_tasks, resources=resources, fiveSecondCPUBins=dict(bins))
    rows.append(row)
assert len(rows) == 16
checker_tasks = {':server:checkstyleMain', ':server:checkstyleTest', ':server:checkstyleInternalClusterTest'}
for row in rows:
    checking = sorted(task for task, value in row['tasks'].items() if value['action'] and task in checker_tasks)
    engines = [phase for phase in row['phases'] if phase['phase'] == 'com/puppycrawl/tools/checkstyle/Checker.process']
    assert len(engines) == len(checking), ('ENGINE_PHASE_COVERAGE', row['attempt'], len(engines), checking)
    row['engineCheckedFiles'] = sum(phase['inputCount'] for phase in engines)
    if row['arm'] == 'N':
        assert not row['adapterTasks'], ('NATIVE_ARM_ADAPTER', row['attempt'])
    else:
        assert sorted(task['task'] for task in row['adapterTasks']) == checking
        for task in row['adapterTasks']:
            contexts = [phase for phase in row['phases'] if phase['phase'].endswith('CheckstyleSuccessHistory.context') and phase.get('task') == task['task']]
            assert len(contexts) == 2, ('CONTEXT_PHASE_COVERAGE', row['attempt'], task['task'])
            task['contextSeconds'] = sum(phase['elapsedNs'] for phase in contexts)/1e9
result = dict(status='verified', actualWorkCountsVerified=True, attempts=rows,
    clockOffsetSpreadNS=max(offsets)-min(offsets),
    limits=['Per-phase and sampled CPU durations do not add up to whole-build elapsed time',
            'Retain all phases, idle gaps and host observations; no causal conclusion from timing alone'])
(D / 'analysis/diagnostic-phases.json').write_text(json.dumps(result, indent=2) + '\n')
(D / 'analysis/windows.tsv').write_text(''.join('\t'.join(window) + '\n' for window in windows))
for row in rows:
    print(row['attempt'], 'native seconds', round(row['nativeSeconds'], 3))
    for task in row['adapterTasks']:
        print(task)
