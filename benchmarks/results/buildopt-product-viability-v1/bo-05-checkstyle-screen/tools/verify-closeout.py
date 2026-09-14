"""Verify the completed screen against immutable sources and all raw pairs."""
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
result = load(RUN / 'result.json')
profile = load(P / 'analysis/profile-supported.json')
host = load(P / 'analysis/host-pressure.json')
phases = load(P / 'analysis/diagnostic-phases.json')
receipt = load(P / 'receipts/profile-supported-end.json')
assert 'controlBaseline' not in m
assert result['decision'] == 'FIXTURE_VERIFIED'
assert result['actualGradleStarts'] == result['workflowStarts'] == result['gradleReservations'] == 16
assert result['nestedStarts'] == result['unknownGradleReservations'] == 0
assert profile['metadataJVMActual'] == 16 and profile['measuredPairs'] == 6
assert len(host['requests']) == len(phases['attempts']) == 16 and phases['actualWorkCountsVerified']
assert receipt['exitCode'] == 0 and receipt['observationFailure'] is None
assert len(list((RUN / 'pairs').glob('*.json'))) == 8
assert len(list((RUN / 'attempts').glob('*/native-start.json'))) == 16

source_rows = []
for replication in (1, 2):
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
        assert identity['head'] == m['history'][-1]['commit'] and identity['commonGit'] == m['commonGit']
        identities.append(identity)
closures = []
for cfg in (RUN / 'sessions').glob('*/worker-config.json'):
    closure = load(cfg.parent / 'closed.json')
    assert not closure['remaining'] and not (Path('/sys/fs/cgroup') / closure['cgroup'].lstrip('/')).exists()
    closures.append(closure)
assert len(closures) == 4
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
verification = dict(status='verified', decision=profile['decision'], sourceIdentity=source_rows,
    worktrees=identities, frozenBindingsMatched=len(freeze), actualOwnerStarts=16,
    liveComparatorJVMs=8, independentComparatorJVMs=8, extraComparatorPasses=0,
    retryStarts=0, ownedSessionsClosed=4, historicalStatePreserved=True,
    sourceCorrectnessQualifiedForScreen=True, measurementPrecisionQualified=False,
    productValueQualified=False, allocatedBytes=size, withinDeadline=True,
    verificationUTC=now, verifier=bindings.bind(Path(__file__).resolve()))
with (P / 'analysis/verification.json').open('x') as output:
    output.write(json.dumps(verification,indent=2)+'\n')
allocation.update(status='closed', actualHelpers=16, actualOwnerStarts=16,
    completedOwnerRequests=16, finalAllocatedBytes=size, closedUTC=now)
(P / 'allocation.json').write_text(json.dumps(allocation,indent=2)+'\n')
s['steps'].update(screen='verified', analysis='verified')
s.update(runningProcesses=[], ownerStarts=16, metadataJVMs=16, decision=profile['decision'],
    verification=str(P/'analysis/verification.json'), nextAction='Export all evidence, update tracker and publish the completed BO-05 block.')
(P / 'task-state.json').write_text(json.dumps(s,indent=2)+'\n')
print(json.dumps({k:v for k,v in verification.items() if k not in ('sourceIdentity','worktrees')}))
