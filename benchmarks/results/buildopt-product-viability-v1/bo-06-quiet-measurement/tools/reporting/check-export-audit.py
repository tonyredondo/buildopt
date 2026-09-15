"""Check that an internally rehashed export cannot change key closure claims."""
from pathlib import Path
import hashlib
import json
import shutil
import subprocess
import sys
import tempfile

root = Path(sys.argv[1]).resolve()
report = Path(sys.argv[2]).resolve()
results = []
# Copy-on-write edits preserve the original export and avoid copying large files.
for name in ('owner-count', 'protected-start', 'missing-refusal-file', 'false-prefix-pass'):
    with tempfile.TemporaryDirectory(prefix='audit-', dir=report.parent) as temporary:
        directory = Path(temporary) / 'evidence'
        shutil.copytree(root, directory, copy_function=__import__('os').link)
        index = json.loads((directory / 'evidence-manifest.json').read_text())
        def change(relative, apply):
            path = directory / relative
            value = json.loads(path.read_text()); apply(value)
            data = (json.dumps(value, indent=2) + '\n').encode()
            path.unlink(); path.write_bytes(data)
            row = next(row for row in index['files'] if row['export'] == relative)
            row['sha256'] = hashlib.sha256(data).hexdigest(); row['bytes'] = len(data)
            if 'original' in row: row['original']['sha256'] = row['sha256']
        if name == 'owner-count':
            change('inputs/closed-allocation.json', lambda value: value.update(actualOwnerStarts=26))
        elif name == 'protected-start':
            change('receipts/allocation-closeout.json', lambda value: value.update(protectedStarts=1))
        elif name == 'missing-refusal-file':
            index['missingAttemptFiles'].pop()
        else:
            index['decision'] = 'PREFIX_PASSED'
            for relative in ('inputs/closed-allocation.json', 'receipts/allocation-closeout.json'):
                change(relative, lambda value: value.update(decision='PREFIX_PASSED'))
        path = directory / 'evidence-manifest.json'; path.unlink()
        path.write_text(json.dumps(index, indent=2) + '\n')
        result = subprocess.run([sys.executable, '-B', str(directory / 'verify-evidence.py')], capture_output=True, text=True, timeout=120)
        assert result.returncode != 0 and 'AssertionError' in result.stderr, (name, result.stderr)
        results.append(dict(case=name, exitCode=result.returncode, rejected=True, error=result.stderr.splitlines()[-3:]))
report.write_text(json.dumps(dict(cases=results, passed=True, ownerBuilds=0, comparatorStarts=0), indent=2)+'\n')
print('All four rehashed false claims rejected; original export unchanged.')
