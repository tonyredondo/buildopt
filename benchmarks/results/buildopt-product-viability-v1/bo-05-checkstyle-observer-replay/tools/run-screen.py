"""Run the frozen native-versus-V2 17-to-20 screen; retain every sample."""
from pathlib import Path
import datetime, hashlib, importlib.util, json, os, shutil, signal, subprocess, sys, time
sys.dont_write_bytecode = True

P = Path(__file__).resolve().parent
R = P.parents[3]
I = P.parent / 'bv006-cpu-isolation'
F = P.parent / 'bv006-checkstyle-finalization'
def load(f): return json.loads(Path(f).read_text())
def sha(f): return hashlib.sha256(Path(f).read_bytes()).hexdigest()
def now(): return datetime.datetime.now(datetime.timezone.utc).isoformat()
def save(f, x):
    assert not f.exists(), f
    f.write_text(json.dumps(x, indent=2) + '\n')
def atomic(f, x):
    tmp = f.with_suffix(f.suffix + '.tmp')
    tmp.write_text(json.dumps(x, indent=2) + '\n')
    tmp.replace(f)
def stat(pid):
    raw = Path(f'/proc/{pid}/stat').read_text()
    fields = raw.rsplit(')', 1)[1].split()
    return dict(pid=pid, startTicks=int(fields[19]), cpuTicks=int(fields[11]) + int(fields[12]), state=fields[0], minorFaults=int(fields[7]), majorFaults=int(fields[9]), rssPages=int(fields[21]))
label = 'supported'
cpus = 8
spec = importlib.util.spec_from_file_location('bindings', I / 'bindings.py')
bindings = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bindings)
for binding in load(P / 'inputs/launch-freeze.json'):
    assert bindings.bind(binding['path']) == binding, binding['path']
assert load(F / 'analysis/preliminary-controller.json')['actualOwnerStarts'] == 0
assert load(F / 'receipts/owner-runner-final-proof.json')['status'].startswith('verified')
assert load(F / 'analysis/correctness-v2.json')['status'].startswith('verified')
assert load(F / 'receipts/agent-proof-v2-verified.json')['exitCode'] == 0
assert load(F / 'receipts/agent-proof-v2-verified.json')['contextEvents']
profile = P / 'profiles' / label
run = profile / 'run'
assert not run.exists()
m = load(profile / 'manifest.json')
allocation = load(P / 'allocation.json')
starts = 2 * len(m['history']) * m['replications']
offset = 17
assert starts == 16 and allocation['ownerReserved'] == allocation['actualOwnerStarts'] == 0
assert 'controlBaseline' not in m
assert m['replications'] == 2 and m['executionEnd'] == 3 and m['limits']['retryPairs'] == 0
assert load(P / 'receipts/readiness.json')['status'] == 'verified for exploratory screen'
assert allocation['actualHelpers'] + starts <= allocation['maxStandaloneHelpers']
allocation['ownerReserved'] += starts
assert allocation['ownerReserved'] <= allocation['maxOwnerControllerReservations']
assert datetime.datetime.now(datetime.timezone.utc) < datetime.datetime.fromisoformat(allocation['deadlineUTC'])
assert shutil.disk_usage(P).free >= allocation['minimumFreeBytes']
atomic(P / 'allocation.json', allocation)
os.sched_setaffinity(0, {9})
assert os.sched_getaffinity(0) == {9}
command = [m['executable']['path'], 'run', str(profile / 'manifest.json')]
receipt = dict(profile=cpus, command=command, cwd=str(R), startUTC=now(), startBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME), gradleReserved=starts, helperJVMReserved=starts, driverAndSamplerAffinity=[9])
logpath = P / f'logs/profile-{label}.log'
samplepath = profile / 'process-samples.jsonl'
statepath = P / 'task-state.json'
groups = {}
completed = set()
identities = {}
samples = 0

def inspect_worktrees():
    for rep in range(1, m['replications'] + 1):
        for arm in ('N', 'I'):
            repo = run / f'r{rep}' / arm / 'repo'
            if not (repo / '.git').exists(): continue
            def git(*args): return subprocess.check_output(['git', '-C', str(repo), *args], text=True).strip()
            head = git('rev-parse', 'HEAD')
            key = f'r{rep}-{arm}'
            if identities.get(key, {}).get('head') == head: continue
            record = dict(repository=str(repo), branch=git('branch', '--show-current'), head=head,
                          commonGit=git('rev-parse', '--path-format=absolute', '--git-common-dir'),
                          remoteTarget='read-only subject history; no publication', locator=str(statepath),
                          arm=arm, replication=rep, profile=cpus)
            assert head in {row['commit'] for row in m['history']} and record['commonGit'] == m['commonGit']
            identities[key] = record
            state = load(statepath)
            state['worktrees'] = list(identities.values())
            atomic(statepath, state)
            atomic(profile / 'worktree-identities.json', list(identities.values()))

def host_observation():
    data = {}
    for line in Path('/proc/stat').read_text().splitlines():
        fields = line.split()
        if fields[0].startswith('cpu'):
            data[fields[0]] = list(map(int, fields[1:9]))
        elif fields[0] in ('procs_running', 'procs_blocked'):
            data[fields[0]] = int(fields[1])
    return dict(cpu=data, loadavg=Path('/proc/loadavg').read_text().strip(),
                pressure={name:Path('/proc/pressure',name).read_text() for name in ('cpu','io','memory')},
                ticksPerSecond=os.sysconf('SC_CLK_TCK'))

def observe(output):
    global samples
    begin = time.clock_gettime_ns(time.CLOCK_BOOTTIME)
    for configpath in (run / 'sessions').glob('*/worker-config.json'):
        cfg = load(configpath)
        assert cfg['nativeAffinity'] == m['affinity'] and cfg['observerAffinity'] == '8'
        if cfg['unit'] not in groups:
            response = subprocess.run(['systemctl', '--user', 'show', cfg['unit'], '--property=ControlGroup', '--value'], capture_output=True, text=True)
            group = response.stdout.strip()
            if group:
                assert group.endswith('/' + cfg['unit'])
                groups[cfg['unit']] = dict(path=group, config=cfg)
    group_rows = []
    for unit, group in groups.items():
        cgroup = Path('/sys/fs/cgroup') / group['path'].lstrip('/')
        if not cgroup.exists(): continue
        processes = []
        procs = [cgroup / 'cgroup.procs', *cgroup.glob('**/cgroup.procs')]
        pids = set()
        for f in procs:
            try: pids.update(map(int, f.read_text().split()))
            except FileNotFoundError: pass
        for pid in sorted(pids):
            try:
                identity = stat(pid)
                commandline = Path(f'/proc/{pid}/cmdline').read_bytes().replace(b'\0', b' ').decode(errors='replace')
                masks = {}
                for thread in Path(f'/proc/{pid}/task').iterdir():
                    try:
                        mask = ','.join(map(str, sorted(os.sched_getaffinity(int(thread.name)))))
                        masks.setdefault(mask, []).append(int(thread.name))
                    except ProcessLookupError: pass
                assert stat(pid)['startTicks'] == identity['startTicks']
                io = {}
                for line in Path(f'/proc/{pid}/io').read_text().splitlines():
                    key, value = line.split(':', 1); io[key] = int(value)
                wait = Path(f'/proc/{pid}/wchan').read_text().strip()
                processes.append(dict(identity=identity, command=commandline, threadMasks=masks, io=io, mainThreadWaitChannel=wait))
            except (FileNotFoundError, ProcessLookupError): pass
        try:
            cpu_usage = int(dict(line.split() for line in (cgroup / 'cpu.stat').read_text().splitlines())['usage_usec'])
        except FileNotFoundError:
            continue
        group_rows.append(dict(unit=unit, cgroup=group['path'], processes=processes, cpuUsageUS=cpu_usage))
    if True:
        output.write(json.dumps(dict(beginBootNS=begin, endBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME), monotonicNS=time.monotonic_ns(), epochNS=time.time_ns(), host=host_observation(), groups=group_rows)) + '\n')
        output.flush()
        samples += 1
    for nativepath in (run / 'attempts').glob('*/native-finish.json'):
        attempt = nativepath.parent
        if attempt.name in completed: continue
        native = load(nativepath)
        start = load(attempt / 'start.json')
        arm = start['arm']
        if native['exitCode'] == 0 and start['ordinal'] == 0:
            logs = '\n'.join(f.read_text(errors='replace') for f in attempt.glob('native.*.log'))
            # Exact files are resolved below; do not accept absence of worker diagnostics.
            if not logs:
                logs = '\n'.join(f.read_text(errors='replace') for f in attempt.glob('*') if f.is_file() and ('stdout' in f.name or 'stderr' in f.name))
            assert '[BUILDOPT-REG-ERROR]' not in logs, ('AGENT_ERROR', attempt.name)
            assert 'phase=com/puppycrawl/tools/checkstyle/Checker.process' in logs, ('WORKER_CAPTURE_MISSING', attempt.name)
            if arm == 'I':
                assert 'ContentAwareChecker.process' in logs and 'CheckstyleSuccessHistory$Commit.execute' in logs and 'CheckstyleSuccessHistory.context' in logs, ('CANDIDATE_PHASE_CAPTURE_MISSING', attempt.name)
        state = run / f'r{start["replication"]}' / arm / 'repo/.gradle/buildopt-checkstyle'
        dest = profile / 'candidate-state' / attempt.name
        dest.mkdir(parents=True)
        entries = []
        for f in sorted(state.glob('**/success')):
            data = f.read_bytes()
            relative = f.relative_to(state)
            copied = dest / relative
            copied.parent.mkdir(parents=True, exist_ok=True)
            copied.write_bytes(data)
            entries.append(dict(path='repo/.gradle/buildopt-checkstyle/' + relative.as_posix(), sha256=hashlib.sha256(data).hexdigest(), bytes=len(data), copy=str(copied)))
        if arm == 'N':
            assert not entries, ('UNEXPECTED_NATIVE_CANDIDATE_STATE', attempt.name)
        if native['exitCode'] == 0 and arm == 'I':
            expected_tasks = {'checkstyleMain','checkstyleTest','checkstyleInternalClusterTest'}
            assert {Path(entry['path']).parent.name for entry in entries} == expected_tasks, ('CANDIDATE_ACTIVATION_MISSING', attempt.name, entries)
        save(dest / 'receipt.json', dict(attempt=attempt.name, native=bindings.bind(nativepath), originalOrdinal=start['ordinal'] + offset, arm=arm, atUTC=now(), atBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME), entries=entries))
        completed.add(attempt.name)

failure = None
with logpath.open('x') as log, samplepath.open('x') as samples_output:
    child = subprocess.Popen(command, cwd=R, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
    receipt.update(pid=child.pid, procStartTicks=stat(child.pid)['startTicks'])
    save(P / f'receipts/profile-{label}-start.json', receipt)
    s = load(statepath)
    s.update(runningProcesses=[dict(pid=child.pid, procStartTicks=receipt['procStartTicks'], receipt=str(P / f'receipts/profile-{label}-start.json'))], activeProfile=label, gradleReserved=allocation['ownerReserved'])
    atomic(statepath, s)
    deadline = time.monotonic() + m['limits']['maxRunNS'] / 1e9
    next_identity = 0
    try:
        while child.poll() is None:
            if time.monotonic() >= deadline or datetime.datetime.now(datetime.timezone.utc) >= datetime.datetime.fromisoformat(allocation['deadlineUTC']) or shutil.disk_usage(P).free < allocation['minimumFreeBytes']:
                raise RuntimeError('Registered wall/free-space bound reached')
            observe(samples_output)
            diagnostic_bytes = samplepath.stat().st_size
            if diagnostic_bytes > allocation['maxDiagnosticBytes']:
                raise RuntimeError('Diagnostic artifact byte bound reached')
            if time.monotonic() >= next_identity:
                inspect_worktrees()
                next_identity = time.monotonic() + 5
            time.sleep(1.0)
        observe(samples_output)
        inspect_worktrees()
    except Exception as error:
        failure = repr(error)
        if child.poll() is None: os.killpg(child.pid, signal.SIGTERM)
        for unit, group in groups.items():
            assert group['path'].endswith('/' + unit) and unit.startswith('buildopt-replay-')
            subprocess.run(['systemctl', '--user', 'stop', unit], timeout=15, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    receipt.update(exitCode=child.wait(), endUTC=now(), endBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME), observationFailure=failure, processSamples=samples, finishedNativeRequests=len(completed))
receipt.update(logSHA256=sha(logpath), processSamplesSHA256=sha(samplepath))
save(P / f'receipts/profile-{label}-end.json', receipt)
allocation = load(P / 'allocation.json')
if (run / 'result.json').exists():
    result = load(run / 'result.json')
    allocation['actualOwnerStarts'] = result['actualGradleStarts']
else:
    allocation['actualOwnerStarts'] = len(list((run / 'attempts').glob('*/native-start.json')))
allocation['completedOwnerRequests'] = len(completed)
atomic(P / 'allocation.json', allocation)
s = load(statepath)
s['runningProcesses'] = []
atomic(statepath, s)
print(json.dumps(receipt))
if receipt['exitCode'] != 0 or failure: print(logpath.read_text()[-5000:])
sys.exit(0 if receipt['exitCode'] == 0 and failure is None else 1)
