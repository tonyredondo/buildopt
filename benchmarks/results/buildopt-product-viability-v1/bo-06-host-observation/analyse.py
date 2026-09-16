"""Reconstruct quiet windows from the complete host-only trace.

Uses the unchanged quietWindow interval rule in the recorded runner source.
This diagnostic cannot admit a build: it has no worker or launch-age receipt.
"""
from pathlib import Path
import hashlib
import json
import sys

ROOT = Path(__file__).resolve().parent


def analyse():
    record = json.loads((ROOT / 'observation.json').read_text())
    allocation = json.loads((ROOT / 'allocation.json').read_text())
    policy = json.loads((ROOT / 'policy.json').read_text())
    assert hashlib.sha256((ROOT / 'policy.json').read_bytes()).hexdigest() == allocation['policySHA256']
    assert hashlib.sha256((ROOT / 'observe.py').read_bytes()).hexdigest() == allocation['observerSHA256']
    assert policy['maximumPressurePPM'] == 100000
    samples = record['samples']
    assert record['status'] == 'COMPLETE' and len(samples) == allocation['sampleCount']
    assert allocation['durationNS'] <= record['endBootNS'] - record['startBootNS'] <= allocation['hardMaximumNS']
    intervals = []
    for i, sample in enumerate(samples):
        assert sample['begin']['boot'] == sample['end']['boot'] == record['boot']
        assert 0 <= sample['end']['ns'] - sample['begin']['ns'] <= policy['maximumReadNS']
        assert set(sample['totalsUS']) == {'cpu', 'io', 'memory'}
        for key, total in sample['totalsUS'].items():
            line = next(line for line in sample['raw'][key].splitlines() if line.startswith('some '))
            assert int(dict(field.split('=') for field in line.split()[1:])['total']) == total >= 0
        if i == 0:
            continue
        previous = samples[i-1]
        gap = sample['end']['ns'] - previous['end']['ns']
        assert 0 < gap <= policy['maximumGapNS'] and sample['begin']['ns'] >= previous['end']['ns']
        delta = {key: sample['totalsUS'][key]-previous['totalsUS'][key] for key in ('cpu','io','memory')}
        assert min(delta.values()) >= 0
        intervals.append({'sampleIndex': i, 'gapNS': gap, 'deltaUS': delta,
                          'pressureFraction': {key: value*1000/gap for key,value in delta.items()},
                          'aboveThreshold': [key for key,value in delta.items() if value > gap//10000]})
    windows = []
    for i, sample in enumerate(samples):
        starts = [j for j in range(i+1) if samples[j]['end']['ns'] <= sample['end']['ns']-policy['windowNS']]
        decision = 'INCOMPLETE_WINDOW'
        if starts:
            decision = 'PRESSURE_IN_WINDOW' if any(row['aboveThreshold'] for row in intervals[starts[-1]:i]) else 'QUIET_WINDOW'
        windows.append({'sampleIndex': i, 'elapsedSeconds': (sample['end']['ns']-record['startBootNS'])/1e9, 'decision': decision})
    quiet = [row for row in windows if row['decision'] == 'QUIET_WINDOW']
    checks = record['processChecks']
    process_clear = all(not x['indexers'] and not x['unavailable'] and not x['oldCgroupExists'] and not any(p['sameIdentity'] for p in x['oldProcesses']) for x in checks)
    return {'schema': 'buildopt.host-only-analysis/v1', 'observationStatus': record['status'],
            'decision': 'HOST_ONLY_QUIET_WINDOWS_OBSERVED' if quiet and process_clear else 'HOST_ONLY_QUIET_UNCONFIRMED',
            'sampleCount': len(samples), 'intervalCount': len(intervals),
            'elapsedSeconds': (record['endBootNS']-record['startBootNS'])/1e9,
            'sampledSeconds': sum(row['gapNS'] for row in intervals)/1e9,
            'observerCPUSeconds': record['observerCPUSeconds'],
            'quietWindowCount': len(quiet), 'firstQuietSeconds': quiet[0]['elapsedSeconds'] if quiet else None,
            'finalWindowQuiet': windows[-1]['decision'] == 'QUIET_WINDOW',
            'processChecks': len(checks), 'processChecksClear': process_clear,
            'pressure': {key: {'weightedPercent': sum(row['deltaUS'][key] for row in intervals)*100000/sum(row['gapNS'] for row in intervals),
                               'maximumPercent': max(row['pressureFraction'][key] for row in intervals)*100,
                               'intervalsAboveThreshold': sum(key in row['aboveThreshold'] for row in intervals)}
                         for key in ('cpu','io','memory')},
            'intervals': intervals, 'windows': windows,
            'projectBuilds': 0, 'comparisonJVMs': 0,
            'causalAttribution': 'unproven', 'buildAdmission': False, 'performanceResult': None}


if __name__ == '__main__':
    result = analyse()
    if sys.argv[1:] == ['--verify']:
        assert result == json.loads((ROOT / 'analysis.json').read_text())
        manifest = ROOT / 'evidence-manifest.json'
        if manifest.exists():
            for item in json.loads(manifest.read_text())['files']:
                path = (ROOT / item['path']).resolve()
                assert path.is_relative_to(ROOT)
                assert hashlib.sha256(path.read_bytes()).hexdigest() == item['sha256']
        print(f"Verified {result['sampleCount']} samples and {result['quietWindowCount']} quiet windows; no build admission.")
    else:
        print(json.dumps(result, indent=2))
