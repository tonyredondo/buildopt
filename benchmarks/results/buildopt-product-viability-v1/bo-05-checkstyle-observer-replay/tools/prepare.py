"""Prepare the registered BO-05 screen without launching Gradle or a JVM."""
from pathlib import Path
import ast
import copy
import datetime
import difflib
import hashlib
import importlib.util
import json
import shutil
import subprocess
import sys

sys.dont_write_bytecode = True
P = Path(__file__).resolve().parent
R = P.parents[3]
F = P.parent / 'bv006-checkstyle-finalization'
A = P.parent / 'bo-02-measurement-control-2026-09-14'

def load(path):
    return json.loads(path.read_text())

def save(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('x') as output:
        output.write(json.dumps(value, indent=2) + '\n')

def replace(text, before, after):
    assert text.count(before) == 1, before
    return text.replace(before, after)

def adapt(name, destination, edits):
    original = (A / name).read_text()
    edited = original
    for before, after in edits:
        edited = replace(edited, before, after)
    ast.parse(edited)
    with (P / destination).open('x') as output:
        output.write(edited)
    (P / 'inputs' / (destination + '.diff')).write_text(''.join(difflib.unified_diff(
        original.splitlines(True), edited.splitlines(True), fromfile=str(A / name), tofile=str(P / destination))))

spec = importlib.util.spec_from_file_location('bindings', P.parent / 'bv006-cpu-isolation/bindings.py')
bindings = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bindings)
prior = load(F / 'inputs/comparison-supported-final-freeze.json')
for binding in prior + load(A / 'inputs/launch-freeze.json'):
    assert bindings.bind(binding['path']) == binding, binding['path']
assert load(A / 'analysis/verification.json')['decision'] == 'NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL'
proof = load(F / 'analysis/correctness-v2.json')
assert proof['status'].startswith('verified') and proof['coreSourceUnchanged']
fixture_bindings = []
for case in proof['cases']:
    folder = F / 'fixture-runs' / case['case']
    for name, key in [('build.log', 'logSHA256'), ('state.json', 'stateSHA256')]:
        b = bindings.bind(folder / name)
        assert b['sha256'] == case[key], (case['case'], name)
        fixture_bindings.append(b)
    assert load(folder / 'end.json')['exitCode'] == case['exitCode']
    fixture_bindings.append(bindings.bind(folder / 'end.json'))
assert load(F / 'receipts/core-tests.json')['exitCode'] == 0
assert load(F / 'receipts/owner-runner-final-proof.json')['status'].startswith('verified')

m = load(F / 'profiles/supported/manifest.json')
preflight = load(F / 'analysis/next-window-source-preflight.json')
assert [row['ordinal'] for row in preflight['revisions']] == [17, 18, 19, 20]
history = []
checks = []
def git(*args):
    return subprocess.check_output(['git', '--git-dir', m['commonGit'], *args], timeout=30)
for index, row in enumerate(preflight['revisions']):
    commit = row['commit']
    metadata = git('show', '-s', '--format=%H%n%T%n%P%n%cI', commit).decode().strip().splitlines()
    parent = metadata[2].split()[0]
    assert metadata[0] == commit and metadata[1] == row['tree']
    if index:
        assert parent == history[-1]['commit']
    changed = git('diff-tree', '--no-commit-id', '--no-renames', '--name-status', '-r', '-z', parent, commit)
    history.append(dict(ordinal=index, commit=commit, parent=parent, tree=metadata[1],
        committedUTC=datetime.datetime.fromisoformat(metadata[3]).astimezone(datetime.timezone.utc).isoformat().replace('+00:00', 'Z'),
        changedPathsSHA256=hashlib.sha256(changed).hexdigest()))
    checked = []
    for recipe in (m['baseline'], m['candidate']):
        for path, expected in recipe['prerequisites'].items():
            assert hashlib.sha256(git('show', commit + ':' + path)).hexdigest() == expected, (commit, path)
            checked.append(path)
        for item in recipe['files']:
            entry = git('ls-tree', commit, '--', item['path']).decode().strip()
            if item['beforeSHA256']:
                assert entry
                assert int(entry.split()[0], 8) & 0o777 == item['beforeMode']
                assert hashlib.sha256(git('show', commit + ':' + item['path'])).hexdigest() == item['beforeSHA256']
            else:
                assert not entry, (commit, item['path'])
            assert bindings.bind(item['after']['path']) == item['after']
            checked.append(item['path'])
    checks.append(dict(originalOrdinal=index + 17, commit=commit, checkedPaths=checked, allMatch=True))

m.pop('controlBaseline')
m.update(subject='elasticsearch-checkstyle-native-screen-17-20', history=history,
    runRoot=str(P / 'profiles/supported/run'), replications=2, executionEnd=3)
now = datetime.datetime.now(datetime.timezone.utc)
deadline = now + datetime.timedelta(hours=3)
m['frozenUTC'] = now.isoformat()
m['limits'].update(deadlineUTC=deadline.isoformat(), maxRunNS=10800*10**9,
    maxBytes=64*2**30-256*2**20, maxWorkflowStarts=16, maxGradleStarts=16)
assert shutil.disk_usage(P).free >= 40*2**30
save(P / 'profiles/supported/manifest.json', m)
allocation = dict(status='allocated', startedUTC=now.isoformat(), deadlineUTC=deadline.isoformat(),
    maxElapsedSeconds=10800, maxNewBytes=64*2**30, minimumFreeBytes=40*2**30,
    maxDiagnosticBytes=128*2**20, reservedNonRunBytes=256*2**20,
    maxOwnerControllerReservations=16, maxActualOwnerStarts=16, maxStandaloneHelpers=16,
    ownerReserved=0, actualOwnerStarts=0, actualHelpers=0, completedOwnerRequests=0,
    actualCompilerCommands=0, actualFixtureStarts=0, retriesAllowed=0,
    accounting='Four cold plus twelve measured owner starts; eight live and eight independent comparator JVMs. No extra builds or comparator pass.')
save(P / 'allocation.json', allocation)
save(P / 'inputs/allocation-frozen.json', allocation)
save(P / 'receipts/readiness.json', dict(status='verified for exploratory screen',
    checkedUTC=now.isoformat(), priorFrozenBindings=len(prior), priorAAControl='NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL',
    sourcePreflight=checks, supportedLifecycleCases=6, coreTestsReused=4,
    unchangedRunner=True, unchangedCandidate=True, unchangedOutputPolicy=True,
    nativeStarts=0, helperJVMStarts=0, formalG0Qualified=False, formalG3Qualified=False,
    scope='Existing scoped correctness and A/A disposition permit exploratory comparison; full timing and protected validation remain unqualified.',
    lifecycleProof=bindings.bind(F/'analysis/correctness-v2.json'),
    liveAndIndependentOutputProof=bindings.bind(A/'analysis/verification.json'),
    fixtureBindings=fixture_bindings))
save(P / 'inputs/protocol.json', dict(
    question='Does supported Checkstyle V2 merit further validation against native Checkstyle on the same full command?',
    primaryMetric='Complete request duration plus required machine costs outside that envelope',
    comparison='Same :server:precommit --continue; retained ForbiddenPatterns baseline in both arms; Checkstyle correction in I only',
    history=[17,18,19,20], coldOrdinals=[17], decisionOrdinals=[18,19,20], replications=2,
    rule='Each replication needs at least 1s mean and 5% aggregate saving over all three post-cold pairs, at least two positive pairs, native checking and candidate reuse. Both must pass.',
    outcomes=['EXPLORATORY_MATERIAL_SIGNAL','NO_MATERIAL_SIGNAL_IN_THIS_SCREEN','INCOMPLETE_SCREEN'],
    coldAccounting='Keep all cold timings and paired excess. Report all request and outside-envelope costs; no payback or sustained net-value claim.',
    pressurePolicy='Retain every request and original host flags; no causal attribution or timing-driven retry.',
    noG0OrG3Promotion=True, noProtectedHistory=True, protocolDocument=bindings.bind(P/'inputs/protocol.md')))
freeze = prior + fixture_bindings + [bindings.bind(P/path) for path in [
    'prepare.py','run-screen-v2.py','run-screen.py','analyze-screen.py','analyze-host.py','decision.py',
    'analyze-phases-v2.py','analyze-order.py','verify-closeout.py','status-v2.py','test-screen-tools-v2.py','test-observer-repair.py','export-evidence.py',
    'receipts/observer-repair-tests.json','receipts/reproducible-tool-tests/result.json','receipts/prior-observer-proof.json',
    'profiles/supported/manifest.json','inputs/allocation-frozen.json','inputs/protocol.json',
    'inputs/protocol.md','receipts/readiness.json']]
save(P / 'inputs/launch-freeze.json', freeze)
s = load(P / 'task-state.json')
s['steps'].update(recovery='verified', earlyBO06='verified for this screen', freeze='in progress')
s.update(allocation=str(P/'allocation.json'), launchFreeze=str(P/'inputs/launch-freeze.json'),
    historicalStateSHA256=bindings.bind(P.parent/'task-state.json')['sha256'],
    nextAction='Validate the fresh manifest and adaptation checks before launching the registered screen.')
(P/'task-state.json').write_text(json.dumps(s,indent=2)+'\n')
print(json.dumps(dict(prepared=True, frozenBindings=len(freeze), deadlineUTC=deadline.isoformat(), sourceRows=len(checks))))
