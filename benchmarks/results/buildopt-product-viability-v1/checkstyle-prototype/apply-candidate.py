#!/usr/bin/env python3
"""Apply or remove this exact local prototype; never changes Git's index/history."""
import argparse
import hashlib
import json
from pathlib import Path
import stat
import subprocess
import sys


def digest(path):
    if path.is_symlink() or (path.exists() and not path.is_file()):
        raise ValueError('Unsupported owned path type: ' + str(path))
    return hashlib.sha256(path.read_bytes()).hexdigest() if path.exists() else None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('repository', type=Path)
    parser.add_argument('--inverse', action='store_true')
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    root = args.repository.resolve(strict=True)
    bundle = Path(__file__).resolve().parent
    manifest = json.loads((bundle / 'candidate-manifest.json').read_text())
    actual_root = subprocess.check_output(['git', 'rev-parse', '--show-toplevel'], cwd=root, text=True).strip()
    if Path(actual_root) != root:
        raise ValueError('An exact Git worktree root is required')
    for name, key in [('candidate.patch', 'patchSHA256'), ('inverse.patch', 'inverseSHA256')]:
        if digest(bundle / name) != manifest[key]:
            raise ValueError('Patch integrity mismatch: ' + name)
    before_key, after_key = ('postimageSHA256', 'preimageSHA256') if args.inverse else ('preimageSHA256', 'postimageSHA256')
    states = []
    for item in manifest['files']:
        path = root / item['path']
        if not path.parent.resolve().is_relative_to(root):
            raise ValueError('Owned path escapes the worktree')
        actual = digest(path)
        if actual is not None and stat.S_IMODE(path.stat().st_mode) != item['mode']:
            raise ValueError('Owned mode drift: ' + item['path'])
        states.append((actual == item[before_key], actual == item[after_key]))
    if all(after for before, after in states):
        print(json.dumps({'status': 'already removed' if args.inverse else 'already applied', 'writes': 0}))
        return
    if not all(before for before, after in states):
        raise ValueError('Owned source is drifted or partially applied; nothing was written')
    if not args.inverse:
        for relative, expected in manifest['prerequisites'].items():
            if digest(root / relative) != expected:
                raise ValueError('Unsupported source prerequisite: ' + relative)
    patch = bundle / ('inverse.patch' if args.inverse else 'candidate.patch')
    subprocess.run(['git', 'apply', '--check', '--', str(patch)], cwd=root, check=True)
    if args.check:
        print(json.dumps({'status': 'applicable inverse' if args.inverse else 'applicable', 'writes': 0}))
        return
    subprocess.run(['git', 'apply', '--', str(patch)], cwd=root, check=True)
    for item in manifest['files']:
        if digest(root / item['path']) != item[after_key]:
            raise ValueError('Postimage verification failed: ' + item['path'])
    print(json.dumps({'status': 'removed' if args.inverse else 'applied', 'verifiedPaths': len(states)}))


if __name__ == '__main__':
    try:
        main()
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(2)
