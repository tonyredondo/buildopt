"""Same-session continuation of the already running approved BO-06 control."""
from common import *
import datetime,os,subprocess,time

def now():return datetime.datetime.now(datetime.timezone.utc)
def run_stage(name,script,*args):
 assert not load(P/'task-state.json').get('stopRequested',False)
 a=load(P/'allocation.json');assert now()<datetime.datetime.fromisoformat(a['deadlineUTC'])
 log=P/'logs'/f'pipeline-{name}.log';start=now()
 with log.open('x') as out:
  child=subprocess.Popen([sys.executable,'-B',str(P/script),*args],cwd=R,stdout=out,stderr=subprocess.STDOUT)
  state=load(P/'task-state.json');state['pipelineStage']=dict(name=name,pid=child.pid,startedUTC=start.isoformat());atomic(P/'task-state.json',state)
  code=child.wait()
 save(P/'receipts'/f'pipeline-{name}.json',dict(script=script,args=list(args),exitCode=code,startedUTC=start.isoformat(),endedUTC=now().isoformat(),log=bind(log)))
 assert code==0,(name,log.read_text()[-3000:])

assert (P/'receipts/control-start.json').exists() and not (P/'analysis/control.json').exists()
save(P/'receipts/pipeline-registration.json',dict(pid=os.getpid(),startedUTC=now().isoformat(),source=bind(Path(__file__)),scope='Wait for current control; check it once; conditionally run exactly one 42-build prefix; never confirmation',stageSources=[bind(P/f) for f in ['analyze_trial_v2.py','analyze_host.py','prepare_prefix.py','run_trial.py']]))
state=load(P/'task-state.json');state['pipeline']=dict(pid=os.getpid(),registration=str(P/'receipts/pipeline-registration.json'),status='waiting for control');atomic(P/'task-state.json',state)
while not (P/'receipts/control-end.json').exists():
 assert now()<datetime.datetime.fromisoformat(load(P/'allocation.json')['deadlineUTC'])
 assert not load(P/'task-state.json').get('stopRequested',False)
 time.sleep(5)
end=load(P/'receipts/control-end.json');assert end['exitCode']==0 and end['observationFailure'] is None
for b in load(P/'receipts/pipeline-registration.json')['stageSources']:assert bind(b['path'])==b
run_stage('control-analysis','analyze_trial_v2.py','control')
run_stage('control-host','analyze_host.py','control')
if load(P/'analysis/control.json')['passed']:
 run_stage('prefix-preparation','prepare_prefix.py')
 run_stage('prefix','run_trial.py','prefix')
 run_stage('prefix-analysis','analyze_trial_v2.py','prefix')
 run_stage('prefix-host','analyze_host.py','prefix')
state=load(P/'task-state.json');state['pipeline']['status']='finished';state['pipelineStage']=None
state['nextAction']='Review complete trial evidence; close allocation; export and publish result. BO-07 remains separately allocated.'
atomic(P/'task-state.json',state)
print('Approved owner sequence finished',flush=True)
