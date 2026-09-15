"""Activate the published inputs; never change the measured identity."""
from common import *
import datetime,shutil,subprocess,time
Q=P.parent/'bo-06-disk-accounting-2026-09-15'
label=sys.argv[1];assert label in ('control','prefix')
a=load(P/'allocation.json');assert a['status']=='allocated'
now=datetime.datetime.now(datetime.timezone.utc)
remaining=(datetime.datetime.fromisoformat(a['deadlineUTC'])-now).total_seconds();assert remaining>=900
freeze=load(Q/'receipts/next-control-admission.json')
assert bind(Q/'receipts/next-control-admission.json')['sha256']==bind(R/'benchmarks/results/buildopt-product-viability-v1/bo-06-disk-accounting/next-control-admission.json')['sha256']
for key in ('comparatorQualification','ownerPolicy','localProof','quietPolicy'):assert bind(freeze[key]['path'])==freeze[key]
local=load(freeze['localProof']['path'])
for b in [local['executable'],*local['package']]:assert bind(b['path'])==b,b
kind='control' if label=='control' else 'development'
m=load(Q/'manifests'/f'{kind}-proposal.json')
original=load(Q/'manifests'/f'{kind}-proposal.json')
m['runRoot']=str(P/'profiles'/label/'run')
used=int(subprocess.check_output(['du','-s','-B1',str(P)],text=True).split()[0])
m['limits'].update(deadlineUTC=a['deadlineUTC'],maxRunNS=int(remaining*1e9),maxBytes=a['maxNewBytes']-used-1024*2**20)
assert m['limits']['maxBytes']>0 and shutil.disk_usage(P).free>=a['minimumFreeBytes']
readiness=load(Q/'receipts'/f'{kind}-readiness.json')
assert readiness['identity']==freeze['measurementIdentity']
if label=='prefix':
    summary=load(P/'analysis/control-summary-v4.json');assert summary['decision']=='CONTROL_PASSED' and summary['passed']
    assert a['ownerReserved']==a['actualOwnerStarts']==a['actualHelpers']==8
    assert len(load(P/'analysis/control-host-pressure.json')['requests'])==8
    readiness['controlProof']=load(P/'receipts/control-trial-proof.json')
    assert readiness['controlProof']['manifest']['path']==str(P/'profiles/control/run/manifest.json')
    # The outer sampler uses a per-stage cap. Deduct the completed control's
    # diagnostics so both stages stay inside the shared 256-MiB limit.
    prior_diagnostics=(P/'profiles/control/process-samples.jsonl').stat().st_size
    a['maxDiagnosticBytes']=load(P/'inputs/allocation-frozen.json')['maxDiagnosticBytes']-prior_diagnostics
    assert a['maxDiagnosticBytes']>0
    a['controlDiagnosticBytes']=prior_diagnostics;atomic(P/'allocation.json',a)
save(P/'receipts'/f'{label}-readiness.json',readiness)
m['measurementReadiness']=bind(P/'receipts'/f'{label}-readiness.json')
assert all(m[k]==original[k] for k in m if k not in ('runRoot','limits','measurementReadiness'))
assert all(m['limits'][k]==original['limits'][k] for k in original['limits'] if k not in ('deadlineUTC','maxRunNS','maxBytes'))
save(P/'profiles'/label/'manifest.json',m)
command=[m['executable']['path'],'validate',str(P/'profiles'/label/'manifest.json')]
start=time.monotonic_ns();result=subprocess.run(command,capture_output=True,text=True,timeout=300)
save(P/'receipts'/f'{label}-validation.json',dict(command=command,exitCode=result.returncode,stdout=result.stdout,stderr=result.stderr,durationNS=time.monotonic_ns()-start,manifest=bind(P/'profiles'/label/'manifest.json')))
assert result.returncode==0,result.stderr
identity=subprocess.check_output([m['executable']['path'],'research-identity',str(P/'profiles'/label/'manifest.json')],text=True,timeout=30).strip()
assert identity==freeze['measurementIdentity']
assert not Path(m['runRoot']).exists()
inputs=[m['executable'],*m['package'],m['outputs']['owner'],freeze['comparatorQualification'],freeze['localProof'],bind(P/'profiles'/label/'manifest.json'),bind(P/'receipts'/f'{label}-readiness.json')]
inputs += [bind(f) for f in sorted([*P.glob('*.py'), *P.glob('tools/*.py')])]
save(P/'inputs'/f'{label}-freeze.json',inputs)
s=load(P/'task-state.json');s['steps']['input recovery and activation']='verified';s['steps']['identical-code control' if label=='control' else 'development prefix']='in progress';s['nextAction']='Execute '+label+' without retries';atomic(P/'task-state.json',s)
print(label,'admitted',identity,flush=True)
