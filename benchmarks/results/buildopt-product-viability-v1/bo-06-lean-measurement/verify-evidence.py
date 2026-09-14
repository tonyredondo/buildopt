#!/usr/bin/env python3
"""Check the portable records and arithmetic; no builds or output JVMs."""
from pathlib import Path
import gzip
import hashlib
import json
import math
import tarfile

ROOT = Path(__file__).resolve().parent


def read(relative):
    return json.loads((ROOT / relative).read_text())


def digest(data):
    return hashlib.sha256(data).hexdigest()


def same_number(actual, expected):
    assert math.isclose(actual, expected, rel_tol=1e-12, abs_tol=1e-9), (actual, expected)


index = read('evidence-manifest.json')
seen = set()
for entry in index['files']:
    relative = entry['export']
    assert relative not in seen and '..' not in Path(relative).parts
    seen.add(relative)
    path = ROOT / relative
    data = path.read_bytes()
    assert len(data) == entry['bytes'] and digest(data) == entry['sha256'], relative
    if 'original' in entry:
        original = gzip.decompress(data) if entry.get('compression') == 'gzip' else data
        assert digest(original) == entry['original']['sha256'], relative

members = []
with tarfile.open(ROOT / 'runner-source.tar.gz') as archive:
    for member in archive:
        assert not member.issym() and not member.islnk()
        assert not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
        if member.isfile():
            content = archive.extractfile(member).read()
            members.append(dict(path=member.name, sha256=digest(content), bytes=len(content)))
assert members == read('source-archive-members.json')

base = Path('profiles/control/run')
manifest = read(base / 'manifest.json')
result = read(base / 'result.json')
summary = read('analysis/control-summary-v4.json')
assert manifest['phase'] == 'QUALIFICATION' and manifest['mode'] == 'P_LEAN_RESEARCH_V1'
assert manifest['controlBaseline']['files'] == manifest['baseline']['files'] + manifest['candidate']['files']
assert manifest['replications'] == 1 and manifest['executionEnd'] == 3
assert result['manifestSHA256'] == digest((ROOT / base / 'manifest.json').read_bytes())
assert result['workflowStarts'] == result['actualGradleStarts'] == result['gradleReservations'] == 8
assert result['nestedStarts'] == result['unknownGradleReservations'] == 0
assert len(result['slots']) == 1 and len(result['slots'][0]) == 4
for ordinal, slot in enumerate(result['slots'][0]):
    pair = read(base / 'pairs' / f'r1-{ordinal:03d}.json')
    assert pair['slot'] == slot and slot['class'] == 'COMPARABLE'
    assert slot['ownerMetadataJVMStarts'] == 1 and slot['extraCandidateNS'] == 0
    for arm in ('N', 'I'):
        attempt = base / 'attempts' / f'r1-{ordinal:03d}-g0-{arm}'
        end = read(attempt / 'end.json')
        native = read(attempt / 'native-finish.json')
        assert native['exitCode'] == 0 and native['outcome'] == 'EXITED'
        assert end['durationNS'] == end['end']['ns'] - end['begin']['ns']
        assert end['begin']['ns'] <= native['start']['ns'] <= native['end']['ns'] <= end['end']['ns']
        assert digest((ROOT / attempt / 'stdout.log').read_bytes()) == native['stdoutSHA256']
        assert digest((ROOT / attempt / 'stderr.log').read_bytes()) == native['stderrSHA256']
        assert end['durationNS'] == slot['nativeNS' if arm == 'N' else 'candidateNS']

slots = result['slots'][0][1:]
native_ns = sum(row['nativeNS'] for row in slots)
candidate_ns = sum(row['candidateNS'] for row in slots)
costs = []
for path in (ROOT / base / 'costs').glob('*.json'):
    cost = json.loads(path.read_text())
    if 'durationNS' in cost:
        assert cost['durationNS'] == cost['endNS'] - cost['startNS'] >= 0
        costs.append(cost)
outside_ns = sum(c['durationNS'] for c in costs if c['class'] == 'customer-machine' and not c['insideEnvelope'])
difference = abs(native_ns - candidate_ns)
assert not (difference >= 3e9 and difference >= .05 * min(native_ns, candidate_ns))
same_number(summary['nativeSeconds'], native_ns / 1e9)
same_number(summary['candidateSeconds'], candidate_ns / 1e9)
same_number(summary['controlAbsoluteDifferenceSeconds'], difference / 1e9)
same_number(summary['controlDifferencePercentOfFaster'], 100 * difference / min(native_ns, candidate_ns))
same_number(summary['netSavedSeconds'], (native_ns - candidate_ns - outside_ns) / 1e9)
same_number(summary['netSecondsPerScheduled'], (native_ns - candidate_ns - outside_ns) / 3e9)
assert summary['decision'] == 'CONTROL_PASSED' and summary['passed']
assert summary['recordedWorkflowOnly'] and not summary['ordinaryGradleClaim']

policy = read('inputs/measured-owner-policy.json')
diagnosis = read('analysis/prefix-admission-diagnosis.json')
assert policy['qualification'] == {'path': '', 'sha256': ''}
assert diagnosis['currentComparator'] == policy['comparator']
assert diagnosis['oldComparatorSHA256'] != policy['comparator']['sha256']
assert diagnosis['prefixWorkflowStarts'] == diagnosis['protectedWorkflowStarts'] == 0
assert read('receipts/prefix-validation.json')['exitCode'] == 1
closeout = read('receipts/allocation-closeout-v2.json')
assert closeout['allocationResult'] == 'INCOMPLETE_PREFIX_NOT_ADMITTED'
assert closeout['ownerStarts'] == closeout['comparisonJVMs'] == 8
assert closeout['prefixStarts'] == closeout['protectedStarts'] == closeout['retries'] == 0
assert closeout['fixtureStarts'] == 94 and closeout['limitsMet'] and closeout['sourcesUnchanged']
ledger_bytes = (ROOT / 'inputs/closed-allocation.json').read_bytes()
assert digest(ledger_bytes) == closeout['allocation']['sha256']
ledger = json.loads(ledger_bytes)
assert ledger['actualOwnerStarts'] == ledger['actualHelpers'] == 8
assert closeout['allocatedBytes'] < ledger['maxNewBytes']
assert closeout['freeBytes'] >= ledger['minimumFreeBytes']
assert len(closeout['units']) == 2 and all(u['exitCode'] != 0 for u in closeout['units'])
assert len(read('analysis/control-host-pressure.json')['requests']) == 8
assert read('receipts/control-closure.json')['livePairs'] == read('receipts/control-closure.json')['independentPairs'] == 4
print(f'PASS: {len(seen)} exported files; eight build records; control arithmetic; incomplete prefix retained.')
print('Full output reconstruction and current process checks require the retained local state.')
