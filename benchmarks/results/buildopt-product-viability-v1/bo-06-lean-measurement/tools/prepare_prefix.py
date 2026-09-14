from common import *
import datetime,subprocess
control=load(P/'analysis/control.json');assert control['passed'] and control['decision']=='CONTROL_PASSED'
a=load(P/'allocation.json');assert a['actualOwnerStarts']==a['actualHelpers']==8 and a['ownerReserved']==8
m=load(P/'profiles/control/manifest.json');m.pop('controlBaseline')
m.update(subject='elasticsearch-checkstyle-lean-development-0-20',phase='ENGINEERING',history=load(P.parent/'bo-06-readiness-2026-09-14/inputs/seed-history.json'),runRoot=str(P/'profiles/prefix/run'),prefixEnd=20,executionEnd=20)
now=datetime.datetime.now(datetime.timezone.utc);remaining=(datetime.datetime.fromisoformat(a['deadlineUTC'])-now).total_seconds();assert remaining>=900
m['frozenUTC']=now.isoformat();m['limits'].update(maxRunNS=int(remaining*1e9),maxBytes=a['maxNewBytes']-bytes_used(P)-1024*2**20,maxWorkflowStarts=42,maxGradleStarts=42)
assert m['limits']['maxBytes']>0
readiness=load(P/'receipts/control-readiness.json');readiness['controlProof']=load(P/'receipts/control-trial-proof.json')
save(P/'receipts/prefix-readiness.json',readiness);m['measurementReadiness']=bind(P/'receipts/prefix-readiness.json')
save(P/'profiles/prefix/manifest.json',m)
result=subprocess.run([m['executable']['path'],'validate',str(P/'profiles/prefix/manifest.json')],capture_output=True,text=True)
save(P/'receipts/prefix-validation.json',dict(exitCode=result.returncode,stdout=result.stdout,stderr=result.stderr));assert result.returncode==0,result.stderr
freeze=[bind(P/'source'),bind(P/'history-replay-lean'),bind(P/'profiles/prefix/manifest.json'),bind(P/'receipts/prefix-readiness.json')]+[bind(x) for x in sorted(P.glob('*.py'))]
save(P/'inputs/prefix-freeze.json',freeze)
state=load(P/'task-state.json');state['steps']['development prefix']='in progress';state['nextAction']='Run exactly 42 builds on original anchor0 through20; no protected history';atomic(P/'task-state.json',state)
print('Development prefix frozen,42 owner builds',flush=True)
