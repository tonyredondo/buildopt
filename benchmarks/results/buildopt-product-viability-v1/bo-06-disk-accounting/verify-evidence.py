#!/usr/bin/env python3
"""Check the published fixture evidence without starting builds or helpers."""

import hashlib
import json
from pathlib import Path
import tarfile

ROOT = Path(__file__).resolve().parent
read = lambda name: json.loads((ROOT / name).read_text())
sha = lambda raw: hashlib.sha256(raw).hexdigest()
manifest = read('evidence-manifest.json')
for name, expected in manifest['files'].items():
    path = Path(name)
    assert not path.is_absolute() and '..' not in path.parts
    assert sha((ROOT / path).read_bytes()) == expected, name

freeze = read('source-freeze.json')
assert sha((ROOT / 'runner-source.tar.gz').read_bytes()) == freeze['archiveSHA256']
assert sha((ROOT / 'poc-product-viability-v1.json').read_bytes()) == freeze['protocolSHA256']
with tarfile.open(ROOT / 'runner-source.tar.gz') as archive:
    members = archive.getmembers()
    assert len(members) == len(freeze['members'])
    assert all(member.isfile() and Path(member.name).name == member.name for member in members)
    assert {member.name: sha(archive.extractfile(member).read()) for member in members} == freeze['members']
for name, expected in freeze['externalInputs'].items():
    assert sha((ROOT / name).read_bytes()) == expected

index = read('fixture-records-index.json')
assert sha((ROOT / 'fixture-records.tar.gz').read_bytes()) == index['archiveSHA256']
with tarfile.open(ROOT / 'fixture-records.tar.gz') as archive:
    members = archive.getmembers()
    assert all(member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts for member in members)
    assert len(members) == len(index['members'])
    assert {member.name: sha(archive.extractfile(member).read()) for member in members} == index['members']
    starts = sorted(member.name for member in members if member.name.endswith('/native-pid.json'))
verification = read('verification.json')
assert starts == verification['nativeStartReceipts']
assert len(starts) == verification['totalNativeFixtureStarts'] == 24
assert verification['ownerBuilds'] == verification['comparatorJVMs'] == verification['protectedSourceReads'] == 0
assert len(verification['checks']) == 7
assert all(row['exitCode'] == 0 and row['binarySHA256'] == freeze['fixtureExecutableSHA256'] for row in verification['checks'])

rows = [json.loads(line) for line in (ROOT / 'receipts/disk-performance.jsonl').read_text().splitlines()]
assert len(rows) == verification['performanceCases'] == 8
assert {(row['files'], row['version'], row['repeat']) for row in rows} == {(count, version, repeat) for count in (16, 32768) for version in ('full', 'notified') for repeat in (1, 2)}
for row in rows:
    assert row['checks'] == 20 and row['elapsedNS'] > 0 and row['cpuNS'] > 0
    assert 0 < row['firstCheckCPUNS'] <= row['cpuNS']
    if row['version'] == 'notified':
        assert row['observer']['fallbackReason'] == '' and row['observer']['fullScans'] == 0
for repeat in (1, 2):
    pair = {row['version']: row for row in rows if row['files'] == 32768 and row['repeat'] == repeat}
    assert pair['notified']['cpuNS'] < pair['full']['cpuNS']

admission = read('next-control-admission.json')
assert admission['status'] == 'verified' and not admission['priorControlCanQualifyNewIdentity']
assert admission['measurementIdentity'] != admission['previousIdentity']
control, development = admission['proposals']
assert control['exitCode'] == 0 and development['exitCode'] != 0
assert control['identity'] == development['identity'] == admission['measurementIdentity']
assert control['runRootAbsent'] and development['runRootAbsent']
assert read('quiet-start-policy.json')['maximumPressurePPM'] == 100000
print('Disk accounting evidence verified: 24 fixture starts, eight cost cases, no project builds.')
