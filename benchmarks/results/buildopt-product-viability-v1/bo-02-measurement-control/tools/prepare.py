"""Allocate one same-code control; preserve the historical runner and schedule."""
from pathlib import Path
import ast
import copy
import datetime
import difflib
import hashlib
import importlib.util
import json
import shutil
import sys

sys.dont_write_bytecode = True
P = Path(__file__).resolve().parent
R = P.parents[3]
F = P.parent / 'bv006-checkstyle-finalization'

def load(path):
    return json.loads(path.read_text())

def save(path, value):
    with path.open('x') as output:
        output.write(json.dumps(value, indent=2) + '\n')

def replace(text, before, after):
    assert text.count(before) == 1, before
    return text.replace(before, after)

def adapt(source, destination, edits):
    original = (F / source).read_text()
    edited = original
    for before, after in edits:
        edited = replace(edited, before, after)
    ast.parse(edited)
    (P / destination).write_text(edited)
    (P / 'inputs' / (destination + '.diff')).write_text(''.join(difflib.unified_diff(
        original.splitlines(True), edited.splitlines(True), fromfile=str(F / source), tofile=str(P / destination))))

spec = importlib.util.spec_from_file_location('bindings', P.parent / 'bv006-cpu-isolation/bindings.py')
bindings = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bindings)
prior = load(F / 'inputs/comparison-supported-final-freeze.json')
for binding in prior:
    assert bindings.bind(binding['path']) == binding, binding['path']

adapt('run-comparison-supported.py', 'run-control.py', [
    ('Run the frozen current-versus-revised 6-to-7 comparison', 'Run the frozen identical-code 6-to-7 control'),
    ("I = P.parent / 'bv006-cpu-isolation'", "I = P.parent / 'bv006-cpu-isolation'\nF = P.parent / 'bv006-checkstyle-finalization'"),
    ("P / 'inputs/comparison-supported-final-freeze.json'", "P / 'inputs/launch-freeze.json'"),
    ("P / 'analysis/preliminary-controller.json'", "F / 'analysis/preliminary-controller.json'"),
    ("P / 'receipts/owner-runner-final-proof.json'", "F / 'receipts/owner-runner-final-proof.json'"),
    ("P / 'analysis/correctness-v2.json'", "F / 'analysis/correctness-v2.json'"),
    ("assert load(P / 'receipts/agent-proof-v2-verified.json')['exitCode'] == 0", "assert load(F / 'receipts/agent-proof-v2-verified.json')['exitCode'] == 0"),
    ("assert load(P / 'receipts/agent-proof-v2-verified.json')['contextEvents']", "assert load(F / 'receipts/agent-proof-v2-verified.json')['contextEvents']"),
    ("assert starts == 8 and allocation['ownerReserved'] == 16 and allocation['actualOwnerStarts'] == 2", "assert starts == 4 and allocation['ownerReserved'] == allocation['actualOwnerStarts'] == 0\nassert m['controlBaseline']['files'] == m['baseline']['files'] + m['candidate']['files']\nassert m['controlBaseline']['prerequisites'] == m['candidate']['prerequisites']\nassert m['replications'] == 1 and m['limits']['retryPairs'] == 0"),
    ("statepath = P.parent / 'task-state.json'", "statepath = P / 'task-state.json'"),
    ("state['engineeringPrefix']['checkstyleFinalization']['worktrees']", "state['worktrees']"),
    ("s['engineeringPrefix']['checkstyleFinalization'].update(", "s.update("),
    ("allocation['actualOwnerStarts'] = 2 + result['actualGradleStarts']", "allocation['actualOwnerStarts'] = result['actualGradleStarts']"),
    ("allocation['actualOwnerStarts'] = 2 + len(list((run / 'attempts').glob('*/native-start.json')))", "allocation['actualOwnerStarts'] = len(list((run / 'attempts').glob('*/native-start.json')))"),
    ("allocation['completedOwnerRequests'] = 2 + len(completed)", "allocation['completedOwnerRequests'] = len(completed)"),
    ("s['engineeringPrefix']['checkstyleFinalization']['runningProcesses']", "s['runningProcesses']"),
])

# Existing analyzers read raw files only. The frozen runner's `run` already
# performs both live and independent C5 checks; never invoke a third pass.
adapt('analyze-comparison-v2.py', 'analyze-control.py', [
    ("assert len(workers) == 4", "assert len(workers) == 2"),
    ("warmupBuilds=4", "warmupBuilds=2"),
    ("    report['schema'] = 'buildopt.checkstyle-finalization/profile/v1'", "    report['schema'] = 'buildopt.identical-code-control/profile/v1'"),
    ("    report['armLabels'] = {'N':'current optimized implementation', 'I':'revised optimized implementation'}", "    report['armLabels'] = {'N':'supported V2', 'I':'identical supported V2'}"),
    ("    report['materialSignal'] = all(pair['customerSavingMS'] >= 1000 for pair in measured) and metrics['customerMS']['savingPercent'] >= 5\n    report['decision'] = 'EXPLORATORY_FINALIZATION_SAVING' if report['materialSignal'] else 'FINALIZATION_VALUE_NOT_ESTABLISHED'", "    assert len(measured) == 1\n    pair = measured[0]\n    delta = pair['customerSavingMS']\n    faster = min(pair['N']['customerMS'], pair['I']['customerMS'])\n    report['absoluteDifferenceMS'] = abs(delta)\n    report['signedDifferenceMS'] = delta\n    report['differencePercentOfFaster'] = 100 * abs(delta) / faster\n    report['materialSignal'] = abs(delta) >= 1000 and 100 * abs(delta) / faster >= 5\n    report['decision'] = 'SPURIOUS_MATERIAL_SIGNAL_IN_AA' if report['materialSignal'] else 'NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL'"),
    ("'Two selected measured pairs; no precision or general product claim'", "'One selected measured pair; no precision or general product claim'"),
    ("'Both arms are optimized; no new native baseline comparison'", "'Both arms use identical supported V2; this measures a false signal, not optimization value'"),
])
adapt('analyze-host-v2.py', 'analyze-host.py', [
    ('assert len(requests) == 8', 'assert len(requests) == 4'),
    ('External workload is inferred, not attributed to Rust, .NET or a named process', 'External workload is inferred; no specific process is identified'),
])
adapt('analyze-phases-v2.py', 'analyze-phases.py', [
    ('assert len(rows) == 8', 'assert len(rows) == 4'),
    ('for replication in (1,2):', 'for replication in (1,):'),
])
adapt('analyze-order-v2.py', 'analyze-order.py', [
    ('for rep in (1,2):', 'for rep in (1,):'),
    ("status='verified descriptive order and idle-age reconstruction; causal attribution unverified'", "status='verified descriptive order and idle-age reconstruction for one A/A pair; causal attribution unverified'"),
    ("bothSecondRequestsSlower=all(rows[i+1]['customerSeconds']>rows[i]['customerSeconds'] for i in (0,2))", "secondRequestSlower=rows[1]['customerSeconds']>rows[0]['customerSeconds']"),
    ("observation='The first measured arm is I in replication1 and N in replication2. Both first requests are near83s; both second requests are slower. Their private daemons also have longer idle ages. This is an association, not proof that capture, cache eviction or another process caused the delay.'", "observation='Same-code 6-to-7 control; order and idle intervals are descriptive, not a causal explanation.'"),
    ("implication='The original both-pairs saving criterion fails. No first-only comparison, outlier removal, timing replacement or new value gate is permitted. A small identical-code A/A control should precede additional native-value claims.'", "implication='Apply the predeclared A/A threshold to the whole request. No sample removal, timing retry or value gate is permitted.'"),
    ("nextControl='Four owner builds: identical correctness-qualified code on both arms, same6->7 schedule and observation. Predeclare whether it produces a spurious material difference; do not expand to native17..20 until this measurement concern is resolved.'", "nextControl='No control rerun authorized by this result. Record the measurement disposition before another timing allocation.'"),
    ("'Only two replications; no variance or causal estimate'", "'Only one replication; no variance or causal estimate'"),
    ("'Whole-build comparisons across replications are descriptive and cannot replace the registered paired decision'", "'This control cannot establish measurement precision or optimization value'"),
])

now = datetime.datetime.now(datetime.timezone.utc)
deadline = now + datetime.timedelta(minutes=90)
m = load(F / 'profiles/supported/manifest.json')
m['subject'] = 'elasticsearch-checkstyle-identical-supported-v2-control'
m['frozenUTC'] = now.isoformat()
m['runRoot'] = str(P / 'profiles/supported/run')
m['replications'] = 1
m['controlBaseline'] = dict(identity='identical-supported-v2-control',
    files=copy.deepcopy(m['baseline']['files'] + m['candidate']['files']),
    prerequisites=copy.deepcopy(m['candidate']['prerequisites']))
m['limits'].update(deadlineUTC=deadline.isoformat(), maxRunNS=5400*10**9,
    maxBytes=32*2**30-256*2**20, maxWorkflowStarts=4, maxGradleStarts=4)
save(P / 'profiles/supported/manifest.json', m)
assert m['controlBaseline']['files'] == m['baseline']['files'] + m['candidate']['files']
assert shutil.disk_usage(P).free >= 40*2**30
allocation = dict(status='allocated', startedUTC=now.isoformat(), deadlineUTC=deadline.isoformat(),
    maxElapsedSeconds=5400, maxNewBytes=32*2**30, minimumFreeBytes=40*2**30,
    maxDiagnosticBytes=128*2**20, reservedNonRunBytes=256*2**20,
    maxOwnerControllerReservations=4, maxActualOwnerStarts=4, maxStandaloneHelpers=4,
    ownerReserved=0, actualOwnerStarts=0, actualHelpers=0, completedOwnerRequests=0,
    actualCompilerCommands=0, actualFixtureStarts=0, retriesAllowed=0,
    accounting='Four native starts; two live and two independently reconstructed metadata JVMs included by the frozen run command. No standalone check command.')
save(P / 'allocation.json', allocation)
save(P / 'inputs/allocation-frozen.json', allocation)
save(P / 'inputs/protocol.json', dict(
    question='Does the preserved schedule produce a material difference with identical supported V2 code?',
    absoluteThresholdMS=1000, percentThresholdOfFaster=5,
    falseSignal='SPURIOUS_MATERIAL_SIGNAL_IN_AA',
    noFalseSignal='NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL',
    correctnessFailure='INCOMPLETE_CONTROL',
    primaryMetric='Attempt duration plus all customer-machine costs outside that request envelope',
    builds=4, measuredPairs=1, timingRetry=False, sourceIdentityRequired=True,
    referenceProtocol=bindings.bind(R/'docs/plans/buildopt-checkstyle-measurement-control-v1.md')))
freeze = prior + [bindings.bind(P / path) for path in (
    'run-control.py', 'profiles/supported/manifest.json', 'inputs/allocation-frozen.json',
    'inputs/protocol.json', 'analyze-control.py', 'analyze-host.py', 'analyze-phases.py', 'analyze-order.py')]
save(P / 'inputs/launch-freeze.json', freeze)
save(P / 'receipts/preparation.json', dict(status='verified', priorBindingsMatched=len(prior),
    nativeStarts=0, metadataJVMStarts=0, sameEffectiveFiles=True, effectiveFileCount=6,
    samePrerequisites=True, syntaxChecked=5, deadlineUTC=deadline.isoformat(),
    freeBytes=shutil.disk_usage(P).free,
    helperAccounting='Source inspection: run invokes writeResult, which reconstructs all pairs; no extra check invocation needed.'))
state = load(P / 'task-state.json')
state.update(nextAction='Validate fresh manifest once, then run frozen four-build control',
    allocation=str(P/'allocation.json'), launchFreeze=str(P/'inputs/launch-freeze.json'))
(P/'task-state.json').write_text(json.dumps(state,indent=2)+'\n')
print(json.dumps(dict(prepared=True, frozenBindings=len(freeze), deadlineUTC=deadline.isoformat())))
