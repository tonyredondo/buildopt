"""Close the retained incomplete measurement without starting another comparison."""
from pathlib import Path
import sys
sys.dont_write_bytecode=True
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from common import *
import datetime,shutil,subprocess
s=load(P/'task-state.json');assert s['pipeline']['status']=='stopped with retained failure'
a=load(P/'allocation.json');assert a['status']=='allocated'
Q=P.parent/'bo-06-quiet-start-integration-2026-09-15'
identity=load(Q/'receipts/next-control-admission.json')['measurementIdentity']
trials=[];units=[];controllers=[]
for label in ('control','prefix'):
    profile=P/'profiles'/label;run=profile/'run'
    if not run.exists():continue
    summary_path=P/'analysis'/('control-summary-v4.json' if label=='control' else 'prefix-retained-summary.json')
    summary=load(summary_path)
    assert summary['status']==('verified' if label=='control' else 'verified incomplete accounting')
    for b in load(P/'inputs'/f'{label}-freeze.json'):assert bind(b['path'])==b,b
    m=load(profile/'manifest.json')
    observed=subprocess.check_output([m['executable']['path'],'research-identity',str(profile/'manifest.json')],text=True,timeout=30).strip()
    assert observed==identity
    end=load(P/'receipts'/f'{label}-end.json');assert end['exitCode']==(0 if label=='control' else 1) and end['observationFailure'] is None
    pid=end['pid'];proc=Path(f'/proc/{pid}/stat')
    if proc.exists():assert int(proc.read_text().rsplit(')',1)[1].split()[19])!=end['procStartTicks']
    controllers.append(dict(pid=pid,startTicks=end['procStartTicks'],originalProcessAbsent=True))
    for f in sorted((run/'sessions').glob('*/worker-config.json')):
        cfg=load(f);closed=load(f.parent/'closed.json')
        assert not closed['remaining'] and not (Path('/sys/fs/cgroup')/closed['cgroup'].lstrip('/')).exists()
        r=subprocess.run(['systemctl','--user','is-active',cfg['unit']],capture_output=True,text=True,timeout=15)
        assert r.returncode!=0 and r.stdout.strip() in ('inactive','unknown')
        units.append(dict(unit=cfg['unit'],status=r.stdout.strip(),closure=bind(f.parent/'closed.json')))
    trials.append(dict(label=label,summary=bind(summary_path),decision=summary['decision'],requests=summary['requests'],livePairs=summary['livePairs'],independentPairs=summary['independentPairs']))
control=load(P/'analysis/control-summary-v4.json')
assert len(trials)==(2 if control['passed'] else 1)
assert len(units)==2*len(trials)
requests=sum(t['requests'] for t in trials);helpers=sum(t['livePairs']+t['independentPairs'] for t in trials)
assert requests==a['actualOwnerStarts']==25 and helpers==24
assert a['ownerReserved']==a['helpersReserved']==50
assert requests<=a['maxOwnerStarts'] and helpers<=a['maxHelpers']
a['actualHelpers']=helpers
if not control['passed']:assert not (P/'profiles/prefix/run').exists()
diagnostics=sum(f.stat().st_size for f in (P/'profiles').glob('*/process-samples.jsonl'))
assert diagnostics<=load(P/'inputs/allocation-frozen.json')['maxDiagnosticBytes']
allocated=int(subprocess.check_output(['du','-s','-B1',str(P)],text=True).split()[0]);free=shutil.disk_usage(P).free
now=datetime.datetime.now(datetime.timezone.utc)
assert now<datetime.datetime.fromisoformat(a['deadlineUTC']) and allocated<a['maxNewBytes'] and free>=a['minimumFreeBytes']
decision=trials[-1]['decision']
assert decision=='INCOMPLETE_DEVELOPMENT_QUIET_TIMEOUT'
pipeline=load(P/'receipts/pipeline-start.json')
assert not Path('/proc/'+str(pipeline['pid'])).exists()
controllers.append(dict(pid=pipeline['pid'],originalProcessAbsent=True))
a.update(status='closed verified',closedUTC=now.isoformat(),allocatedBytes=allocated,diagnosticBytes=diagnostics,decision=decision)
atomic(P/'allocation.json',a);save(P/'inputs/closed-allocation.json',a)
receipt=dict(status='verified',decision=decision,ownerStarts=requests,comparisonJVMs=helpers,ownerReservations=a['ownerReserved'],helperReservations=a['helpersReserved'],retries=0,protectedStarts=0,allocatedBytes=allocated,freeBytes=free,diagnosticBytes=diagnostics,allocation=bind(P/'inputs/closed-allocation.json'),measurementIdentity=identity,trials=trials,units=units,controllers=controllers,sourcesUnchanged=True,recordedWorkflowOnly=True,ordinaryGradleClaim=False)
save(P/'receipts/allocation-closeout.json',receipt)
s['steps']['development prefix']='partial';s['actualHelpers']=helpers;s['steps']['allocation closure and decision']='verified';s['steps']['publication']='in progress';s['decision']=decision;s['nextAction']='Export and publish the passing control and incomplete development; no retries or protected replay';s['runningProcesses']=[];atomic(P/'task-state.json',s)
print(json.dumps(dict(decision=decision,ownerStarts=requests,comparisonJVMs=helpers,processesClosed=True)))
