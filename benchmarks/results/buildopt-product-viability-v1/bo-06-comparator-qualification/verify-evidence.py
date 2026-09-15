#!/usr/bin/env python3
"""Check exported evidence and bindings without builds or comparator JVMs."""
from pathlib import Path
import hashlib
import json
import tarfile

ROOT = Path(__file__).resolve().parent

def read(path):
    return json.loads((ROOT / path).read_text())

def digest(data):
    return hashlib.sha256(data).hexdigest()

index = read('evidence-manifest.json')
originals = {}
seen = set()
for row in index['files']:
    path = Path(row['export'])
    assert not path.is_absolute() and '..' not in path.parts and str(path) not in seen
    seen.add(str(path))
    data = (ROOT / path).read_bytes()
    assert len(data) == row['bytes'] and digest(data) == row['sha256'], str(path)
    if 'original' in row:
        assert digest(data) == row['original']['sha256']
        originals[row['original']['path']] = (row['original']['sha256'], path)

def exported(binding):
    sha, path = originals[binding['path']]
    assert sha == binding['sha256']
    return path

for name in ('runner-source', 'comparator-fixtures'):
    members = []
    with tarfile.open(ROOT / (name + '.tar.gz')) as archive:
        for member in archive:
            assert member.isfile() and not Path(member.name).is_absolute()
            assert '..' not in Path(member.name).parts
            data = archive.extractfile(member).read()
            members.append(dict(path=member.name, sha256=digest(data), bytes=len(data), mode=member.mode))
    assert members == read(name + '-members.json')
    if name == 'runner-source':
        assert all('/' not in row['path'] for row in members)
        entries = [dict(path=x['path'], kind='file', mode=x['mode'], size=x['bytes'], sha256=x['sha256'], target='') for x in members]
        source_sha = digest((json.dumps(entries, indent=2) + '\n').encode())

policy = read('inputs/qualified-owner-policy.json')
old = read('inputs/measured-owner-policy.json')
assert old['qualification'] == {'path': '', 'sha256': ''}
assert {k:v for k,v in old.items() if k != 'qualification'} == {k:v for k,v in policy.items() if k != 'qualification'}
q = read(exported(policy['qualification']))
identities = [policy['schema'], policy['kind'], policy['reusePolicy']] + [policy[k]['sha256'] for k in ('baseContract','dateContract','interpreter','comparator','lexicalProjector','metadataJava')] + [b['sha256'] for b in policy['metadataClasspath']]
assert q['decision'] == 'OWNER_OUTPUT_POLICY_QUALIFIED'
assert q['comparatorSHA256'] == policy['comparator']['sha256'] == digest((ROOT / 'source/owner_compare.py').read_bytes())
assert q['readersSHA256'] == digest((json.dumps(identities, indent=2) + '\n').encode())
for binding in q['cases'] + q['nativeReuse']:
    exported(binding)

matrix = read('receipts/comparator-matrix.json')
assert matrix['status'] == 'verified' and matrix['tests'] == 33
assert matrix['failures'] == matrix['errors'] == matrix['skipped'] == matrix['builds'] == 0
assert matrix['source']['sha256'] == q['comparatorSHA256']
assert matrix['actualJavaStarts'] == 2
assert sum(row['java'] for row in matrix['subprocesses']) == 2
binding = read('receipts/reader-binding.json')
assert binding['exitCode'] == binding['ownerBuilds'] == binding['fixtureWorkflows'] == 0
assert binding['source']['sha256'] == source_sha
assert 'PASS' in (ROOT / exported(binding['log'])).read_text()

retained = read('analysis/retained-comparisons.json')
assert retained['status'] == 'verified' and len(retained['comparisons']) == 6
assert retained['newBuilds'] == 0 and retained['comparatorJVMs'] == 6
for row in retained['comparisons']:
    exported(row['request'])
    assert row['status'] == 'verified' and row['newBuilds'] == 0
    assert row['report']['status'] == 'equivalent' and row['report']['metadataJVMStarts'] == 1
    assert row['report']['policySHA256'] == digest((ROOT / 'inputs/measured-owner-policy.json').read_bytes())
    assert sum(e['java'] for e in row['subprocesses']) == 1
assert read('receipts/native-reuse-001-comparison.json')['report']['counts']['causalOrigins'] == 268
assert read('receipts/lean-control-3-comparison.json')['report']['counts']['causalOrigins'] == 256

admission = read('analysis/admission-matrix.json')
assert admission['status'] == 'verified' and len(admission['cases']) == 20
assert len({c['name'] for c in admission['cases']}) == 20
assert admission['ownerBuilds'] == admission['fixtureWorkflows'] == admission['comparatorJVMs'] == 0
for case in admission['cases']:
    exported(case['manifest'])
    assert case['runRootAbsent'] and case['syntheticTrialData']
    if case['expected'] == 'accepted':
        assert case['exitCode'] == 0
    else:
        assert case['exitCode'] != 0 and case['expected'] in case['stderr']
assert sum(c['exitCode'] == 0 for c in admission['cases']) == 3

freeze = read('analysis/measurement-freeze.json')
assert freeze['status'] == 'verified'
assert freeze['measurementIdentity'] != freeze['previousIdentity']
assert not freeze['oldControlCanQualifyNewPolicy']
control, development = [read(exported(row['manifest'])) for row in freeze['proposals']]
assert all(control[k] == development[k] for k in freeze['sharedFields'])
assert control['outputs']['owner'] == freeze['ownerPolicy']
assert control['package'][0]['sha256'] == source_sha
assert control['controlBaseline']['files'] == control['baseline']['files'] + control['candidate']['files']
assert len(development['history']) == 101 and development['executionEnd'] == development['prefixEnd'] == 20
assert [r['commit'] for r in control['history']] == [r['commit'] for r in development['history'][17:21]]
for row in freeze['proposals']:
    assert not row['syntheticTrialData'] and row['runRootAbsent']
    readiness = read(exported(read(exported(row['manifest']))['measurementReadiness']))
    assert readiness['identity'] == freeze['measurementIdentity']
    assert 'controlProof' not in readiness and 'prefixProof' not in readiness
assert freeze['proposals'][0]['exitCode'] == 0 and freeze['proposals'][1]['exitCode'] != 0
closed = read('inputs/closed-allocation.json')
assert closed['status'] == 'closed verified'
assert closed['actualOwnerBuilds'] == closed['retries'] == 0
assert closed['actualComparatorJVMs'] == closed['jvmReservations'] == 8
assert closed['goCommands'] == 1 and closed['allocatedBytes'] < closed['maxBytes']
proposal = read('inputs/next-measurement-proposal.json')
assert proposal['status'] == 'proposed-not-started'
assert proposal['ownerBuilds'] == proposal['controlBuilds'] + proposal['developmentBuilds'] == 50
assert proposal['comparatorJVMs'] == 2 * proposal['livePairs'] == 50
assert proposal['retries'] == proposal['protectedBuilds'] == 0
print(f"Verified {len(seen)} exported files, 33 comparator tests, six retained comparisons and 20 admission cases. No builds started.")
