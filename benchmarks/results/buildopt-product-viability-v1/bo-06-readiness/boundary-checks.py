"""Eight fixed Gradle boundary checks for supported V2; no timing claim."""
from pathlib import Path
import base64
import datetime
import hashlib
import importlib.util
import json
import os
import shutil
import subprocess
import time
import xml.etree.ElementTree as ET

P = Path(__file__).resolve().parent
R = P.parents[3]
F = P.parent / 'bv006-checkstyle-finalization'
S = P.parent / 'bo-05-checkstyle-observer-replay-2026-09-14'

def load(path):
    return json.loads(path.read_text())

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()

def save(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('x') as stream:
        stream.write(json.dumps(value, indent=2) + '\n')

def update(value):
    target = P / 'task-state.json'
    state = load(target)
    state.update(value)
    temporary = target.with_suffix('.tmp')
    temporary.write_text(json.dumps(state, indent=2) + '\n')
    temporary.replace(target)

spec = importlib.util.spec_from_file_location('bindings', P.parent / 'bv006-cpu-isolation/bindings.py')
bindings = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bindings)
prior = load(S / 'inputs/launch-freeze.json')
for item in prior:
    assert bindings.bind(item['path']) == item, item['path']
save(P / 'receipts/prior-bindings.json', dict(checkedUTC=now(), bindings=prior, allMatch=True))

runtime = load(F / 'inputs/runtime.json')
fixture = P / 'fixture'
fixture.mkdir()
# Copy only the tiny fixture's declared sources, never its Gradle state.
for name in ['settings.gradle', 'build.gradle']:
    shutil.copy2(F / 'fixture-supported' / name, fixture / name)
for name in ['src', 'config']:
    shutil.copytree(F / 'fixture-supported/server' / name, fixture / 'server' / name)
for source in (F / 'fixture-supported/server').glob('adapted-*.xml'):
    shutil.copy2(source, fixture / 'server' / source.name)
build = fixture / 'build.gradle'
text = build.read_text()
old = "if (!name.startsWith('Native')) CheckstyleSuccessHistory.install"
assert text.count(old) == 1
text = text.replace(old, "if (!name.startsWith('Native') && !providers.gradleProperty('disableCorrection').isPresent()) CheckstyleSuccessHistory.install")
build.write_text(text)
shutil.copy2(build, P / 'inputs/boundary-fixture.gradle')

names = ['cache-prime', 'cache-restore', 'edit-after-restore', 'reuse',
         'changed-rules', 'restored-rules', 'disabled', 'reenabled']
registration = dict(frozenUTC=now(), scope='Supported V2 Gradle fixture only',
    cases=names, maxGradleStarts=8, actualGradleStarts=0, maxElapsedSeconds=1200,
    perRequestSeconds=120, maxNewBytes=1073741824, minimumFreeBytes=42949672960,
    noRetries=True, protectedValidationStarts=0,
    expected='Complete native/candidate XML equal on each request; FROM-CACHE clears local history; next edit processes all files; warm reuse processes none; changed rules fall back; disabling checks all files and reenabling checks changed files',
    implementation=bindings.bind(F / 'candidate-v2'),
    runtimeClasses=bindings.bind(F / 'classes-v2'),
    rulesJar=bindings.bind(F / 'build-conventions.jar'),
    agent=bindings.bind(P.parent / 'checkstyle-prototype/observer/agent.jar'),
    fixture=bindings.bind(fixture), script=bindings.bind(Path(__file__)))
save(P / 'inputs/boundary-registration.json', registration)
update(dict(boundaryChecks=dict(status='in progress', registration=str(P / 'inputs/boundary-registration.json'), actualGradleStarts=0)))
started = time.monotonic()
state = fixture / '.gradle/buildopt-checkstyle/server/checkstyleMain/success'
source = fixture / 'server/src/Main/p/A.java'
config = fixture / 'server/config/Main/checkstyle.xml'
adapted = fixture / 'server/adapted-Main.xml'
original_config = config.read_text()
original_adapted = adapted.read_text()
rows = []

def report(path):
    root = ET.parse(path).getroot()
    # Both tasks check the same absolute files; no root or diagnostic normalization.
    return ET.tostring(root)

def invoke(index, expected_count, cache=False, disable=False):
    assert time.monotonic() - started < registration['maxElapsedSeconds']
    assert shutil.disk_usage(P).free >= registration['minimumFreeBytes']
    assert sum(f.stat().st_size for f in P.rglob('*') if f.is_file()) < registration['maxNewBytes']
    case = names[index]
    target = P / 'boundary-runs' / case
    target.mkdir(parents=True)
    unit = 'buildopt-bo06-' + case
    command = [str(Path(runtime['gradle']) / 'bin/gradle'), '--offline', '--no-daemon',
        '--no-configuration-cache', '--build-cache', '--console=plain', '--max-workers=2',
        '--warning-mode=fail', '-Pphase=boundary', '-PobservationFile=' + str(target / 'processing.tsv'),
        '-PobserverAgent=' + str(P.parent / 'checkstyle-prototype/observer/agent.jar')]
    if not cache:
        command.append('--rerun-tasks')
    if disable:
        command.append('-PdisableCorrection=true')
    command += [':server:checkstyleNative', ':server:checkstyleMain']
    env = ['JAVA_HOME=' + str(Path(runtime['java']).parent.parent),
        'GRADLE_USER_HOME=' + str(P / 'fixture-gradle'), 'PATH=/usr/bin:/bin',
        'LANG=C.UTF-8', 'LC_ALL=C.UTF-8', 'TZ=UTC']
    launch = ['systemd-run', '--user', '--wait', '--collect', '--unit=' + unit,
        '--property=WorkingDirectory=' + str(fixture), '--property=RuntimeMaxSec=120',
        '--property=KillMode=control-group', '--property=TimeoutStopSec=15',
        '--property=StandardOutput=append:' + str(target / 'build.log'),
        '--property=StandardError=append:' + str(target / 'build.log'), '/usr/bin/env', *env, *command]
    receipt = dict(case=case, command=command, cwd=str(fixture), unit=unit, startedUTC=now())
    save(target / 'start.json', receipt)
    update(dict(runningProcesses=[receipt], boundaryChecks=dict(status='in progress', actualGradleStarts=index + 1)))
    with (target / 'controller.log').open('x') as log:
        result = subprocess.run(launch, stdout=log, stderr=subprocess.STDOUT, timeout=150)
    receipt.update(exitCode=result.returncode, endedUTC=now())
    closed = subprocess.run(['systemctl', '--user', 'show', unit, '--property=ActiveState', '--value'], text=True, capture_output=True)
    receipt['unitState'] = closed.stdout.strip()
    save(target / 'end.json', receipt)
    update(dict(runningProcesses=[]))
    assert result.returncode == 0, (case, target / 'build.log')
    assert receipt['unitState'] in ['inactive', ''], receipt
    log = (target / 'build.log').read_text()
    assert 'deprecated' not in log.lower(), case
    xml = fixture / 'server/build/reports/checkstyle'
    assert report(xml / 'native.xml') == report(xml / 'main.xml'), case
    for name in ['native', 'main']:
        shutil.copy2(xml / (name + '.xml'), target / (name + '.xml'))
    processed = []
    path = target / 'processing.tsv'
    if path.exists():
        processed = [base64.b64decode(line.split('\t')[-1]).decode()
                     for line in path.read_text().splitlines() if line.startswith('F\t')]
    if cache:
        assert ':server:checkstyleMain FROM-CACHE' in log, case
        assert not state.exists(), 'Cache restore must discard correction history'
    assert len(processed) == expected_count, (case, len(processed), expected_count)
    row = dict(case=case, exitCode=0, xmlEqual=True, processedFiles=len(processed),
        successExists=state.exists(), fromCache=cache, outputSHA256=sha(target / 'main.xml'),
        request=bindings.bind(target / 'end.json'), log=bindings.bind(target / 'build.log'))
    if state.exists():
        shutil.copy2(state, target / 'success')
        row['successSHA256'] = sha(state)
    rows.append(row)
    save(target / 'verification.json', row)
    print(json.dumps(row), flush=True)

try:
    invoke(0, 4)
    assert state.exists()
    shutil.rmtree(fixture / 'server/build')  # Only this task's generated outputs.
    invoke(1, 0, cache=True)
    source.write_text(source.read_text() + '\n')
    invoke(2, 4)
    assert state.exists()
    invoke(3, 2)
    config.write_text(original_config + '\n<!-- Changed rule configuration -->\n')
    adapted.write_text(original_adapted + '\n<!-- Changed rule configuration -->\n')
    invoke(4, 4)
    assert not state.exists()
    config.write_text(original_config)
    adapted.write_text(original_adapted)
    invoke(5, 4)
    assert state.exists()
    source.write_text(source.read_text() + '\n')
    invoke(6, 4, disable=True)
    invoke(7, 3)
    assert state.exists()
    save(P / 'analysis/boundary-checks.json', dict(status='verified', cases=rows,
        actualGradleStarts=8, protectedValidationStarts=0, helperJVMStarts=0,
        scope='Gradle 9.7.1, Java21, Configuration Cache off; fixture checks, no walltime result',
        elapsedSeconds=time.monotonic() - started))
    update(dict(boundaryChecks=dict(status='verified', actualGradleStarts=8), runningProcesses=[]))
except BaseException as error:
    save(P / 'analysis/boundary-checks-failure.json', dict(error=repr(error), completed=rows, atUTC=now()))
    update(dict(status='partial'))
    raise
