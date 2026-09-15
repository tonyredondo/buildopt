"""Run one frozen lean research trial. Does not retry or start another stage."""
from common import *
import datetime,os,signal,shutil,subprocess,time,traceback

def now():return datetime.datetime.now(datetime.timezone.utc).isoformat()
def stat(pid):
 raw=Path(f'/proc/{pid}/stat').read_text();f=raw.rsplit(')',1)[1].split()
 return dict(pid=pid,startTicks=int(f[19]),cpuTicks=int(f[11])+int(f[12]),state=f[0],minorFaults=int(f[7]),majorFaults=int(f[9]),rssPages=int(f[21]))
label=sys.argv[1];assert label in ('control','prefix')
cpus=8;offset=17 if label=='control' else 0
profile=P/'profiles'/label;run=profile/'run';statepath=P/'task-state.json'
m=load(profile/'manifest.json');a=load(P/'allocation.json')
assert not run.exists()
for b in load(P/'inputs'/f'{label}-freeze.json'):assert bind(b['path'])==b,b['path']
assert m['mode']=='P_LEAN_RESEARCH_V1' and m['replications']==1 and m['limits']['retryPairs']==0
assert m['environment']['JAVA_TOOL_OPTIONS']=='-Dfile.encoding=UTF-8'
starts=2*(m['executionEnd']+1);assert starts==(8 if label=='control' else 42)
assert a['ownerReserved']+starts<=a['maxOwnerStarts'] and a['helpersReserved']+starts<=a['maxHelpers']
assert datetime.datetime.now(datetime.timezone.utc)<datetime.datetime.fromisoformat(a['deadlineUTC'])
a['ownerReserved']+=starts;a['helpersReserved']+=starts;atomic(P/'allocation.json',a)
os.sched_setaffinity(0,{9})
command=[m['executable']['path'],'run',str(profile/'manifest.json')]
receipt=dict(profile=label,command=command,cwd=str(R),startUTC=now(),startBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME),gradleReserved=starts,helperJVMReserved=starts)
logpath=P/'logs'/f'{label}-runner.log';samplepath=profile/'process-samples.jsonl'
groups={};completed=set();identities={};samples=0
bindings=binding_module
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

def optional_process_text(path, unavailable):
    try:
        return path.read_text()
    except PermissionError as error:
        unavailable.append(dict(path=str(path), errno=error.errno, reason='permission_denied'))
        return None

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
                unavailable = []
                io_text = optional_process_text(Path(f'/proc/{pid}/io'), unavailable)
                io = None
                if io_text is not None:
                    io = {}
                    for line in io_text.splitlines():
                        key, value = line.split(':', 1); io[key] = int(value)
                wait_text = optional_process_text(Path(f'/proc/{pid}/wchan'), unavailable)
                wait = None if wait_text is None else wait_text.strip()
                processes.append(dict(identity=identity, command=commandline, threadMasks=masks,
                    io=io, mainThreadWaitChannel=wait, unavailableObservations=unavailable))
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
        save(dest / 'receipt.json', dict(attempt=attempt.name, native=bindings.bind(nativepath), originalOrdinal=start['ordinal'] + offset, arm=arm, atUTC=now(), atBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME), entries=entries))
        completed.add(attempt.name)

failure=None
with logpath.open('x') as log,samplepath.open('x') as output:
 child=subprocess.Popen(command,cwd=R,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
 receipt.update(pid=child.pid,procStartTicks=stat(child.pid)['startTicks']);save(P/'receipts'/f'{label}-start.json',receipt)
 state=load(statepath);state['runningProcesses']=[dict(pid=child.pid,procStartTicks=receipt['procStartTicks'],receipt=str(P/'receipts'/f'{label}-start.json'))];atomic(statepath,state)
 next_identity=0
 try:
  while child.poll() is None:
   if datetime.datetime.now(datetime.timezone.utc)>=datetime.datetime.fromisoformat(a['deadlineUTC']) or shutil.disk_usage(P).free<a['minimumFreeBytes']:raise RuntimeError('Approved time or free disk limit reached')
   observe(output)
   if samplepath.stat().st_size>a['maxDiagnosticBytes']:raise RuntimeError('Diagnostic byte limit reached')
   if time.monotonic()>=next_identity:inspect_worktrees();next_identity=time.monotonic()+5
   time.sleep(1)
  observe(output);inspect_worktrees()
 except Exception as error:
  failure=dict(type=type(error).__name__,message=str(error),traceback=traceback.format_exc())
  if child.poll() is None:os.killpg(child.pid,signal.SIGTERM)
  for unit,group in groups.items():
   assert group['path'].endswith('/'+unit) and unit.startswith('buildopt-replay-')
   subprocess.run(['systemctl','--user','stop',unit],timeout=15,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
 receipt.update(exitCode=child.wait(),endUTC=now(),endBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME),observationFailure=failure,processSamples=samples,finishedNativeRequests=len(completed))
receipt.update(log=bind(logpath),processSamples=bind(samplepath))
save(P/'receipts'/f'{label}-end.json',receipt)
a=load(P/'allocation.json');a['actualOwnerStarts']=sum(len(list(x.glob('run/attempts/*/native-start.json'))) for x in (P/'profiles').iterdir())
atomic(P/'allocation.json',a)
state=load(statepath);state['runningProcesses']=[];state['actualOwnerStarts']=a['actualOwnerStarts'];atomic(statepath,state)
print(json.dumps(receipt),flush=True)
if receipt['exitCode']!=0 or failure:raise SystemExit(1)
