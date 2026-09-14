"""Verify retained partial evidence and cleanup; do not qualify the interrupted screen."""
from pathlib import Path
import datetime
import hashlib
import importlib.util
import json
import shutil
import subprocess
import sys

sys.dont_write_bytecode = True
P = Path(__file__).resolve().parent
RUN = P / 'profiles/supported/run'

def load(path):
    return json.loads(path.read_text())

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def bound(binding):
    assert sha(Path(binding['path'])) == binding['sha256'], binding['path']
    return load(Path(binding['path']))

spec = importlib.util.spec_from_file_location('bindings', P.parent / 'bv006-cpu-isolation/bindings.py')
bindings = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bindings)
freeze = load(P / 'inputs/launch-freeze.json')
for binding in freeze:
    assert bindings.bind(binding['path']) == binding, binding['path']
m = load(P / 'profiles/supported/manifest.json')
assert not (RUN / 'result.json').exists()
profile = load(P / 'analysis/incomplete-profile-supported.json')
host = load(P / 'analysis/host-pressure.json')
phases = load(P / 'analysis/diagnostic-phases.json')
receipt = load(P / 'receipts/profile-supported-end.json')
assert 'controlBaseline' not in m
assert profile['decision'] == 'INCOMPLETE_SCREEN'
assert profile['metadataJVMActual'] == 4 and profile['measuredPairs'] == 3
assert len(host['requests']) == len(phases['attempts']) == 8 and phases['actualWorkCountsVerified']
assert receipt['exitCode'] == -15 and receipt['observationFailure'].startswith('PermissionError')
assert len(list((RUN / 'pairs').glob('*.json'))) == 4
assert len(list((RUN / 'attempts').glob('*/native-start.json'))) == 8

source_rows = []
for replication in (1,):
    for ordinal, revision in enumerate(m['history']):
        commit = revision['commit']
        paths = {'repo/' + path.decode() for path in subprocess.check_output([
            'git', '--git-dir', m['commonGit'], 'ls-tree', '-rz', '--name-only', commit], timeout=30).split(b'\0') if path}
        candidate_paths = {'repo/' + item['path'] for item in m['candidate']['files']}
        snapshots = {}
        for arm in ('N', 'I'):
            attempt = RUN / 'attempts' / f'r{replication}-{ordinal:03d}-g0-{arm}'
            start = load(attempt / 'start.json')
            end = load(attempt / 'end.json')
            assert start['replication'] == replication and start['ordinal'] == ordinal
            assert end['candidateApplied'] == (arm == 'I') and end['class'] == 'COMPARABLE'
            before, after = bound(start['before']), bound(end['after'])
            for inventory in (before, after):
                assert inventory['revision'] == commit and inventory['commonGit'] == m['commonGit']
            before_entries = {e['path']:e for e in before['entries'] if e['path'] in paths | candidate_paths}
            after_entries = {e['path']:e for e in after['entries'] if e['path'] in paths | candidate_paths}
            assert set(before_entries) == paths
            for item in m['baseline']['files']:
                for entries in (before_entries, after_entries):
                    e = entries['repo/' + item['path']]
                    assert e['sha256'] == item['after']['sha256'] and e['mode'] == item['afterMode']
            if arm == 'N':
                assert before_entries == after_entries, (replication, ordinal, 'native source changed')
            else:
                assert set(after_entries) == paths | candidate_paths
                for item in m['candidate']['files']:
                    name = 'repo/' + item['path']
                    if item['beforeSHA256']:
                        e = before_entries[name]
                        assert e['sha256'] == item['beforeSHA256'] and e['mode'] == item['beforeMode']
                    else:
                        assert name not in before_entries
                    e = after_entries[name]
                    assert e['sha256'] == item['after']['sha256'] and e['mode'] == item['afterMode']
                assert {k:v for k,v in before_entries.items() if k not in candidate_paths} == {
                    k:v for k,v in after_entries.items() if k not in candidate_paths}
            snapshots[arm] = (before_entries, after_entries)
        assert snapshots['N'][0] == snapshots['I'][0], (replication, ordinal, 'different native preimages')
        source_rows.append(dict(replication=replication, originalOrdinal=ordinal+17, commit=commit,
            nativeSourceFiles=len(paths), candidateChangedFiles=len(candidate_paths),
            nativeArmUnmodified=True, onlyDeclaredCandidateDifference=True,
            nativeEntriesSHA256=hashlib.sha256(json.dumps(snapshots['N'][1],sort_keys=True).encode()).hexdigest(),
            candidateEntriesSHA256=hashlib.sha256(json.dumps(snapshots['I'][1],sort_keys=True).encode()).hexdigest()))

identities = []
for replication in (1, 2):
    for arm in ('N', 'I'):
        repo = RUN / f'r{replication}' / arm / 'repo'
        def git(*args):
            return subprocess.check_output(['git','-C',str(repo),*args],text=True,timeout=30).strip()
        identity = dict(path=str(repo), head=git('rev-parse','HEAD'), branch=git('branch','--show-current'),
            commonGit=git('rev-parse','--path-format=absolute','--git-common-dir'), replication=replication, arm=arm)
        assert identity['head'] == m['history'][-1 if replication == 1 else 0]['commit'] and identity['commonGit'] == m['commonGit']
        identities.append(identity)
closures = []
unit_prefix = 'buildopt-replay-' + hashlib.sha256(str(RUN).encode()).hexdigest()[:16] + '-'
for cfg_path in sorted((RUN / 'sessions').glob('*/worker-config.json')):
    cfg = load(cfg_path); unit = cfg['unit']; assert unit.startswith(unit_prefix)
    completed = subprocess.run(['systemctl','--user','show',unit,
        '--property=ActiveState,SubState,ControlGroup,MainPID','--no-pager'],capture_output=True,text=True,timeout=10)
    assert completed.returncode == 0
    state = dict(line.split('=',1) for line in completed.stdout.splitlines())
    assert state['ActiveState'] == 'inactive' and state['ControlGroup'] == '' and state['MainPID'] == '0'
    closure_path = cfg_path.parent / 'closed.json'
    owner_closure = load(closure_path) if closure_path.exists() else None
    if owner_closure:
        assert not owner_closure['remaining'] and not (Path('/sys/fs/cgroup') / owner_closure['cgroup'].lstrip('/')).exists()
    closures.append(dict(unit=unit, currentState=state, ownerClosure=owner_closure,
        proof='Current external systemd observation; a missing owner closure is not fabricated'))
assert len(closures) == 3
pidpath = Path('/proc', str(receipt['pid']), 'stat')
assert not pidpath.exists() or int(pidpath.read_text().rsplit(')',1)[1].split()[19]) != receipt['procStartTicks']
s = load(P / 'task-state.json')
assert sha(P.parent / 'task-state.json') == s['historicalStateSHA256']
allocation = load(P / 'allocation.json')
size = int(subprocess.check_output(['du','-s','-B1',str(P)],text=True,timeout=120).split()[0])
assert size <= allocation['maxNewBytes']
assert shutil.disk_usage(P).free >= allocation['minimumFreeBytes']
assert datetime.datetime.fromisoformat(receipt['endUTC']) <= datetime.datetime.fromisoformat(allocation['deadlineUTC'])
now = datetime.datetime.now(datetime.timezone.utc).isoformat()
scheduled = []
for replication in (1, 2):
    for ordinal in range(4):
        for arm in ('N','I'):
            attempt = RUN / 'attempts' / f'r{replication}-{ordinal:03d}-g0-{arm}'
            finished = (attempt / 'end.json').exists()
            scheduled.append(dict(replication=replication,originalOrdinal=ordinal+17,arm=arm,
                status='COMPLETED_LIVE_PAIR' if finished else 'NOT_RUN_AFTER_OBSERVER_FAILURE',
                preparationStarted=(attempt / 'start.json').exists(), nativeStarted=(attempt / 'native-start.json').exists(),
                endBinding=bindings.bind(attempt/'end.json') if finished else None,
                retainedFiles=[dict(name=f.name,**bindings.bind(f)) for f in sorted(attempt.glob('*.json'))] if attempt.exists() else []))
assert sum(row['nativeStarted'] for row in scheduled) == 8
unfinished = [dict(name=f.name,**bindings.bind(f),record=load(f)) for f in sorted((RUN/'costs').glob('*.start.json'))
    if not f.with_name(f.name.replace('.start.json','.json')).exists() and not f.name.endswith('.work-start.json')]
verification = dict(status='verified recovery of incomplete evidence',decision='INCOMPLETE_SCREEN',
    sourceIdentity=source_rows, worktrees=identities,frozenBindingsMatched=len(freeze),actualOwnerStarts=8,
    liveComparatorJVMs=4,independentComparatorJVMs=0,extraComparatorPasses=0,retryStarts=0,
    scheduled=scheduled,unclosedCostMarkers=unfinished,closedServiceObservations=closures,
    historicalStatePreserved=True,sourceInventoryChecksPassed=True,fullIndependentOutputProof=False,
    measurementPrecisionQualified=False,productValueQualified=False,allocatedBytes=size,withinDeadline=True,
    verificationUTC=now,verifier=bindings.bind(Path(__file__).resolve()))
with (P / 'analysis/verification.json').open('x') as output:
    output.write(json.dumps(verification,indent=2)+'\n')
allocation.update(status='closed incomplete',actualHelpers=4,actualOwnerStarts=8,
    completedOwnerRequests=8,finalAllocatedBytes=size,closedUTC=now,
    remainingReservationsReusable=False,reason='Observer failure; no automatic replacement trial under the frozen protocol')
(P / 'allocation.json').write_text(json.dumps(allocation,indent=2)+'\n')
s['steps'].update(screen='partial',analysis='verified',observerRepair='verified scoped tests')
s.update(runningProcesses=[],ownerStarts=8,metadataJVMs=4,decision='INCOMPLETE_SCREEN',
    verification=str(P/'analysis/verification.json'),nextAction='Export and publish incomplete BO-05 evidence and the tested observer repair; no timing relaunch.')
(P/'task-state.json').write_text(json.dumps(s,indent=2)+'\n')
print(json.dumps(dict(status=verification['status'],ownerStarts=8,liveComparatorJVMs=4,
    unrunRequests=8,frozenBindingsMatched=len(freeze),allocatedBytes=size,servicesInactive=3)))
