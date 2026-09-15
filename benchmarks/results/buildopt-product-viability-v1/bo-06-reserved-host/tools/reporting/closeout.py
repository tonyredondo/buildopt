"""Close the reserved-host control that refused its first native launch."""
from pathlib import Path
import sys
sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from common import *
import datetime
import shutil
import subprocess

state = load(P / 'task-state.json')
assert state['pipeline']['status'] == 'stopped with retained failure'
allocation = load(P / 'allocation.json')
assert allocation['status'] == 'allocated'
run = P / 'profiles/control/run'
manifest = load(run / 'manifest.json')
result = load(run / 'result.json')
assert result == load(run / 'results/0001.json')
assert len(list((run / 'results').glob('*.json'))) == 1
assert result['manifestSHA256'] == bind(run / 'manifest.json')['sha256']
assert result['decision'] == 'INCOMPLETE_EVIDENCE'
assert result['workflowStarts'] == result['actualGradleStarts'] == result['nestedStarts'] == 0
assert result['gradleReservations'] == result['unknownGradleReservations'] == 1
assert len(result['slots']) == 1 and len(result['slots'][0]) == 4
assert result['slots'][0][0]['class'] == 'HARNESS_INVALID'
assert all(row['class'] == 'NOT_RUN_DEPENDENCY' for row in result['slots'][0][1:])
assert all(row['ownerMetadataJVMStarts'] == 0 for row in result['slots'][0])
assert not list((run / 'pairs').glob('*.json'))
assert not list((run / 'attempts').glob('*/native-start.json'))
assert not (P / 'profiles/prefix').exists()
for binding in load(P / 'inputs/control-freeze.json'):
    assert bind(binding['path']) == binding, binding['path']
identity = subprocess.check_output([manifest['executable']['path'], 'research-identity',
    str(P / 'profiles/control/manifest.json')], text=True, timeout=30).strip()
assert identity == '07ad4c609d81cdcc6f21351577757be88daf2db1e8d4ead08dd6aed94e93dcad'
for name, binding in load(P / 'inputs/sampler-copy-proof.json').items():
    assert bind(name)['sha256'] == binding['sha256'] == bind(binding['source'])['sha256']

refused = run / 'attempts/r1-000-g0-N'
assert [p.name for p in (run / 'attempts').iterdir()] == [refused.name]
assert {f.name for f in refused.iterdir()} == {'start.json', 'quiet-start.json'}
quiet = load(refused / 'quiet-start.json')
assert quiet['decision'] == 'NOT_QUIET_TIMEOUT' and quiet['reason'] == 'quiet-start deadline'
assert quiet['manifestSHA256'] == result['manifestSHA256']
assert quiet['policy'] == manifest['quietStart']
policy = load(manifest['quietStart']['path'])
assert (policy['windowNS'], policy['maximumWaitNS'], policy['maximumPressurePPM']) == (30_000_000_000, 180_000_000_000, 100000)
intervals = []
for before, after in zip(quiet['samples'], quiet['samples'][1:]):
    ns = after['end']['ns'] - before['end']['ns']
    assert 0 < ns <= policy['maximumGapNS']
    assert after['end']['ns'] - after['begin']['ns'] <= policy['maximumReadNS']
    pressure = {key: (after['totalsUS'][key] - before['totalsUS'][key]) * 1000 / ns
                for key in ('cpu', 'io', 'memory')}
    assert all(value >= 0 for value in pressure.values())
    intervals.append(dict(endSeconds=(after['end']['ns'] - quiet['begin']['ns']) / 1e9,
                          seconds=ns / 1e9, pressure=pressure))
pressure_summary = {key: dict(
    intervalsAboveThreshold=sum(row['pressure'][key] > .1 for row in intervals),
    weightedFraction=sum(row['pressure'][key] * row['seconds'] for row in intervals) /
                     sum(row['seconds'] for row in intervals),
    maximumFraction=max(row['pressure'][key] for row in intervals)) for key in ('cpu', 'io', 'memory')}
windows = []
samples = quiet['samples']
for last_index, last in enumerate(samples):
    candidates = [i for i in range(last_index + 1)
                  if samples[i]['end']['ns'] <= last['end']['ns'] - policy['windowNS']]
    if not candidates:
        windows.append('incomplete')
        continue
    first = candidates[-1]
    acceptable = all(samples[i]['totalsUS'][key] - samples[i-1]['totalsUS'][key] <=
                     (samples[i]['end']['ns'] - samples[i-1]['end']['ns']) // 10000
                     for i in range(first + 1, last_index + 1) for key in ('cpu', 'io', 'memory'))
    assert not acceptable, 'A qualifying window was missed'
    windows.append('pressure')
coverage = [dict(originalOrdinal=ordinal + 17, arm=arm,
    outcome='NOT_RUN_QUIET' if ordinal == 0 and arm == 'N' else 'NOT_RUN_DEPENDENCY',
    wholeRequestSeconds=None) for ordinal in range(4) for arm in ('N','I')]
costs = [load(f) for f in sorted((run / 'costs').glob('*.json')) if 'durationNS' in load(f)]
for cost in costs:
    assert cost['durationNS'] == cost['endNS'] - cost['startNS'] >= 0 and cost['class'] == 'research'
decision = 'INCOMPLETE_CONTROL_QUIET_TIMEOUT'
summary = dict(status='verified incomplete accounting', decision=decision, complete=False,
    netSaving=None, requests=0, scheduledBuilds=8, unstartedBuilds=8, livePairs=0,
    independentPairs=0, developmentStarts=0, protectedStarts=0, measuredPairs=0,
    coverage=coverage, costs=costs,
    refusal=dict(seconds=(quiet['end']['ns']-quiet['begin']['ns'])/1e9,
        intervals=len(intervals), pressure=pressure_summary, intervalsData=intervals, windowDecisions=windows,
        runnerCPUSeconds=sum(quiet['cpuEnd'][key]-quiet['cpuStart'][key] for key in ('userNS','systemNS'))/1e9),
    accountingLimit='The raw unknown reservation is preserved. The refusal precedes the first native launch.')
save(P / 'analysis/control-retained-summary.json', summary)

units = []
for config in (run / 'sessions').glob('*/worker-config.json'):
    cfg = load(config); closed = load(config.parent / 'closed.json')
    assert not closed['remaining'] and not (Path('/sys/fs/cgroup') / closed['cgroup'].lstrip('/')).exists()
    check = subprocess.run(['systemctl', '--user', 'is-active', cfg['unit']], capture_output=True, text=True, timeout=15)
    assert check.returncode != 0 and check.stdout.strip() in ('inactive', 'unknown')
    units.append(dict(unit=cfg['unit'], status=check.stdout.strip(), closure=bind(config.parent / 'closed.json')))
assert len(units) == 1
end = load(P / 'receipts/control-end.json')
assert end['exitCode'] == 1 and end['observationFailure'] is None and end['finishedNativeRequests'] == 0
controllers = []
for pid, ticks in [(end['pid'], end['procStartTicks']),
                   (end['samplerAfter']['pid'], end['samplerAfter']['identity']['startTicks'])]:
    path = Path('/proc') / str(pid) / 'stat'
    assert not path.exists() or int(path.read_text().rsplit(')', 1)[1].split()[19]) != ticks
    controllers.append(dict(pid=pid, startTicks=ticks, originalProcessAbsent=True))
assert not Path('/proc/' + str(state['pipeline']['pid'])).exists()
assert allocation['actualOwnerStarts'] == allocation['actualHelpers'] == 0
assert allocation['ownerReserved'] == allocation['helpersReserved'] == 8
allocated = int(subprocess.check_output(['du', '-s', '-B1', str(P)], text=True, timeout=60).split()[0])
diagnostics = (P / 'profiles/control/process-samples.jsonl').stat().st_size
now = datetime.datetime.now(datetime.timezone.utc); free = shutil.disk_usage(P).free
assert now < datetime.datetime.fromisoformat(allocation['deadlineUTC'])
assert allocated < allocation['maxNewBytes'] and free >= allocation['minimumFreeBytes']
assert diagnostics <= allocation['maxDiagnosticBytes']
allocation.update(status='closed verified', decision=decision, closedUTC=now.isoformat(),
                  allocatedBytes=allocated, diagnosticBytes=diagnostics)
atomic(P / 'allocation.json', allocation); save(P / 'inputs/closed-allocation.json', allocation)
close = dict(status='verified', decision=decision, ownerStarts=0, comparisonJVMs=0,
    ownerReservations=8, helperReservations=8, retries=0, protectedStarts=0,
    allocatedBytes=allocated, freeBytes=free, diagnosticBytes=diagnostics,
    allocation=bind(P / 'inputs/closed-allocation.json'), measurementIdentity=identity,
    trials=[dict(label='control', summary=bind(P / 'analysis/control-retained-summary.json'),
                 decision=decision, requests=0, livePairs=0, independentPairs=0)],
    units=units, controllers=controllers, sourcesUnchanged=True,
    recordedWorkflowOnly=True, ordinaryGradleClaim=False)
save(P / 'receipts/allocation-closeout.json', close)
state['steps'].update({'identical-code control':'partial', 'development prefix':'deferred',
                      'allocation closure':'verified', 'closure and publication':'in progress'})
state.update(decision=decision, runningProcesses=[], pipelineStage=None, actualHelpers=0,
             nextAction='Publish the retained refusal and environment observations. No more native timing; resolve disk activity before a new measurement decision.')
atomic(P / 'task-state.json', state)
print(json.dumps(dict(decision=decision, ownerStarts=0, comparisonJVMs=0, unitsClosed=len(units),
                      pressure=pressure_summary, allocatedBytes=allocated)))
