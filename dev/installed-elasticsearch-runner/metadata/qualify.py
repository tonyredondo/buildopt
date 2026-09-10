#!/usr/bin/env python3
"""Build and qualify the retained-output reader using already installed tools.

This launches javac/java/jar only. It neither launches Gradle nor edits captures.
The output directory must be new. All inputs and logs are retained there.
"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import time


def sha(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def binding(path):
    return {'path': str(path), 'sha256': sha(path)}


def write(path, value):
    with path.open('x') as stream:
        json.dump(value, stream, indent=2)
        stream.write('\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['jdk', 'gradle', 'seed-inventory', 'jdk-freeze', 'retained', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    for name, path in vars(args).items():
        if not path.is_absolute() or path.resolve() != path:
            raise ValueError('absolute canonical input required: ' + name)
    args.out.mkdir(mode=0o700)
    started = time.monotonic()
    source = Path(__file__).resolve().parent
    classes = args.out / 'classes'
    classes.mkdir()
    assert sha(args.seed_inventory) == 'e155725cfa2db6b4a7a94625587c81be66d2a4fd18a952615d1fc98716b47aa8'
    seed = json.loads(args.seed_inventory.read_text())
    prefix = 'gradle/wrapper/dists/gradle-9.7.1-bin/1w1c7tv4s851m17nbqdsro2tv/gradle-9.7.1/'
    expected = {item['path'][len(prefix):]: item for item in seed
                if item['path'].startswith(prefix + 'lib/') and item['path'].endswith('.jar')
                and Path(item['path'][len(prefix):]).parent.as_posix() in ['lib', 'lib/plugins']}
    # lib/api contains public API stubs that throw Error. Match the runtime's
    # real lib/* and lib/plugins/* classpath, never recursively include stubs.
    jars = sorted(args.gradle.glob('lib/*.jar')) + sorted(args.gradle.glob('lib/plugins/*.jar'))
    assert set(expected) == {str(path.relative_to(args.gradle)) for path in jars}
    for path in jars:
        item = expected[str(path.relative_to(args.gradle))]
        assert sha(path) == item['sha256'] and path.stat().st_size == item['size']
    # Reuse the externally frozen, already qualified JDK inventory.
    assert sha(args.jdk_freeze) == 'cf759b0909d52f0e125fc61bac45a421146b8654bbc81cf8543fd0e91fb1b044'
    jdk_inventory = json.loads(args.jdk_freeze.read_text())['jdkInventory']
    for item in jdk_inventory:
        path = args.jdk / item['path']
        assert path.stat().st_mode & 0o777 == item['mode']
        if item['type'] == 'file':
            assert sha(path) == item['sha256'] and path.stat().st_size == item['size']
    cp = ':'.join(map(str, jars))
    commands = []

    def run(name, command, data=None, timeout=180):
        before = time.monotonic()
        with (args.out / (name + '.stdout')).open('x') as out, (args.out / (name + '.stderr')).open('x') as err:
            result = subprocess.run(command, input=data, text=True, stdout=out, stderr=err,
                                    timeout=timeout, env={'LANG': 'C.UTF-8', 'LC_ALL': 'C.UTF-8'})
        commands.append({'command': list(map(str, command)), 'exitCode': result.returncode,
                         'seconds': time.monotonic() - before})
        write(args.out / (name + '-command.json'), commands[-1])
        if result.returncode:
            raise RuntimeError(name + ' failed: ' + (args.out / (name + '.stderr')).read_text()[:2000])

    run('compile', [args.jdk / 'bin/javac', '-Xlint:all,-path', '-Werror', '-cp', cp,
                    '-d', classes, source / 'MetadataProjection.java', source / 'MetadataProjectionTest.java'])
    helper = args.out / 'metadata-projection.jar'
    members = sorted(path.name for path in classes.glob('MetadataProjection*.class')
                     if not path.name.startswith('MetadataProjectionTest'))
    jar_args = [args.jdk / 'bin/jar', '--create', '--file', helper, '--date=2020-01-01T00:00:00Z']
    for member in members:
        jar_args.extend(['-C', classes, member])
    run('package', jar_args)
    real_file = args.retained / 'attempts/C001/outputs/server/build/tmp/compileJava/previous-compilation-data.bin'
    run('synthetic', [args.jdk / 'bin/java', '-Xmx512m', '-cp', str(classes) + ':' + cp,
                      'MetadataProjectionTest', real_file])
    contract = json.loads((source.parent / 'metadata-output-contract-v1.json').read_text())
    paths = [args.retained / 'attempts' / row / 'outputs' / item['path']
             for row in ['C001', 'C003'] for item in contract['compilerSelectors'] if item['type'] == 'file']
    run('retained', [args.jdk / 'bin/java', '-Xmx512m', '-cp', str(helper) + ':' + cp,
                     'MetadataProjection'], ''.join(str(path) + '\n' for path in paths))
    lines = (args.out / 'retained.stdout').read_text().splitlines()
    assert len(lines) == len(paths) == 452
    for path, line in zip(paths, lines):
        assert line.split('\t')[0] == sha(path)
    assert all(a.split('\t')[1] == b.split('\t')[1] for a, b in zip(lines[:226], lines[226:]))
    write(args.out / 'tool.json', {'java': binding(args.jdk / 'bin/java'), 'jdkInventory': jdk_inventory,
                                  'helper': binding(helper), 'gradleJars': [binding(path) for path in jars]})
    write(args.out / 'verified.json', {'schemaVersion': 'buildopt.eic/metadata-java-qualification/v1',
                                     'source': binding(source / 'MetadataProjection.java'),
                                     'tests': binding(source / 'MetadataProjectionTest.java'),
                                     'seedInventory': binding(args.seed_inventory), 'jdkFreeze': binding(args.jdk_freeze),
                                     'tool': binding(args.out / 'tool.json'), 'realFiles': 452,
                                     'semanticPairs': 226, 'commands': commands, 'passed': True,
                                     'seconds': time.monotonic() - started, 'gradleStarts': 0})
    print('PASS: standalone negative cases and 226 real semantic pairs; Gradle starts=0', flush=True)


if __name__ == '__main__':
    main()
