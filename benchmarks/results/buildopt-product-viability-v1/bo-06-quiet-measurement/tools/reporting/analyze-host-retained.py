"""Apply the prospectively registered, symmetric shared-host flags to all requests."""
from pathlib import Path
import hashlib
import json

D = Path(__file__).resolve().parents[1]
import sys
label=sys.argv[1];assert label in ('control','prefix')
P = D / 'profiles' / label
expected_complete = 4 if label == 'control' else 21
offset=17 if label=='control' else 0


def load(path):
    return json.loads(path.read_text())


def busy(counters):
    return sum(counters) - counters[3] - counters[4]


def pressure(sample, resource):
    line = next(line for line in sample['host']['pressure'][resource].splitlines() if line.startswith('some '))
    return int(dict(item.split('=') for item in line.split()[1:])['total'])


samples = [json.loads(line) for line in (P / 'process-samples.jsonl').read_text().splitlines()]
intervals = []
for before, after in zip(samples, samples[1:]):
    start, end = before['endBootNS'], after['endBootNS']
    seconds = (end - start) / 1e9
    assert seconds > 0
    hz = after['host']['ticksPerSecond']
    host_seconds = (busy(after['host']['cpu']['cpu']) - busy(before['host']['cpu']['cpu'])) / hz
    previous = {group['unit']: group['cpuUsageUS'] for group in before['groups']}
    current = {group['unit']: group['cpuUsageUS'] for group in after['groups']}
    common = previous.keys() & current.keys()
    owned_seconds = sum(current[unit] - previous[unit] for unit in common) / 1e6
    assert owned_seconds >= 0
    # Creation/destruction intervals have only a lower bound for owned CPU.
    # Their external estimate is retained as an upper bound, never silently precise.
    ownership_complete = previous.keys() == current.keys()
    external_cores = max(0.0, host_seconds - owned_seconds) / seconds
    psi = {resource: (pressure(after, resource) - pressure(before, resource)) / (seconds * 1e6)
           for resource in ('cpu', 'io', 'memory')}
    reasons = []
    if external_cores > 4:
        reasons.append('external CPU > 4 cores')
    if psi['cpu'] > 0.10:
        reasons.append('CPU PSI some > 10%')
    if psi['io'] > 0.10:
        reasons.append('IO PSI some > 10%')
    intervals.append(dict(startBootNS=start, endBootNS=end, seconds=seconds,
                          hostBusyCoreEquivalents=host_seconds / seconds,
                          ownedCoreEquivalents=owned_seconds / seconds,
                          externalCoreEquivalents=external_cores,
                          ownershipComplete=ownership_complete,
                          pressure=psi, contended=bool(reasons), reasons=reasons,
                          samplingQualified=seconds <= 3))

requests = []
unstarted = []
for attempt in sorted((P / 'run/attempts').iterdir()):
    start = load(attempt / 'start.json')
    if not (attempt / 'native-finish.json').exists():
        quiet = load(attempt / 'quiet-start.json')
        assert quiet['decision'] == 'NOT_QUIET_TIMEOUT'
        unstarted.append(dict(attempt=attempt.name, reason=quiet['decision']))
        continue
    native = load(attempt / 'native-finish.json')
    begin, end = native['start']['ns'], native['end']['ns']
    duration = (end - begin) / 1e9
    selected = [(interval, max(0, min(end, interval['endBootNS']) - max(begin, interval['startBootNS'])) / 1e9)
                for interval in intervals if interval['startBootNS'] < end and interval['endBootNS'] > begin]
    coverage = sum(seconds for _, seconds in selected)
    missing_boundary_seconds=max(0,duration-coverage)
    sampling_qualified = all(interval['samplingQualified'] for interval, _ in selected)
    uncertain = sum(seconds for interval, seconds in selected if not interval['ownershipComplete'])
    flagged = sum(seconds for interval, seconds in selected if interval['contended'])
    definite = sum(seconds for interval, seconds in selected if interval['contended'] and
                   (interval['ownershipComplete'] or interval['pressure']['cpu'] > .10 or interval['pressure']['io'] > .10))
    def average(key):
        return sum(interval[key] * seconds for interval, seconds in selected) / duration
    requests.append(dict(attempt=attempt.name, replication=start['replication'],
                         originalOrdinal=start['ordinal'] + offset, arm=start['arm'],
                         nativeSeconds=duration, coverageSeconds=coverage, missingBoundarySeconds=missing_boundary_seconds,
                         maximumSampleGapSeconds=max((interval['seconds'] for interval, _ in selected),default=duration),
                         samplingQualified=sampling_qualified,
                         incompleteOwnershipSeconds=uncertain,
                         externalCPUAttributionQualified=sampling_qualified and uncertain == 0,
                         contendedFractionUpperBound=flagged / duration,
                         contendedFractionLowerBound=definite / duration,
                         contention=('FLAGGED' if definite / duration >= .20 else
                                     'POSSIBLY_FLAGGED' if flagged / duration >= .20 else 'BELOW_REGISTERED_FLAG'),
                         averageExternalCoreEquivalentsUpperBound=average('externalCoreEquivalents'),
                         averageHostBusyCoreEquivalents=average('hostBusyCoreEquivalents'),
                         averageOwnedCoreEquivalentsLowerBound=average('ownedCoreEquivalents'),
                         averagePressure={resource:sum(interval['pressure'][resource]*seconds for interval,seconds in selected)/duration
                                          for resource in ('cpu','io','memory')}))
assert len(requests) == len(list((P / 'run/attempts').glob('*/native-finish.json')))
assert len(requests) + len(unstarted) <= 2 * expected_complete
result = dict(status='verified observation and registered flag reconstruction', requests=requests, unstarted=unstarted, plannedBuilds=2*expected_complete, complete=len(requests)==2*expected_complete,
              samples=len(samples), intervals=intervals, analyzerSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              limits=['Shared CPU affinity is not host isolation',
                      'Counter reads are sequential and quantized; CPU estimates have sampling error',
                      'Host PSI includes the experiment itself and does not prove external interference',
                      'External workload is inferred; no specific process is identified',
                      'Owned service lifecycle gaps bound external CPU; they remain unqualified for precise attribution',
                      'Every interval and request retained; contention flags do not remove measurements'])
output = D / 'analysis' / f'{label}-host-pressure.json'
assert not output.exists()
output.write_text(json.dumps(result, indent=2) + '\n')
for request in requests:
    print(request['attempt'], request['contention'], 'flag fraction', round(request['contendedFractionUpperBound'], 3),
          'external cores upper', round(request['averageExternalCoreEquivalentsUpperBound'], 2),
          'gap', round(request['maximumSampleGapSeconds'], 3))
