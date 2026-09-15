"""Recheck retained real outputs with current comparator bytes; no builds."""
from common import *
import importlib.util, subprocess, time

policy_path = P / 'inputs/measured-owner-policy.json'
policy = load(policy_path)
assert bind(policy['comparator']['path']) == policy['comparator']
spec = importlib.util.spec_from_file_location('measured_comparator', policy['comparator']['path'])
owner = importlib.util.module_from_spec(spec); spec.loader.exec_module(owner)
real_run = owner.subprocess.run
events = []
def observed_run(command, **kwargs):
    assert command[0] in (policy['metadataJava']['path'], policy['lexicalProjector']['path']), command[0]
    is_java = command[0] == policy['metadataJava']['path']
    if is_java: reserve('jvmReservations', 'maxComparatorJVMs')
    start = time.monotonic_ns(); result = real_run(command, **kwargs)
    events.append(dict(executable=command[0], java=is_java, startNS=start, endNS=time.monotonic_ns(), exitCode=result.returncode))
    if is_java:
        a = allocation(); a['actualComparatorJVMs'] += 1; atomic(P / 'allocation.json', a)
    return result
owner.subprocess.run = observed_run

requests = []
for name in ('retained-c5', 'native-reuse-000', 'native-reuse-001'):
    request = load(P / 'inputs' / f'prior-{name}-request.json')
    prior = load(P.parent / 'bv006/reader-requalification-v5-02' / f'{name}-result.json')
    assert bind(P / 'inputs' / f'prior-{name}-request.json')['sha256'] == prior['request']['sha256']
    requests.append((name, request, prior['report']['counts']))
for ordinal in (0, 3):
    run = L / 'profiles/control/run'
    requests.append((f'lean-control-{ordinal}', dict(runRoot=str(run), native=load(run / 'attempts' / f'r1-{ordinal:03d}-g0-N/end.json'), candidate=load(run / 'attempts' / f'r1-{ordinal:03d}-g0-I/end.json'), controlAdapted=True), None))
screen = P.parent / 'bo-05-checkstyle-observer-replay-2026-09-14'
# Recover the exact completed run from its manifest rather than guessing its layout.
manifests = list(screen.glob('profiles/*/manifest.json'))
if not manifests:
    manifests = list(screen.glob('*/manifest.json'))
screen_runs = []
for path in manifests:
    m = load(path)
    if m.get('driver') == 'GRADLE' and Path(m['runRoot'], 'result.json').exists(): screen_runs.append(Path(m['runRoot']))
assert len(set(screen_runs)) == 1, screen_runs
run = screen_runs[0]
requests.append(('screen-native-corrected', dict(runRoot=str(run), native=load(run / 'attempts/r1-003-g0-N/end.json'), candidate=load(run / 'attempts/r1-003-g0-I/end.json'), controlAdapted=False), None))

rows = []
for name, request, expected_counts in requests:
    request['policy'] = bind(policy_path)
    request_path = P / 'inputs' / f'{name}-request.json'; save(request_path, request)
    reserve('comparatorReservations', 'maxComparatorCalls')
    before = len(events); start = time.monotonic_ns()
    report = owner.compare(request)
    assert report['status'] == 'equivalent' and report['policySHA256'] == bind(policy_path)['sha256']
    if expected_counts is not None: assert report['counts'] == expected_counts
    helpers = sum(event['java'] for event in events[before:])
    assert helpers == report['metadataJVMStarts'] == 1
    row = dict(name=name, status='verified', request=bind(request_path), report=report, durationNS=time.monotonic_ns()-start, subprocesses=events[before:], newBuilds=0)
    save(P / 'receipts' / f'{name}-comparison.json', row); rows.append(row)
    print(name, 'equivalent', report['counts'], flush=True)
save(P / 'analysis/retained-comparisons.json', dict(status='verified', comparisons=rows, newBuilds=0, comparatorJVMs=sum(e['java'] for e in events), claim='Output equivalence and reuse only; no new timing or speedup evidence.'))
