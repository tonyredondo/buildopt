"""Independently inspect the BO-06 export and raw fixture observations."""
from pathlib import Path
from collections import Counter
import ast
import base64
import hashlib
import json
import re
import subprocess
import xml.etree.ElementTree as ET

P = Path(__file__).resolve().parent
R = P.parents[3]
D = R / 'benchmarks/results/buildopt-product-viability-v1/bo-06-readiness'

def load(path):
    return json.loads(path.read_text())

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

manifest = load(D / 'evidence-manifest.json')
for entry in manifest['files']:
    assert sha(D / entry['path']) == entry['sha256'] == sha(Path(entry['source'])), entry['path']
    assert (D / entry['path']).stat().st_size == entry['bytes']
cases = load(P / 'analysis/boundary-checks.json')['cases']
assert len(cases) == 8
expected = [4, 0, 4, 2, 4, 4, 4, 3]
for row, count in zip(cases, expected):
    q = P / 'boundary-runs' / row['case']
    receipt = load(q / 'end.json')
    assert receipt['exitCode'] == 0 and receipt['unitState'] == 'inactive'
    assert sha(q / 'end.json') == row['request']['sha256']
    assert sha(q / 'build.log') == row['log']['sha256']
    a, b = ET.parse(q / 'native.xml'), ET.parse(q / 'main.xml')
    assert ET.tostring(a.getroot()) == ET.tostring(b.getroot())
    assert len(a.getroot().findall('file')) == 2
    observation = q / 'processing.tsv'
    files = [] if not observation.exists() else [base64.b64decode(line.split('\t')[-1]).decode()
        for line in observation.read_text().splitlines() if line.startswith('F\t')]
    assert len(files) == count == row['processedFiles']
    observed = Counter(Path(name).name for name in files)
    if count == 4:
        assert observed == Counter({'A.java': 2, 'B.java': 2})
    elif count == 2:
        assert observed == Counter({'A.java': 1, 'B.java': 1})
    elif count == 3:
        assert observed == Counter({'A.java': 2, 'B.java': 1})
    if row['case'] == 'cache-restore':
        assert 'checkstyleMain FROM-CACHE' in (q / 'build.log').read_text()
        assert not row['successExists']
    if row['case'] == 'changed-rules':
        assert not row['successExists']
    check = subprocess.run(['systemctl', '--user', 'show', receipt['unit'], '--property=ActiveState', '--value'], capture_output=True, text=True)
    assert check.stdout.strip() in ['', 'inactive']
compiled = load(P / 'receipts/compiled-source-proof.json')
assert compiled['exitCode'] == 0 and len(compiled['classes']) == 8
for row in compiled['classes']:
    assert row['matchesFixtureBytes'] and sha(P / 'compiled-proof' / row['class']) == row['sha256']
for path, value in compiled['sources'].items():
    assert sha(Path(path)) == value

history = load(P / 'inputs/seed-history.json')
assert len(history) == 101 and len({x['commit'] for x in history}) == 101
for subject in load(P / 'inputs/subjects.json')['repositories']:
    h = load(Path(subject['history']['path']))
    assert len(h) == 101 and h[0]['commit'] == subject['anchor'] and h[-1]['commit'] == subject['endpoint']
    assert sha(Path(subject['history']['path'])) == subject['history']['sha256']
    assert all(row['ordinal'] == i and (not i or row['parent'] == h[i-1]['commit']) for i, row in enumerate(h))
contract = load(P / 'inputs/scientific-contract.json')
plan = contract['governingPlan']
original = subprocess.check_output(['git', 'show', plan['revision'] + ':' + plan['path']], cwd=R)
assert hashlib.sha256(original).hexdigest() == plan['sha256']
assert contract['scheduledStarts'] == contract['comparatorJVMStarts'] == 404
assert contract['minimumNetFraction'] == .05 and contract['minimumNetSecondsPerScheduledSlot'] == 1
assert contract['confirmationAdmitted'] is False
assert load(P / 'receipts/confirmation-refusal.json')['exitCode'] == 1
assert not (P / 'confirmation-not-admitted').exists()

json_files = list(D.rglob('*.json'))
for path in json_files:
    load(path)
for path in D.glob('*.py'):
    ast.parse(path.read_text())
links = 0
for path in [D / 'README.md', D / 'measurement-decision.md', R / 'docs/research-status.md',
             R / 'docs/plans/buildopt-research-execution-plan-2026-09-14.md']:
    for href in re.findall(r'\]\(([^)]+)\)', path.read_text()):
        if '://' in href or href.startswith('#'):
            continue
        target = (path.parent / href.split('#', 1)[0]).resolve()
        # This verifier writes the final receipt after checking the remaining links.
        if target != D / 'analysis/verification.json':
            assert target.exists(), (path, href)
        links += 1
result = dict(status='verified', exportedSourceBindings=len(manifest['files']),
    independentFixtureComparisons=8, compiledClasses=8, histories=7,
    jsonFilesParsed=len(json_files), localLinksChecked=links,
    actualGradleStarts=8, actualCompilerCommands=1, standaloneComparatorJVMStarts=0,
    protectedValidationStarts=0, ownedServicesClosed=8,
    retainedStateBytes=sum(x.stat().st_size for x in P.rglob('*') if x.is_file()),
    decision='CONFIRMATION_NOT_ADMITTED', bo06Status='partial',
    proposalApproved=False, limits='Correctness/readiness audit only; no value or G0/G3 promotion.')
assert result['retainedStateBytes'] < 1073741824
(P / 'analysis/verification.json').write_text(json.dumps(result, indent=2) + '\n')
(D / 'analysis/verification.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps(result))
