#!/usr/bin/env python3
"""Check this incomplete control's records without running builds or comparisons."""
from pathlib import Path
import gzip
import hashlib
import json
import math
import tarfile

ROOT = Path(__file__).resolve().parent
index = json.loads((ROOT / 'evidence-manifest.json').read_text())
files = {row['export']: row for row in index['files']}
assert len(files) == len(index['files'])
originals = {}
for name, row in files.items():
    path = Path(name)
    assert not path.is_absolute() and '..' not in path.parts
    data = (ROOT / path).read_bytes()
    assert len(data) == row['bytes'] and hashlib.sha256(data).hexdigest() == row['sha256'], name
    if 'original' in row:
        plain = gzip.decompress(data) if row.get('compression') == 'gzip' else data
        assert hashlib.sha256(plain).hexdigest() == row['original']['sha256']
        originals[row['original']['path']] = (row['original']['sha256'], plain)

def raw(name):
    name = str(name)
    if name in files:
        return (ROOT / name).read_bytes()
    return gzip.decompress((ROOT / (name + '.gz')).read_bytes())

def read(name):
    return json.loads(raw(name))

def bound(binding):
    digest, data = originals[binding['path']]
    assert digest == binding['sha256']
    return json.loads(data)

def equal_number(actual, expected):
    assert math.isclose(actual, expected, rel_tol=1e-12, abs_tol=1e-9)

close = read('receipts/allocation-closeout.json')
allocation = read('inputs/closed-allocation.json')
summary = read('analysis/control-retained-summary.json')
assert close['status'] == 'verified' and allocation['status'] == 'closed verified'
assert close['decision'] == allocation['decision'] == summary['decision'] == index['decision'] == 'INCOMPLETE_CONTROL_QUIET_TIMEOUT'
assert summary['complete'] is False and summary['netSaving'] is None
assert close['ownerStarts'] == allocation['actualOwnerStarts'] == summary['requests'] == 1
assert close['comparisonJVMs'] == allocation['actualHelpers'] == summary['livePairs'] == summary['independentPairs'] == 0
assert close['ownerReservations'] == close['helperReservations'] == allocation['ownerReserved'] == allocation['helpersReserved'] == 8
assert close['protectedStarts'] == close['retries'] == summary['developmentStarts'] == 0
assert summary['scheduledBuilds'] == 8 and summary['unstartedBuilds'] == 7
assert len(close['units']) == 2
for unit in close['units']:
    assert unit['status'] in ('inactive', 'unknown') and not bound(unit['closure'])['remaining']
assert all(controller['originalProcessAbsent'] for controller in close['controllers'])
assert not any(name.startswith('profiles/prefix/') for name in files)
base = Path('profiles/control/run')
manifest = read(base / 'manifest.json')
result = read(base / 'result.json')
assert result == read(base / 'results/0001.json')
assert result['manifestSHA256'] == hashlib.sha256(raw(base / 'manifest.json')).hexdigest()
assert manifest['mode'] == 'P_LEAN_RESEARCH_V1' and manifest['phase'] == 'QUALIFICATION'
assert manifest['executionEnd'] == 3 and manifest['replications'] == 1
assert manifest['controlBaseline']['files'] == manifest['baseline']['files'] + manifest['candidate']['files']
assert result['workflowStarts'] == result['actualGradleStarts'] == 1
assert result['gradleReservations'] == 2 and result['unknownGradleReservations'] == 1
assert result['nestedStarts'] == 0 and result['decision'] == 'INCOMPLETE_EVIDENCE'
assert not result['replications'][0]['complete']
assert not any(name.startswith(str(base / 'pairs') + '/') for name in files)
assert sum(name.endswith('/native-finish.json') for name in files) == 1
assert len(result['slots']) == 1 and len(result['slots'][0]) == 4
assert result['slots'][0][0]['class'] == 'HARNESS_INVALID'
assert all(slot['class'] == 'NOT_RUN_DEPENDENCY' for slot in result['slots'][0][1:])
assert all(slot['ownerMetadataJVMStarts'] == 0 for slot in result['slots'][0])
completed = base / 'attempts/r1-000-g0-N'
finish = read(completed / 'end.json')
native = bound(finish['native'])
assert native == read(completed / 'native-finish.json')
assert native['exitCode'] == 0 and native['outcome'] == 'EXITED'
assert finish['durationNS'] == finish['end']['ns'] - finish['begin']['ns'] > 0
assert finish['begin']['ns'] <= native['start']['ns'] <= native['end']['ns'] <= finish['end']['ns']
for stream in ('stdout', 'stderr'):
    assert hashlib.sha256(raw(completed / (stream + '.log'))).hexdigest() == native[stream + 'SHA256']
assert len(summary['coverage']) == 8
for position, row in enumerate(summary['coverage']):
    ordinal, arm = position // 2, ('N', 'I')[position % 2]
    assert row['originalOrdinal'] == ordinal + 17 and row['arm'] == arm
    expected = ('COMPLETED_UNPAIRED' if arm == 'N' else 'NOT_RUN_QUIET') if ordinal == 0 else 'NOT_RUN_DEPENDENCY'
    assert row['outcome'] == expected
    if position == 0:
        equal_number(row['wholeRequestSeconds'], finish['durationNS'] / 1e9)
    else:
        assert row['wholeRequestSeconds'] is None
refused = base / 'attempts/r1-000-g0-I'
assert sorted(name[len(str(refused))+1:] for name in files if name.startswith(str(refused)+'/')) == ['quiet-start.json', 'start.json']
quiet = read(refused / 'quiet-start.json')
policy = bound(quiet['policy'])
assert quiet['policy'] == manifest['quietStart'] and quiet['manifestSHA256'] == result['manifestSHA256']
assert quiet['decision'] == 'NOT_QUIET_TIMEOUT' and quiet['reason'] == 'quiet-start deadline'
assert (policy['windowNS'], policy['maximumWaitNS'], policy['maximumPressurePPM']) == (30_000_000_000, 180_000_000_000, 100000)
equal_number(summary['refusal']['seconds'], (quiet['end']['ns']-quiet['begin']['ns']) / 1e9)
intervals = []
for before, after in zip(quiet['samples'], quiet['samples'][1:]):
    ns = after['end']['ns'] - before['end']['ns']
    assert 0 < ns <= policy['maximumGapNS']
    assert after['end']['ns'] - after['begin']['ns'] <= policy['maximumReadNS']
    intervals.append(dict(endSeconds=(after['end']['ns'] - quiet['begin']['ns']) / 1e9,
                          seconds=ns / 1e9, pressure={key:(after['totalsUS'][key]-before['totalsUS'][key])*1000/ns for key in ('cpu','io','memory')}))
assert intervals == summary['refusal']['intervalsData']
assert len(intervals) == summary['refusal']['intervals']
for key, row in summary['refusal']['pressure'].items():
    assert row['intervalsAboveThreshold'] == sum(x['pressure'][key] > .1 for x in intervals)
    equal_number(row['weightedFraction'], sum(x['pressure'][key]*x['seconds'] for x in intervals)/sum(x['seconds'] for x in intervals))
    equal_number(row['maximumFraction'], max(x['pressure'][key] for x in intervals))
cpu = read(completed / 'supervisor-cpu.json')
disk = read(completed / 'disk-observer.json')
observation = read('analysis/disk-observations.json')['requests'][0]
assert disk['attempt'] == native['attempt'] and disk['status'] == observation['disk']
equal_number(observation['supervisorCPUSeconds'], sum(cpu['end'][k]-cpu['begin'][k] for k in ('userNS','systemNS')) / 1e9)
equal_number(observation['nativeSeconds'], (native['end']['ns']-native['start']['ns']) / 1e9)
equal_number(observation['supervisorCoreEquivalent'], observation['supervisorCPUSeconds']/observation['nativeSeconds'])
members = []
with tarfile.open(ROOT / 'runner-source.tar.gz') as archive:
    for member in archive:
        assert member.isfile() and '/' not in member.name
        data = archive.extractfile(member).read()
        members.append(dict(path=member.name, sha256=hashlib.sha256(data).hexdigest(), bytes=len(data), mode=member.mode))
assert members == read('source-archive-members.json')
entries = [dict(path=x['path'],kind='file',mode=x['mode'],size=x['bytes'],sha256=x['sha256'],target='') for x in members]
assert hashlib.sha256((json.dumps(entries,indent=2)+'\n').encode()).hexdigest() == manifest['package'][0]['sha256']
owner_policy = read('inputs/measured-owner-policy.json')
qualification = read('inputs/owner-qualification.json')
assert hashlib.sha256(raw('inputs/measured-owner-compare.py')).hexdigest() == owner_policy['comparator']['sha256'] == qualification['comparatorSHA256']
assert qualification['decision'] == 'OWNER_OUTPUT_POLICY_QUALIFIED'
print(f'Verified {len(files)} files: one build, zero comparison JVMs, seven unstarted control builds; no saving claimed.')
