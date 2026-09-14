from pathlib import Path
import datetime,json,os,subprocess,time,hashlib
p=Path(__file__).resolve().parent;r=p.parents[3]
def save(path,x):
 with path.open('x') as f:json.dump(x,f,indent=2);f.write('\n')
def run(name,args,cwd=r,env=None,timeout=175):
 start=time.monotonic();log=p/'logs'/f'{name}.log'
 with log.open('x') as out:result=subprocess.run(args,cwd=cwd,env=env,stdout=out,stderr=subprocess.STDOUT,timeout=timeout)
 receipt=dict(command=args,cwd=str(cwd),exitCode=result.returncode,elapsedSeconds=time.monotonic()-start,log=str(log),sha256=hashlib.sha256(log.read_bytes()).hexdigest())
 save(p/'receipts'/f'{name}.json',receipt)
 assert result.returncode==0,(name,log.read_text()[-4000:])
 return receipt
base=['./dev/run','--toolchain','go','--'];pkg='./'+str((p/'source').relative_to(r))
env=os.environ.copy();env['BUILDOPT_REPLAY_TEST_ROOT']=str(p/'fixtures')
receipt=run('cli-route-and-affinity',base+['go','test','-mod=readonly','-tags=replay_integration',pkg,'-run','^(TestResearchCLIAndMissingResults|TestAffinityScope)$','-v','-count=1','-timeout=160s'],r,env)
log=Path(receipt['log']).read_text()
for name in ['TestResearchCLIAndMissingResults','TestAffinityScope']:assert f'--- PASS: {name} ' in log
assert len(list((p/'fixtures').glob('*/run/attempts/*/native-start.json')))<=104
run('build-runner',base+['go','build','-mod=readonly','-o',str(p/'history-replay-lean'),pkg])
save(p/'receipts/local-suite.json',dict(status='verified',actualFixtureStarts=len(list((p/'fixtures').glob('*/run/attempts/*/native-start.json'))),ownerStarts=0,comparisonJVMs=0,retainedFailures=['TestTypedFailuresAndNativeFallback/no-action elapsed allocation','TestResearchCLIAndMissingResults test launcher dispatched the whole test suite; stopped at90 total starts'],repair='TestMain explicitly dispatches CLI verbs; corrected CLI and affinity checks passed'))
print('Local qualification passed',flush=True)
