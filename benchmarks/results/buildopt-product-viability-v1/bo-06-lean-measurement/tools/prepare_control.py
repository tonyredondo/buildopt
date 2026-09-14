from common import *
import copy,datetime,shutil,subprocess,time
F=P.parent/'bv006-checkstyle-finalization';S=P.parent/'bo-05-checkstyle-observer-replay-2026-09-14'
assert load(P/'receipts/local-suite.json')['status']=='verified'
for b in load(S/'inputs/launch-freeze.json'):assert bind(b['path'])==b,b['path']
save(P/'receipts/prior-bindings.json',dict(verified=True,count=len(load(S/'inputs/launch-freeze.json'))))
m=load(S/'profiles/supported/manifest.json')
m.update(mode='P_LEAN_RESEARCH_V1',subject='elasticsearch-checkstyle-lean-control-17-20',replications=1,runRoot=str(P/'profiles/control/run'))
m['executable']=bind(P/'history-replay-lean');m['package']=[bind(P/'source')]
m['environment']['JAVA_TOOL_OPTIONS']='-Dfile.encoding=UTF-8'
m['runtime']=[b for b in m['runtime'] if '/diagnostics/' not in b['path']]
combined=copy.deepcopy(m['baseline']);combined['identity']='identical-supported-v2';combined['files']+=copy.deepcopy(m['candidate']['files']);combined['prerequisites'].update(m['candidate']['prerequisites']);m['controlBaseline']=combined
local=dict(schema='buildopt.lean-research/local-proof/v1',exitCode=0,executable=m['executable'],package=m['package'],ownerStarts=0,checks={})
for key,name in {'invocation-output':'TestOwnedReplayAndIndependentChecker','cancellation-closure':'TestDriverDeathKillsDetachedTreeAndRetainsCandidateCost','source-drift':'TestDriftStopsBeforeNextChild','missing-result':'cli-route-and-affinity','readiness-admission':'TestResearchReadinessAdmission','affinity':'cli-route-and-affinity'}.items():
 receipt=load(P/'receipts'/f'{name}.json');assert receipt['exitCode']==0
 # Bind complete test log as well as its structured receipt in package freeze.
 assert bind(receipt['log'])['sha256']==receipt['sha256']
 local['checks'][key]=bind(receipt['log'])
local['checks']['failure-fallback']=bind(P/'receipts/failure-fallback-qualification.json')
save(P/'receipts/local-proof.json',local)
now=datetime.datetime.now(datetime.timezone.utc);deadline=now+datetime.timedelta(hours=10)
a=dict(status='allocated',startedUTC=now.isoformat(),startedBootNS=time.clock_gettime_ns(time.CLOCK_BOOTTIME),deadlineUTC=deadline.isoformat(),maxElapsedSeconds=36000,maxNewBytes=80*2**30,minimumFreeBytes=40*2**30,maxDiagnosticBytes=256*2**20,maxOwnerStarts=50,maxHelpers=50,ownerReserved=0,helpersReserved=0,actualOwnerStarts=0,actualHelpers=0,retriesAllowed=0)
assert shutil.disk_usage(P).free>=a['minimumFreeBytes']
save(P/'allocation.json',a);save(P/'inputs/allocation-frozen.json',a)
m['frozenUTC']=now.isoformat();m['limits'].update(deadlineUTC=a['deadlineUTC'],maxRunNS=36000*10**9,maxBytes=80*2**30-bytes_used(P)-1024*2**20,maxWorkflowStarts=8,maxGradleStarts=8)
save(P/'inputs/identity-manifest.json',m)
identity=subprocess.check_output([m['executable']['path'],'research-identity',str(P/'inputs/identity-manifest.json')],text=True).strip()
save(P/'receipts/control-readiness.json',dict(schema='buildopt.lean-research/readiness/v1',mode=m['mode'],identity=identity,localProof=bind(P/'receipts/local-proof.json')))
m['measurementReadiness']=bind(P/'receipts/control-readiness.json')
save(P/'profiles/control/manifest.json',m)
result=subprocess.run([m['executable']['path'],'validate',str(P/'profiles/control/manifest.json')],capture_output=True,text=True)
save(P/'receipts/control-validation.json',dict(exitCode=result.returncode,stdout=result.stdout,stderr=result.stderr));assert result.returncode==0,result.stderr
# Source, scripts and qualification evidence fixed before owner execution.
freeze=[bind(P/'inputs/approved-method.md'),bind(P/'logs/trial-admission.log'),bind(P/'source'),bind(P/'history-replay-lean'),bind(P/'profiles/control/manifest.json'),bind(P/'receipts/local-proof.json')]
freeze += [bind(x) for x in sorted(P.glob('*.py'))]
freeze += [bind(x) for x in sorted((P/'receipts').glob('Test*.json'))]
save(P/'inputs/control-freeze.json',freeze)
state=load(P/'task-state.json');state['steps']['local qualification']='verified';state['steps']['identical-code control']='in progress';state['nextAction']='Run exactly eight identical-code owner builds';atomic(P/'task-state.json',state)
print(json.dumps(dict(status='control frozen',identity=identity,deadlineUTC=a['deadlineUTC'])))
