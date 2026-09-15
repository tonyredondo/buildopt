#!/usr/bin/env python3
"""Audit exported records and arithmetic; never start builds or comparators."""
from pathlib import Path
import gzip
import hashlib
import json
import math
import tarfile

ROOT = Path(__file__).resolve().parent
index = json.loads((ROOT / 'evidence-manifest.json').read_text())
exports = {row['export']: row for row in index['files']}
assert len(exports) == len(index['files'])
originals = {}
for name, row in exports.items():
    path = Path(name)
    assert not path.is_absolute() and '..' not in path.parts
    data = (ROOT / name).read_bytes()
    assert len(data) == row['bytes'] and hashlib.sha256(data).hexdigest() == row['sha256'], name
    if 'original' in row:
        raw = gzip.decompress(data) if row.get('compression') == 'gzip' else data
        assert hashlib.sha256(raw).hexdigest() == row['original']['sha256']
        originals[row['original']['path']] = (row['original']['sha256'], raw)

def raw(name):
    if str(name) in exports:
        return (ROOT / name).read_bytes()
    row = exports[str(name) + '.gz']
    assert row['compression'] == 'gzip'
    return gzip.decompress((ROOT / (str(name) + '.gz')).read_bytes())

def read(name):
    return json.loads(raw(name))

def bound(binding):
    digest, data = originals[binding['path']]
    assert digest == binding['sha256']
    return data

def equal_number(actual, expected):
    assert math.isclose(actual, expected, rel_tol=1e-12, abs_tol=1e-9), (actual, expected)

members = []
with tarfile.open(ROOT / 'runner-source.tar.gz') as archive:
    for member in archive:
        assert member.isfile() and not Path(member.name).is_absolute()
        assert '..' not in Path(member.name).parts
        data = archive.extractfile(member).read()
        members.append(dict(path=member.name, sha256=hashlib.sha256(data).hexdigest(), bytes=len(data), mode=member.mode))
assert members == read('source-archive-members.json')
assert all('/' not in row['path'] for row in members)
entries = [dict(path=x['path'], kind='file', mode=x['mode'], size=x['bytes'], sha256=x['sha256'], target='') for x in members]
source_digest = hashlib.sha256((json.dumps(entries, indent=2) + '\n').encode()).hexdigest()

close = read('receipts/allocation-closeout.json')
allocation = read('inputs/closed-allocation.json')
assert close['status'] == 'verified' and allocation['status'] == 'closed verified'
assert close['decision'] == allocation['decision'] == index['decision'] == 'INCOMPLETE_DEVELOPMENT_QUIET_TIMEOUT'
assert close['protectedStarts'] == close['retries'] == 0
assert close['sourcesUnchanged'] and close['recordedWorkflowOnly'] and not close['ordinaryGradleClaim']
assert len(close['units']) == 4 and all(row['status'] in ('inactive', 'unknown') for row in close['units'])
assert [trial['label'] for trial in close['trials']] == ['control', 'prefix']
owner_starts = comparisons = 0
control = None
for trial in close['trials']:
    label = trial['label']; base = Path('profiles') / label / 'run'
    manifest = read(base / 'manifest.json'); result = read(base / 'result.json')
    summary = json.loads(bound(trial['summary']))
    end = read('receipts/' + label + '-end.json')
    count = manifest['executionEnd'] + 1
    completed_pairs = 4 if label == 'control' else 8
    actual_builds = 8 if label == 'control' else 17
    assert count == (4 if label == 'control' else 21)
    assert manifest['phase'] == ('QUALIFICATION' if label == 'control' else 'ENGINEERING')
    assert manifest['mode'] == 'P_LEAN_RESEARCH_V1' and manifest['replications'] == 1
    assert manifest['package'][0]['sha256'] == source_digest
    assert manifest['limits']['retryPairs'] == 0
    assert end['exitCode'] == (0 if label == 'control' else 1) and end['observationFailure'] is None
    assert result['manifestSHA256'] == hashlib.sha256(raw(base / 'manifest.json')).hexdigest()
    assert result == read(base / 'results/0001.json')
    assert sum(name.startswith(str(base / 'results') + '/') for name in exports) == 1
    assert result['workflowStarts'] == result['actualGradleStarts'] == actual_builds
    assert result['gradleReservations'] == (8 if label == 'control' else 18)
    assert result['unknownGradleReservations'] == (0 if label == 'control' else 1)
    assert result['nestedStarts'] == 0
    assert len(result['slots']) == 1 and len(result['slots'][0]) == count
    if label == 'control':
        closure = read('receipts/control-closure.json')
        assert closure['closed'] and closure['exitCode'] == 0
        assert closure['manifestSHA256'] == result['manifestSHA256']
        assert closure['livePairs'] == closure['independentPairs'] == count
        for evidence in closure['evidence']: bound(evidence)
    costs = []
    for name in exports:
        if name.startswith(str(base / 'costs') + '/') and name.endswith('.json'):
            value = read(name)
            if 'durationNS' in value:
                assert value['durationNS'] == value['endNS'] - value['startNS'] >= 0
                costs.append(value)
    for cost in costs:
        if cost['purpose'] == 'complete-request':
            continue
        resources = read(base / 'recorder-resources' / (cost['id'] + '.json'))
        before, after = resources['before'], resources['after']
        assert resources['phase'] == cost['id']
        assert (before['pid'], before['startTicks']) == (after['pid'], after['startTicks'])
        assert before['end']['ns'] <= cost['startNS'] <= cost['endNS'] <= after['begin']['ns']
        assert set(before['io']) == set(after['io']) == {'rchar', 'wchar', 'syscr', 'syscw', 'read_bytes', 'write_bytes', 'cancelled_write_bytes'}
        assert all(after['io'][key] >= value for key, value in before['io'].items())
        assert all(after['cpu'][key] >= value for key, value in before['cpu'].items())
    outside = sum(c['durationNS'] for c in costs if c['class'] == 'customer-machine' and not c['insideEnvelope'])
    policy = json.loads(bound(manifest['quietStart']))

    def audit_attempt(attempt):
        directory = base / 'attempts' / attempt
        finish = read(directory / 'end.json')
        native = json.loads(bound(finish['native']))
        assert native == read(directory / 'native-finish.json')
        assert native['exitCode'] == 0 and native['outcome'] == 'EXITED'
        assert finish['durationNS'] == finish['end']['ns'] - finish['begin']['ns']
        assert finish['begin']['ns'] <= native['start']['ns'] <= native['end']['ns'] <= finish['end']['ns']
        start = read(directory / 'start.json'); arm = start['arm']
        quiet = read(directory / 'quiet-start.json')
        assert quiet['decision'] == 'QUIET_START' and quiet['attempt'] == attempt
        assert quiet['manifestSHA256'] == result['manifestSHA256'] and quiet['policy'] == manifest['quietStart']
        last_sample = quiet['samples'][-1]['end']
        assert native['start']['boot'] == last_sample['boot']
        assert 0 <= native['start']['ns'] - last_sample['ns'] <= policy['maximumLaunchAgeNS']
        assert quiet['end']['ns'] <= finish['begin']['ns']
        wait_cost = next(c for c in costs if c['id'] == attempt + '-quiet-start')
        assert wait_cost['class'] == 'research' and not wait_cost['insideEnvelope']
        assert wait_cost['startNS'] <= quiet['begin']['ns'] <= quiet['end']['ns'] <= wait_cost['endNS']
        for stream in ('stdout', 'stderr'):
            assert hashlib.sha256(raw(directory / (stream + '.log'))).hexdigest() == native[stream + 'SHA256']
        return finish, native, arm

    for ordinal, slot in enumerate(result['slots'][0][:completed_pairs]):
        pair = read(base / 'pairs' / f'r1-{ordinal:03d}.json')
        assert pair['slot'] == slot and slot['class'] in ('COMPARABLE', 'NATIVE_RETAINED')
        assert slot['ownerMetadataJVMStarts'] == 1 and slot['extraCandidateNS'] == 0
        order = []
        for binding in pair['attempts']:
            attempt = Path(binding['path']).parent.name
            finish, native, arm = audit_attempt(attempt)
            assert finish == json.loads(bound(binding))
            assert finish['durationNS'] == slot['nativeNS' if arm == 'N' else 'candidateNS']
            order.append((native['start']['ns'], arm))
        assert [arm for _, arm in sorted(order)] == (['N', 'I'] if ordinal % 2 == 0 else ['I', 'N'])
    assert sum(name.startswith(str(base / 'pairs') + '/') for name in exports) == completed_pairs
    assert sum(name.startswith(str(base / 'attempts') + '/') and name.endswith('/native-finish.json') for name in exports) == actual_builds
    assert trial['requests'] == actual_builds
    assert trial['livePairs'] == trial['independentPairs'] == completed_pairs
    if label == 'control':
        native_ns = sum(s['nativeNS'] for s in result['slots'][0][1:])
        candidate_ns = sum(s['candidateNS'] for s in result['slots'][0][1:])
        saving = native_ns - candidate_ns - outside
        equal_number(summary['nativeSeconds'], native_ns / 1e9)
        equal_number(summary['candidateSeconds'], candidate_ns / 1e9)
        equal_number(summary['outsideRequestSeconds'], outside / 1e9)
        equal_number(summary['netSavedSeconds'], saving / 1e9)
        equal_number(summary['netSavingPercent'], 100 * saving / native_ns)
        equal_number(summary['netSecondsPerScheduled'], saving / (count - 1) / 1e9)
        control = manifest
        assert manifest['controlBaseline']['files'] == manifest['baseline']['files'] + manifest['candidate']['files']
        difference = abs(native_ns - candidate_ns)
        passed = not (difference >= 3e9 and difference >= .05 * min(native_ns, candidate_ns))
        assert summary['passed'] == passed is True
        assert summary['decision'] == trial['decision'] == 'CONTROL_PASSED'
    else:
        assert len(manifest['history']) == 101 and manifest['prefixEnd'] == manifest['executionEnd'] == 20
        assert manifest['outputs'] == control['outputs'] and manifest['candidate'] == control['candidate']
        readiness = json.loads(bound(manifest['measurementReadiness']))
        proof = readiness['controlProof']
        assert json.loads(bound(proof['manifest'])) == control
        assert [r['commit'] for r in control['history']] == [r['commit'] for r in manifest['history'][17:21]]
        assert result['decision'] == 'INCOMPLETE_EVIDENCE'
        assert not result['replications'][0]['complete']
        assert result['slots'][0][8]['class'] == 'HARNESS_INVALID'
        assert all(s['class'] == 'NOT_RUN_DEPENDENCY' for s in result['slots'][0][9:])
        assert summary['decision'] == trial['decision'] == close['decision']
        assert summary['complete'] is False and summary['netSaving'] is None
        assert summary['scheduledBuilds'] == 42 and summary['unstartedBuilds'] == 25
        assert len(summary['coverage']) == 42
        for position, row in enumerate(summary['coverage']):
            ordinal, arm = position // 2, ('N', 'I')[position % 2]
            assert (row['ordinal'], row['arm']) == (ordinal, arm)
            expected = 'COMPLETED_PAIR' if ordinal < 8 else ('COMPLETED_UNPAIRED' if ordinal == 8 and arm == 'N' else 'NOT_RUN_QUIET' if ordinal == 8 else 'NOT_RUN_DEPENDENCY')
            assert row['outcome'] == expected
            for evidence in row['evidence']: bound(evidence)
            if ordinal < 8:
                equal_number(row['wholeRequestSeconds'], result['slots'][0][ordinal]['nativeNS' if arm == 'N' else 'candidateNS'] / 1e9)
            elif expected == 'COMPLETED_UNPAIRED':
                finish, _, actual_arm = audit_attempt('r1-008-g0-N')
                assert actual_arm == 'N'
                equal_number(row['wholeRequestSeconds'], finish['durationNS'] / 1e9)
            else:
                assert row['wholeRequestSeconds'] is None
        refused = base / 'attempts/r1-008-g0-I'
        assert sorted(name[len(str(refused)) + 1:] for name in exports if name.startswith(str(refused) + '/')) == ['quiet-start.json', 'start.json']
        assert not any(c['id'] == 'r1-008-g0-I-request' for c in costs)
        quiet = read(refused / 'quiet-start.json')
        assert quiet['decision'] == 'NOT_QUIET_TIMEOUT' and quiet['reason'] == 'quiet-start deadline'
        assert quiet['manifestSHA256'] == result['manifestSHA256'] and quiet['policy'] == manifest['quietStart']
        equal_number(summary['refusal']['seconds'], (quiet['end']['ns'] - quiet['begin']['ns']) / 1e9)
        intervals = []
        for before, after in zip(quiet['samples'], quiet['samples'][1:]):
            ns = after['end']['ns'] - before['end']['ns']
            intervals.append(dict(seconds=ns / 1e9, pressure={key: (after['totalsUS'][key] - before['totalsUS'][key]) * 1000 / ns for key in ('cpu','io','memory')}))
        assert intervals == summary['refusal']['intervalsData']
        assert len(intervals) == summary['refusal']['intervals'] == 179
        for key, reported in summary['refusal']['pressure'].items():
            assert reported['intervalsAboveThreshold'] == sum(row['pressure'][key] > .10 for row in intervals)
            equal_number(reported['weightedFraction'], sum(row['pressure'][key] * row['seconds'] for row in intervals) / sum(row['seconds'] for row in intervals))
            equal_number(reported['maximumFraction'], max(row['pressure'][key] for row in intervals))
        assert all(row['pressure']['io'] > .10 for row in intervals)
        missing = {row['path'] for row in index['missingAttemptFiles']}
        assert missing == {str(refused / name) for name in ('end.json','native-start.json','native-pid.json','native-finish.json','stdout.log','stderr.log','supervisor-cpu.json','graph.jsonl')}
    owner_starts += actual_builds; comparisons += 2 * completed_pairs
assert owner_starts == close['ownerStarts'] == allocation['actualOwnerStarts'] == 25
assert comparisons == close['comparisonJVMs'] == allocation['actualHelpers'] == 24
assert close['ownerReservations'] == close['helperReservations'] == 50
assert owner_starts <= close['ownerReservations'] and comparisons <= close['helperReservations']
policy = read('inputs/measured-owner-policy.json')
qualification = read('inputs/owner-qualification.json')
assert hashlib.sha256(raw('inputs/measured-owner-compare.py')).hexdigest() == policy['comparator']['sha256'] == qualification['comparatorSHA256']
assert hashlib.sha256(raw('inputs/owner-qualification.json')).hexdigest() == policy['qualification']['sha256']
assert qualification['decision'] == 'OWNER_OUTPUT_POLICY_QUALIFIED'
print(f"Verified {len(exports)} evidence files, {owner_starts} builds and {comparisons} recorded comparator calls: {close['decision']}")
