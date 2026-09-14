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

assert load(P/'analysis/control-summary-v4.json')['decision']=='CONTROL_PASSED'
assert load(P/'allocation.json')['actualOwnerStarts']==load(P/'allocation.json')['actualHelpers']==8
assert len(load(P/'analysis/control-host-pressure.json')['requests'])==8
save(P/'receipts/pipeline-v2-registration.json',dict(pid=os.getpid(),startedUTC=now().isoformat(),source=bind(Path(__file__)),scope='Continue after verified control; run exactly one42-build prefix; never confirmation',stageSources=[bind(P/f) for f in ['analyze_trial_v4.py','analyze_host.py','prepare_prefix.py','run_trial.py']],controlSummary=bind(P/'analysis/control-summary-v4.json')))
state=load(P/'task-state.json');state['pipelineHistory']=[state.get('pipeline'),state.get('pipelineStage')];state['pipeline']=dict(pid=os.getpid(),registration=str(P/'receipts/pipeline-v2-registration.json'),status='running development prefix');atomic(P/'task-state.json',state)
run_stage('prefix-preparation','prepare_prefix.py')
run_stage('prefix','run_trial.py','prefix')
run_stage('prefix-analysis','analyze_trial_v4.py','prefix')
run_stage('prefix-host','analyze_host.py','prefix')
state=load(P/'task-state.json');state['pipeline']['status']='finished';state['pipelineStage']=None
state['nextAction']='Review complete trial evidence; close allocation; export and publish result. BO-07 remains separately allocated.'
atomic(P/'task-state.json',state)
print('Approved owner sequence finished',flush=True)
