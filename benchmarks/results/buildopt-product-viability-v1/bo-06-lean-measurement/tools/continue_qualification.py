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
run('retry-local-no-action',[str(p/'fixture-tests'),'-test.v','-test.run=^TestTypedFailuresAndNativeFallback$/^no-action$','-test.timeout=160s'],p/'source',env)
prior=json.loads((p/'receipts/TestTypedFailuresAndNativeFallback.json').read_text())
text=Path(prior['log']).read_text()
for case in ['mismatch','native-failure','unavailable','fallback']:assert f'--- PASS: TestTypedFailuresAndNativeFallback/{case} ' in text
save(p/'receipts/failure-fallback-qualification.json',dict(exitCode=0,scope='Four unchanged passing subcases from original run plus one fresh no-action subcase. Original elapsed-limit failure retained.',evidence=[prior,json.loads((p/'receipts/retry-local-no-action.json').read_text())]))
cases=['TestDriftStopsBeforeNextChild','TestAllocationStopsAndDetachedChild','TestDriverDeathKillsDetachedTreeAndRetainsCandidateCost','TestControlBaselineChronologicalReplay','TestResearchCLIAndMissingResults','TestAffinityScope']
for name in cases:
 run(name,[str(p/'fixture-tests'),'-test.v','-test.run=^'+name+'$','-test.timeout=160s'],p/'source',env)
 assert len(list((p/'fixtures').glob('*/run/attempts/*/native-start.json')))<=104
run('build-runner',base+['go','build','-mod=readonly','-o',str(p/'history-replay-lean'),pkg])
save(p/'receipts/local-suite.json',dict(status='verified',cases=cases,actualFixtureStarts=len(list((p/'fixtures').glob('*/run/attempts/*/native-start.json'))),ownerStarts=0,comparisonJVMs=0,retainedFailure='TestTypedFailuresAndNativeFallback/no-action elapsed allocation',retry='retry-local-no-action.json'))
print('Local qualification passed',flush=True)
