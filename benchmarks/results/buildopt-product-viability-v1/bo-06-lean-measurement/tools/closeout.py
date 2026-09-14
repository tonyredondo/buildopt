"""Close the incomplete allocation without changing its frozen inputs."""
from common import *
import datetime, shutil, subprocess

diagnosis_path = P / 'analysis/prefix-admission-diagnosis.json'
now = load(diagnosis_path)['observedUTC'] if diagnosis_path.exists() else datetime.datetime.now(datetime.timezone.utc).isoformat()
run = P / 'profiles/control/run'
m = load(run / 'manifest.json')
prefix = load(P / 'profiles/prefix/manifest.json')
policy = load(m['outputs']['owner']['path'])
old_path = R / 'benchmarks/results/buildopt-product-viability-v1/bv006/owner-readiness-v5/owner-qualification.json'
old = load(old_path)
assert policy['qualification'] == {'path': '', 'sha256': ''}
assert old['comparatorSHA256'] != policy['comparator']['sha256']
assert load(P / 'receipts/prefix-validation.json')['stderr'] == 'lstat : no such file or directory\n'
assert m['outputs'] == prefix['outputs']
assert not (P / 'profiles/prefix/run').exists()
for b in load(P / 'inputs/control-freeze.json'):
    assert bind(b['path']) == b, b['path']
for key in ('executable', 'protocol'):
    assert bind(m[key]['path']) == m[key]
for b in m['package']:
    assert bind(b['path']) == b
assert bind(m['outputs']['owner']['path']) == m['outputs']['owner']
summary = load(P / 'analysis/control-summary-v4.json')
assert summary['decision'] == 'CONTROL_PASSED'
assert summary['requests'] == 8
units = []
for f in sorted((run / 'sessions').glob('*/worker-config.json')):
    cfg = load(f)
    closed = load(f.parent / 'closed.json')
    result = subprocess.run(['systemctl', '--user', 'is-active', cfg['unit']], capture_output=True, text=True)
    assert result.returncode != 0 and result.stdout.strip() in ('inactive', 'unknown')
    assert not closed['remaining']
    assert not (Path('/sys/fs/cgroup') / closed['cgroup'].lstrip('/')).exists()
    units.append(dict(unit=cfg['unit'], exitCode=result.returncode, stdout=result.stdout, stderr=result.stderr, closure=bind(f.parent / 'closed.json')))
assert len(units) == 2
dead = []
for pid in (576931, 576934, 537750, 364448):
    assert not Path(f'/proc/{pid}').exists()
    dead.append(pid)
diagnosis = dict(status='verified', decision='PREFIX_NOT_ADMITTED_MISSING_COMPARATOR_QUALIFICATION',
    observedUTC=now, ownerPolicy=m['outputs']['owner'], missingField='qualification',
    source=bind(P / 'source/owner_policy.go'), identitySource=bind(P / 'source/research_measurement.go'),
    admissionFailure=bind(P / 'receipts/prefix-validation.json'),
    oldQualification=bind(old_path), oldComparatorSHA256=old['comparatorSHA256'],
    currentComparator=policy['comparator'],
    explanation='The owner policy permits missing qualification only in QUALIFICATION. ENGINEERING reaches checkBinding on the empty qualification path. The older proof binds another comparator. Adding proof changes the owner-policy binding included in the frozen measurement identity.',
    responsibility='Preparation omission: local CLI checks did not validate the real owner ENGINEERING manifest before the control.',
    candidateFailure=False, prefixWorkflowStarts=0, protectedWorkflowStarts=0,
    frozenControlChanged=False, qualificationBypassed=False)
if diagnosis_path.exists():
    assert load(diagnosis_path) == diagnosis
else:
    save(diagnosis_path, diagnosis)
a = load(P / 'allocation.json')
assert a['actualOwnerStarts'] == a['actualHelpers'] == a['ownerReserved'] == a['helpersReserved'] == 8
allocated = int(subprocess.check_output(['du', '-s', '-B1', str(P)], text=True).split()[0])
free = shutil.disk_usage(P).free
assert allocated < a['maxNewBytes'] and free >= a['minimumFreeBytes']
closed = dict(status='verified', allocationResult='INCOMPLETE_PREFIX_NOT_ADMITTED', observedUTC=now,
    ownerStarts=8, comparisonJVMs=8, ownerReservations=8, helperReservations=8,
    fixtureStarts=94, retries=0, prefixStarts=0, protectedStarts=0,
    allocatedBytes=allocated, freeBytes=free, allocation=bind(P / 'allocation.json'),
    control=bind(P / 'analysis/control-summary-v4.json'), diagnosis=bind(P / 'analysis/prefix-admission-diagnosis.json'),
    units=units, absentControllerPIDs=dead,
    limitsMet=True, sourcesUnchanged=True, futureOwnerExecutionAuthorizedByThisCloseout=False)
save(P / 'receipts/allocation-closeout.json', closed)
a.update(status='closed incomplete', closedUTC=now, allocatedBytes=allocated)
atomic(P / 'allocation.json', a)
state = load(P / 'task-state.json')
state['steps'].update({'local qualification': 'partial', 'identical-code control': 'verified', 'development prefix': 'blocked', 'confirmation freeze': 'blocked'})
state['qualification']['scope'] = 'Verified local fixtures and control admission; real owner ENGINEERING qualification was omitted.'
state['status'] = 'partial'
state['decision'] = diagnosis['decision']
state['pipeline']['status'] = 'stopped before prefix; prerequisite absent'
state['pipelineStage']['exitCode'] = 1
state['runningProcesses'] = []
state['nextAction'] = 'Publish the verified control and incomplete allocation. Recover comparator qualification, then validate actual owner admission before proposing a new frozen sequence. Do not start BO-07.'
state['closeout'] = str(P / 'receipts/allocation-closeout.json')
atomic(P / 'task-state.json', state)
print(json.dumps({k: v for k, v in closed.items() if k not in ('units', 'control', 'diagnosis', 'allocation')}))
