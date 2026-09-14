"""Check source identity and allocation after the frozen control has completed."""
from pathlib import Path
import datetime
import hashlib
import importlib.util
import json
import os
import shutil
import subprocess
import sys

sys.dont_write_bytecode = True
P = Path(__file__).resolve().parent
R = P.parents[3]
RUN = P / 'profiles/supported/run'

def load(path):
    return json.loads(Path(path).read_text())

def bound(binding):
    path = Path(binding['path'])
    assert hashlib.sha256(path.read_bytes()).hexdigest() == binding['sha256'], path
    return load(path)

def save(path, value):
    with path.open('x') as output:
        output.write(json.dumps(value, indent=2) + '\n')

spec = importlib.util.spec_from_file_location('bindings', P.parent/'bv006-cpu-isolation/bindings.py')
bindings = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bindings)
freeze = load(P/'inputs/launch-freeze.json')
for binding in freeze:
    assert bindings.bind(binding['path']) == binding, binding['path']

m = load(P/'profiles/supported/manifest.json')
actual = load(RUN/'manifest.json')
assert m == actual
profile = load(P/'analysis/profile-supported.json')
phases = load(P/'analysis/diagnostic-phases.json')
host = load(P/'analysis/host-pressure.json')
order = load(P/'analysis/order-and-idle-age.json')
result = load(RUN/'result.json')
receipt = load(P/'receipts/profile-supported-end.json')
assert receipt['exitCode'] == 0 and receipt['observationFailure'] is None
assert result['decision'] == 'FIXTURE_VERIFIED'
assert result['actualGradleStarts'] == result['workflowStarts'] == result['gradleReservations'] == 4
assert profile['metadataJVMActual'] == 4 and profile['measuredPairs'] == 1
assert phases['actualProcessedAndReusedCountsEquivalent']
assert len(host['requests']) == 4 and len(order['rows']) == 2
assert m['controlBaseline']['files'] == m['baseline']['files'] + m['candidate']['files']

source_rows = []
for ordinal in range(2):
    revision = m['history'][ordinal]['commit']
    output = subprocess.check_output(['git','--git-dir',m['commonGit'],'ls-tree','-rz','--name-only',revision])
    paths = {'repo/' + path.decode() for path in output.split(b'\0') if path}
    paths.update('repo/' + item['path'] for item in m['controlBaseline']['files'])
    sources = {}
    for arm in ('N','I'):
        attempt = RUN/'attempts'/f'r1-{ordinal:03d}-g0-{arm}'
        start = load(attempt/'start.json')
        end = load(attempt/'end.json')
        inventories = [bound(start['before']), bound(end['after'])]
        retained = []
        for index, inventory in enumerate(inventories):
            assert inventory['revision'] == revision and inventory['commonGit'] == m['commonGit']
            entries = {entry['path']:entry for entry in inventory['entries'] if entry['path'] in paths}
            recipes = m['controlBaseline']['files'] if arm == 'N' or index == 1 else m['baseline']['files']
            added = set()
            if arm == 'I' and index == 0:
                for item in m['candidate']['files']:
                    name = 'repo/' + item['path']
                    if item['beforeSHA256']:
                        assert entries[name]['sha256'] == item['beforeSHA256']
                        assert entries[name]['mode'] == item['beforeMode']
                    else:
                        assert name not in entries
                        added.add(name)
            assert entries.keys() == paths - added
            for item in recipes:
                entry = entries['repo/'+item['path']]
                assert entry['sha256'] == item['after']['sha256'] and entry['mode'] == item['afterMode']
            retained.append(entries)
        # The frozen I request applies its patch inside the measured envelope.
        # Its preflight records the preimage, whereas N already has the same V2
        # files through controlBaseline. Preserve and verify that distinction.
        effective_before = dict(retained[0])
        if arm == 'I':
            for item in m['candidate']['files']:
                name = 'repo/' + item['path']
                effective_before[name] = retained[1][name]
        assert effective_before == retained[1], (ordinal, arm, 'undeclared source change')
        sources[arm] = retained[1]
    assert sources['N'] == sources['I'], (ordinal, 'same-code source mismatch')
    source_rows.append(dict(originalOrdinal=ordinal+6,revision=revision,sourceFiles=len(paths),
        onlyDeclaredPatchApplied=True, identicalEffectiveSourcesBetweenArms=True,
        entriesSHA256=hashlib.sha256(json.dumps(sources['N'],sort_keys=True).encode()).hexdigest()))

identities = []
for arm in ('N','I'):
    repo = RUN/'r1'/arm/'repo'
    def git(*args):
        return subprocess.check_output(['git','-C',str(repo),*args],text=True).strip()
    identity = dict(path=str(repo), head=git('rev-parse','HEAD'),branch=git('branch','--show-current'),
        commonGit=git('rev-parse','--path-format=absolute','--git-common-dir'))
    assert identity['head'] == m['history'][1]['commit'] and identity['commonGit'] == m['commonGit']
    identities.append(identity)

closures = []
for cfg in (RUN/'sessions').glob('*/worker-config.json'):
    closure = load(cfg.parent/'closed.json')
    assert not closure['remaining'] and not (Path('/sys/fs/cgroup')/closure['cgroup'].lstrip('/')).exists()
    closures.append(closure)
assert len(closures) == 2
assert not Path('/proc',str(receipt['pid'])).exists() or int(Path('/proc',str(receipt['pid']),'stat').read_text().rsplit(')',1)[1].split()[19]) != receipt['procStartTicks']
state = load(P/'task-state.json')
assert hashlib.sha256((P.parent/'task-state.json').read_bytes()).hexdigest() == state['historicalStateSHA256']

allocation = load(P/'allocation.json')
size = int(subprocess.check_output(['du','-s','-B1',str(P)],text=True).split()[0])
assert size <= allocation['maxNewBytes']
assert shutil.disk_usage(P).free >= allocation['minimumFreeBytes']
assert datetime.datetime.fromisoformat(receipt['endUTC']) <= datetime.datetime.fromisoformat(allocation['deadlineUTC'])
verification = dict(status='verified',decision=profile['decision'],sourceIdentity=source_rows,
    worktrees=identities,frozenBindingsMatched=len(freeze),actualOwnerStarts=4,
    liveComparatorJVMs=2,independentComparatorJVMs=2,extraComparatorPasses=0,
    retryStarts=0,ownedSessionsClosed=len(closures),historicalStatePreserved=True,
    sourceCorrectnessQualified=True,measurementPrecisionQualified=False,productValueQualified=False,
    allocatedBytes=size,freeBytes=shutil.disk_usage(P).free,withinDeadline=True,
    verificationUTC=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    analyzer=bindings.bind(Path(__file__).resolve()))
save(P/'analysis/verification.json',verification)
allocation.update(status='closed',actualHelpers=4,actualOwnerStarts=4,completedOwnerRequests=4,
    finalAllocatedBytes=size,closedUTC=verification['verificationUTC'])
(P/'allocation.json').write_text(json.dumps(allocation,indent=2)+'\n')
state.update(status='verified',active=False,runningProcesses=[],decision=profile['decision'],
    verification=str(P/'analysis/verification.json'),
    nextAction='Record timing disposition in governing tracker; no further owner starts in this allocation')
(P/'task-state.json').write_text(json.dumps(state,indent=2)+'\n')
print(json.dumps(verification))
