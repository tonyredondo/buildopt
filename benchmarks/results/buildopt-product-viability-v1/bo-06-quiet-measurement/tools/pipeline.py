"""One approved control and, only if it passes, one complete development replay."""
from common import *
import datetime,os,subprocess,time,traceback

def now():return datetime.datetime.now(datetime.timezone.utc)
def stage(name,script,*args):
    a=load(P/'allocation.json');assert a['status']=='allocated' and now()<datetime.datetime.fromisoformat(a['deadlineUTC'])
    assert not load(P/'task-state.json').get('stopRequested',False)
    log=P/'logs'/f'pipeline-{name}.log';start=now()
    with log.open('x') as out:
        child=subprocess.Popen([sys.executable,'-B',str(P/script),*args],cwd=R,stdout=out,stderr=subprocess.STDOUT)
        s=load(P/'task-state.json');s['pipelineStage']=dict(name=name,pid=child.pid,startedUTC=start.isoformat());atomic(P/'task-state.json',s)
        code=child.wait()
    save(P/'receipts'/f'pipeline-{name}.json',dict(script=script,args=list(args),exitCode=code,startedUTC=start.isoformat(),endedUTC=now().isoformat(),log=bind(log)))
    print(name,'exit',code,flush=True)
    if code:raise RuntimeError(name+' failed; retained '+str(log))

save(P/'receipts/pipeline-start.json',dict(pid=os.getpid(),startedUTC=now().isoformat(),source=bind(Path(__file__)),scope='8 control builds and, after a passing control, all 42 development builds. No retry or confirmation.'))
s=load(P/'task-state.json');s['pipeline']=dict(pid=os.getpid(),status='running',registration=str(P/'receipts/pipeline-start.json'));atomic(P/'task-state.json',s)
try:
    for label in ('control','prefix'):
        stage(label+'-preparation','prepare_trial.py',label)
        stage(label+'-run','run_trial.py',label)
        stage(label+'-analysis','analyze_trial_v4.py',label)
        stage(label+'-host','analyze_host.py',label)
        summary=load(P/'analysis'/f'{label}-summary-v4.json')
        print(label,summary['decision'],summary['netSavingPercent'],summary['netSecondsPerScheduled'],flush=True)
        if label=='control' and not summary['passed']:
            print('Control stopped the allocation; development will not start',flush=True)
            break
    s=load(P/'task-state.json');s['pipeline']['status']='finished';s['pipelineStage']=None;s['nextAction']='Close allocation and verify all counts, inputs and process closure; publish the complete decision.';atomic(P/'task-state.json',s)
except BaseException as error:
    save(P/'receipts/pipeline-failure.json',dict(type=type(error).__name__,message=str(error),traceback=traceback.format_exc(),atUTC=now().isoformat()))
    s=load(P/'task-state.json');s['pipeline']['status']='stopped with retained failure';s['nextAction']='Observe retained failure and close the allocation; do not repeat any owner build.';atomic(P/'task-state.json',s)
    raise
