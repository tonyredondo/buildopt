"""Recover immutable BO-06 inputs and test confirmation refusal; no builds."""
from pathlib import Path
import datetime
import hashlib
import json
import subprocess

P = Path(__file__).resolve().parent
R = P.parents[3]
S = P.parent / 'bo-05-checkstyle-observer-replay-2026-09-14'
F = P.parent / 'bv006-checkstyle-finalization'

def load(path):
    return json.loads(path.read_text())

def bound(path):
    return dict(path=str(path), sha256=hashlib.sha256(path.read_bytes()).hexdigest())

def save(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('x') as stream:
        stream.write(json.dumps(value, indent=2) + '\n')

def command(args, timeout=120):
    result = subprocess.run(args, cwd=R, capture_output=True, text=True, timeout=timeout)
    if result.returncode:
        raise RuntimeError((args, result.returncode, result.stderr[:2000]))
    return result.stdout

manifest = load(S / 'profiles/supported/manifest.json')
runner = manifest['executable']['path']
assert bound(Path(runner)) == manifest['executable']
seed = P / 'inputs/seed-history.json'
command([runner, 'history', manifest['commonGit'], '16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7', '101', str(seed)])
expected = P.parent / 'bv006/inputs/seed-history-v2-101.json'
history = load(seed)
assert history == load(expected)
assert len(history) == 101
assert all(history[17 + index]['commit'] == row['commit'] for index, row in enumerate(manifest['history']))

original_subjects = R / 'benchmarks/results/buildopt-product-viability-v1/bv006/subjects.json'
subjects = load(original_subjects)
assert len(subjects['repositories']) == 6 and subjects['selected'] == []
replications = []
for owner in subjects['repositories']:
    name = Path(owner['commonGit']).stem
    target = P / 'inputs/replication-histories' / (name + '.json')
    target.parent.mkdir(exist_ok=True)
    command([runner, 'history', owner['commonGit'], owner['endpoint'], '101', str(target)])
    rows = load(target)
    assert len(rows) == 101 and rows[0]['commit'] == owner['anchor'] and rows[-1]['commit'] == owner['endpoint']
    assert all(row['ordinal'] == index and (not index or row['parent'] == rows[index - 1]['commit']) for index, row in enumerate(rows))
    replications.append(dict(**owner, history=bound(target), actualNativeStarts=0, candidateTimingsRead=False))
save(P / 'inputs/subjects.json', dict(original=bound(original_subjects), selectionRule=subjects['selectionRule'], selected=[], repositories=replications))
save(P / 'analysis/history-verification.json', dict(status='verified', seed=bound(seed), priorSeed=bound(expected),
    seedRevisions=101, replicationHistories=len(replications), replicationRevisions=606,
    preservedSeed=True, preservedOrderAndEndpoints=True, reads='Git commit/tree metadata and changed-path digests only',
    candidateValidationSourcesRead=False, candidateValidationTimingsRead=False, nativeStarts=0))

# Preserve the current implementation and scientific choices. This is deliberately
# not a launchable manifest: no successful overhead receipt or allocation exists.
draft = dict(manifest)
draft.update(phase='CONFIRMATION', history=history, prefixEnd=20, executionEnd=100,
    subject='elasticsearch-checkstyle-fixed-confirmation',
    runRoot=str(P / 'confirmation-not-admitted'), subjects=bound(P / 'inputs/subjects.json'))
# Keep the historical frozen bounds unchanged for this structural refusal probe.
# Its expired deadline provides an additional barrier; validation does not launch.
save(P / 'inputs/confirmation-not-admitted.json', draft)
probe = subprocess.run([runner, 'validate', str(P / 'inputs/confirmation-not-admitted.json')], cwd=R, capture_output=True, text=True, timeout=120)
assert probe.returncode == 1
assert probe.stderr.strip() == 'confirmation missing qualification/subjects binding', probe.stderr
assert not Path(draft['runRoot']).exists()
save(P / 'receipts/confirmation-refusal.json', dict(command=probe.args, exitCode=probe.returncode,
    error=probe.stderr.strip(), missingField='overhead', nativeStarts=0, runRootCreated=False,
    limits='Structural refusal only. The copied expired 16-start limits are not a proposed or authorized confirmation allocation.'))

old = R / 'benchmarks/results/buildopt-product-viability-v1/bv006/readiness-decision.json'
precision = R / 'benchmarks/results/buildopt-product-viability-v1/bv006-attribution/analysis/precision-pilot-result.json'
d = load(old)
p = load(precision)
assert d['actualOwnerQualificationPassed'] is False
assert d['armImbalanceNS'] > d['maximumArmImbalanceNS']
assert p['statistics']['decision'] == 'INSUFFICIENT_PRECISION_NO_CONFIRMATION'
assert p['statistics']['requiredPairsPerArm'] == 25656
save(P / 'analysis/measurement-readiness.json', dict(status='blocked', decision='CONFIRMATION_NOT_ADMITTED',
    originalOwnerResult=bound(old), originalImbalanceNS=d['armImbalanceNS'], maximumImbalanceNS=d['maximumArmImbalanceNS'],
    precisionResult=bound(precision), requiredPairsPerArm=25656, oldMaximumPairsPerArm=512,
    shortScreenOverheadBinding=manifest['overhead'],
    diagnosticAgentPresent='-javaagent:' in manifest['environment'].get('JAVA_TOOL_OPTIONS', ''),
    formalG0Qualified=False, formalG3Status='untested; BO-07 outcome, not a BO-06 prerequisite',
    blockers=[
        'The screened executable has no current actual-owner overhead qualification.',
        'The closed 100-ms precision design exceeded its fixed feasible size; no smaller retry is admitted.',
        'The short screen includes diagnostic Java instrumentation; its findings cannot simply become primary confirmation measurements.',
        'The 500-ms observed-runner contract remains unqualified; external samples in the chosen v2 screen do not satisfy it.',
        'No complete supported-V2 development-prefix replay or feasible full-confirmation resource allocation has been frozen.'
    ], protectedValidationStarts=0, ownerStarts=0,
    next='Choose and preregister a feasible primary measurement design within BO-06; preserve the current failed qualifications and all savings/output criteria. Do not rerun the old precision pilot.'))

pins = []
for name in ['candidate', 'baseline', 'command', 'environment', 'runtime', 'acquisition', 'outputs', 'affinity', 'daemonPolicy']:
    pins.append((name, manifest[name]))
save(P / 'inputs/candidate-and-workflow.json', dict(sourceManifest=bound(S / 'profiles/supported/manifest.json'),
    status='screened inputs preserved; diagnostic instrumentation is not qualified for confirmation', **dict(pins)))
save(P / 'inputs/scientific-contract.json', dict(
    source=bound(Path(manifest['protocol']['path'])),
    governingPlan=dict(repository='https://github.com/tonyredondo/buildopt.git',
        revision=load(P / 'task-state.json')['target']['head'],
        path='docs/plans/buildopt-research-execution-plan-2026-09-14.md',
        sha256=hashlib.sha256(command(['git', 'show', load(P / 'task-state.json')['target']['head'] + ':docs/plans/buildopt-research-execution-plan-2026-09-14.md']).encode()).hexdigest()),
    replayContract=bound(R / 'docs/plans/buildopt-product-viability-v1-replay.md'),
    fixedImplementation=manifest['candidate']['identity'], replications=2, anchor=0, prefixEnd=20, validationEnd=100,
    scheduledStarts=404, comparatorJVMStarts=404, validationSlotsPerReplication=80,
    minimumNetFraction=0.05, minimumNetSecondsPerScheduledSlot=1, minimumComparableSlots=76,
    bootstrap=dict(blocks=[5,10], samples=10000, seed=20260908, positiveLower95Required=True),
    p95Allowance=dict(seconds=1, nativeFraction=0.05, use='larger'),
    sameMechanismOwnersRequired=2, totalSelectedOwners=3,
    costRule='Full request plus required machine work outside it once. No prefix saving credit; charge positive aggregate prefix excess. Preserve unavailable-slot candidate costs.',
    allocationActive=False, confirmationAdmitted=False))
print('Seed history, six replication histories and confirmation refusal verified; no builds started.')
