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
assert close['decision'] == allocation['decision'] == index['decision']
assert close['protectedStarts'] == close['retries'] == 0
assert close['sourcesUnchanged'] and close['recordedWorkflowOnly'] and not close['ordinaryGradleClaim']
assert not any(row['status'] not in ('inactive', 'unknown') for row in close['units'])
owner_starts = comparisons = 0
control = None
for trial in close['trials']:
    label = trial['label']; base = Path('profiles') / label / 'run'
    manifest = read(base / 'manifest.json'); result = read(base / 'result.json')
    summary = read('analysis/' + label + '-summary-v4.json')
    end = read('receipts/' + label + '-end.json')
    closure = read('receipts/' + label + '-closure.json')
    count = manifest['executionEnd'] + 1
    assert count == (4 if label == 'control' else 21)
    assert manifest['phase'] == ('QUALIFICATION' if label == 'control' else 'ENGINEERING')
    assert manifest['mode'] == 'P_LEAN_RESEARCH_V1' and manifest['replications'] == 1
    assert manifest['package'][0]['sha256'] == source_digest
    assert manifest['limits']['retryPairs'] == 0
    assert end['exitCode'] == 0 and end['observationFailure'] is None
    assert result['manifestSHA256'] == hashlib.sha256(raw(base / 'manifest.json')).hexdigest()
    assert result['workflowStarts'] == result['actualGradleStarts'] == result['gradleReservations'] == 2 * count
    assert result['unknownGradleReservations'] == result['nestedStarts'] == 0
    assert len(result['slots']) == 1 and len(result['slots'][0]) == count
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
    outside = sum(c['durationNS'] for c in costs if c['class'] == 'customer-machine' and not c['insideEnvelope'])
    for ordinal, slot in enumerate(result['slots'][0]):
        pair = read(base / 'pairs' / f'r1-{ordinal:03d}.json')
        assert pair['slot'] == slot and slot['class'] in ('COMPARABLE', 'NATIVE_RETAINED')
        assert slot['ownerMetadataJVMStarts'] == 1 and slot['extraCandidateNS'] == 0
        order = []
        for binding in pair['attempts']:
            finish = json.loads(bound(binding)); native = json.loads(bound(finish['native']))
            assert native['exitCode'] == 0 and native['outcome'] == 'EXITED'
            assert finish['durationNS'] == finish['end']['ns'] - finish['begin']['ns']
            assert finish['begin']['ns'] <= native['start']['ns'] <= native['end']['ns'] <= finish['end']['ns']
            attempt = Path(binding['path']).parent.name
            start = read(base / 'attempts' / attempt / 'start.json'); arm = start['arm']
            assert hashlib.sha256(raw(base / 'attempts' / attempt / 'stdout.log')).hexdigest() == native['stdoutSHA256']
            assert hashlib.sha256(raw(base / 'attempts' / attempt / 'stderr.log')).hexdigest() == native['stderrSHA256']
            assert finish['durationNS'] == slot['nativeNS' if arm == 'N' else 'candidateNS']
            order.append((native['start']['ns'], arm))
        assert [arm for _, arm in sorted(order)] == (['N', 'I'] if ordinal % 2 == 0 else ['I', 'N'])
    native_ns = sum(s['nativeNS'] for s in result['slots'][0][1:])
    candidate_ns = sum(s['candidateNS'] for s in result['slots'][0][1:])
    saving = native_ns - candidate_ns - outside
    equal_number(summary['nativeSeconds'], native_ns / 1e9)
    equal_number(summary['candidateSeconds'], candidate_ns / 1e9)
    equal_number(summary['outsideRequestSeconds'], outside / 1e9)
    equal_number(summary['netSavedSeconds'], saving / 1e9)
    equal_number(summary['netSavingPercent'], 100 * saving / native_ns)
    equal_number(summary['netSecondsPerScheduled'], saving / (count - 1) / 1e9)
    if label == 'control':
        control = manifest
        assert manifest['controlBaseline']['files'] == manifest['baseline']['files'] + manifest['candidate']['files']
        difference = abs(native_ns - candidate_ns)
        passed = not (difference >= 3e9 and difference >= .05 * min(native_ns, candidate_ns))
        expected = 'CONTROL_PASSED' if passed else 'CONTROL_MATERIAL_DIFFERENCE'
    else:
        assert len(manifest['history']) == 101 and manifest['prefixEnd'] == manifest['executionEnd'] == 20
        assert manifest['outputs'] == control['outputs'] and manifest['candidate'] == control['candidate']
        readiness = json.loads(bound(manifest['measurementReadiness']))
        proof = readiness['controlProof']
        assert json.loads(bound(proof['manifest'])) == control
        assert [r['commit'] for r in control['history']] == [r['commit'] for r in manifest['history'][17:21]]
        passed = saving >= 20e9 and saving >= .05 * native_ns
        expected = 'PREFIX_PASSED' if passed else 'PREFIX_INSUFFICIENT_SAVING'
    assert summary['passed'] == passed and summary['decision'] == trial['decision'] == expected
    owner_starts += 2 * count; comparisons += 2 * count
assert owner_starts == close['ownerStarts'] == close['ownerReservations'] == allocation['actualOwnerStarts']
assert comparisons == close['comparisonJVMs'] == close['helperReservations'] == allocation['actualHelpers']
assert close['decision'] == close['trials'][-1]['decision']
assert len(close['trials']) == (2 if read('analysis/control-summary-v4.json')['passed'] else 1)
policy = read('inputs/measured-owner-policy.json')
qualification = read('inputs/owner-qualification.json')
assert hashlib.sha256(raw('inputs/measured-owner-compare.py')).hexdigest() == policy['comparator']['sha256'] == qualification['comparatorSHA256']
assert hashlib.sha256(raw('inputs/owner-qualification.json')).hexdigest() == policy['qualification']['sha256']
assert qualification['decision'] == 'OWNER_OUTPUT_POLICY_QUALIFIED'
print(f"Verified {len(exports)} evidence files, {owner_starts} builds and {comparisons} recorded comparator calls: {close['decision']}")
