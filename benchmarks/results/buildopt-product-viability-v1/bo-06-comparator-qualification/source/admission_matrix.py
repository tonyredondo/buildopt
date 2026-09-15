"""Admission only: the measured runner is invoked with validate, never run.
Synthetic trial records are isolated test data and cannot qualify live replay.
"""
from common import *
import copy, subprocess, time
allocation()
runner=L/'history-replay-lean'
root=P/'scratch/admission-only'
root.mkdir()
marker=root/'SYNTHETIC-TEST-DATA.txt'
marker.write_text('Synthetic admission records. No builds ran. Not timing or live trial proof.\n')
rows=[]
identity_calls=0

def identity(m):
    global identity_calls
    path=root/f'identity-{identity_calls}.json'; identity_calls+=1
    save(path,m)
    return subprocess.check_output([str(runner),'research-identity',str(path)],text=True,timeout=30).strip()

def ready(m,name,control=None,prefix=None,local=None):
    r=dict(schema='buildopt.lean-research/readiness/v1',mode=m['mode'],identity=identity(m),localProof=local or bind(L/'receipts/local-proof.json'))
    if control is not None:r['controlProof']=control
    if prefix is not None:r['prefixProof']=prefix
    path=root/f'{name}-readiness.json';save(path,r)
    m['measurementReadiness']=bind(path)
    return m

def validate(name,m,expected=''):
    allocation()
    assert not Path(m['runRoot']).exists(), 'validation destination must remain absent'
    path=root/f'{name}-manifest.json';save(path,m)
    start=time.monotonic_ns()
    r=subprocess.run([str(runner),'validate',str(path)],capture_output=True,text=True,timeout=300)
    row=dict(name=name,manifest=bind(path),expected=expected or 'accepted',exitCode=r.returncode,stdout=r.stdout,stderr=r.stderr,durationNS=time.monotonic_ns()-start,runRootAbsent=not Path(m['runRoot']).exists(),syntheticTrialData=True)
    save(P/'receipts'/f'admission-{name}.json',row);rows.append(row)
    assert row['runRootAbsent']
    if expected: assert r.returncode!=0 and expected in r.stderr, row
    else: assert r.returncode==0,row
    print(name, 'PASS',flush=True)

def trial(m,name,control=True,kind='pass'):
    d=root/f'synthetic-{name}';d.mkdir();(d/'costs').mkdir()
    prior=copy.deepcopy(m);prior.pop('measurementReadiness',None)
    prior['runRoot']=str(d);prior['replications']=1
    count=4 if control else 21
    prior['executionEnd']=count-1
    if control:
        prior['phase']='QUALIFICATION';prior['prefixEnd']=0
        prior['history']=copy.deepcopy(m['history'][17:21])
        for i,row in enumerate(prior['history']):row['ordinal']=i
        prior['controlBaseline']=copy.deepcopy(control_template['controlBaseline'])
    else:
        prior['phase']='ENGINEERING';prior['prefixEnd']=20;prior.pop('controlBaseline',None)
    if kind=='wrong-history':prior['history'][1]['commit']='0'*40
    save(d/'manifest.json',prior)
    slots=[dict(ordinal=i,class_='COMPARABLE') for i in range(count)]
    for i,s in enumerate(slots):
        s['class']=s.pop('class_');s.update(reason='',nativeNS=100_000_000_000,candidateNS=100_000_000_000 if control else 80_000_000_000,extraCandidateNS=0,nativeActions=1,candidateActions=1,attempts=[],ownerMetadataJVMStarts=0)
        if kind=='negative':s['candidateNS']=120_000_000_000
    if kind=='incomplete':slots.pop()
    result=dict(schema='buildopt.history-replay/result/v2',manifestSHA256=bind(d/'manifest.json')['sha256'],replications=[],slots=[slots],workflowStarts=2*count,gradleReservations=2*count,actualGradleStarts=2*count,nestedStarts=0,unknownGradleReservations=0,decision='SYNTHETIC_TEST_DATA',reasons=[])
    # Use the actual result schema, without borrowing actual trial measurements.
    result['schema']=load(L/'profiles/control/run/result.json')['schema']
    save(d/'result.json',result)
    closure=dict(schema='buildopt.lean-research/closure/v1',manifestSHA256=result['manifestSHA256'],exitCode=0,livePairs=count,independentPairs=count,closed=kind!='open',evidence=[bind(marker)])
    save(d/'closure.json',closure)
    return dict(manifest=bind(d/'manifest.json'),result=bind(d/'result.json'),closure=bind(d/'closure.json'),costs=bind(d/'costs'),outsideRequestNS=0)

control_template=load(L/'profiles/control/manifest.json')
engineering=load(L/'profiles/prefix/manifest.json')
for m in (control_template,engineering):
    m['outputs']['owner']=bind(P/'inputs/qualified-owner-policy.json')
    m['runRoot']=str(P/'scratch/NEVER-EXECUTE-admission-target')
    m.pop('measurementReadiness',None)
control=ready(copy.deepcopy(control_template),'valid-control')
validate('valid-control',control)
good_control=trial(engineering,'control')
good_prefix=trial(engineering,'prefix',False)
engineering=ready(engineering,'valid-engineering',good_control)
validate('valid-engineering',engineering)
confirmation=copy.deepcopy(engineering);confirmation.update(phase='CONFIRMATION',executionEnd=100,replications=2)
confirmation['subjects']=load(P.parent/'bo-06-readiness-2026-09-14/inputs/confirmation-not-admitted.json')['subjects']
confirmation['limits']['maxWorkflowStarts']=404;confirmation['limits']['maxGradleStarts']=404
confirmation['externalCosts']=[]
for rep in (1,2):
    cost=dict(schema='buildopt.history-replay/external-cost/v1',id=f'synthetic-adoption-{rep}',class_='customer-machine',purpose='adoption',replication=rep,ordinal=0,start=dict(ns=1,utc='2026-09-15T00:00:00Z',boot='synthetic-test'),end=dict(ns=1,utc='2026-09-15T00:00:00Z',boot='synthetic-test'),evidence=[bind(marker)],explanation='Synthetic admission test only; not measured zero preparation.')
    cost['class']=cost.pop('class_');path=root/f'synthetic-adoption-{rep}.json';save(path,cost);confirmation['externalCosts'].append(bind(path))
confirmation=ready(confirmation,'valid-confirmation',good_control,good_prefix)
validate('valid-confirmation',confirmation)

m=copy.deepcopy(control);m.pop('measurementReadiness');validate('missing-local-proof',m,'own readiness')
m=copy.deepcopy(engineering);ready(m,'missing-control');validate('missing-control',m,'missing required control or prefix proof')
m=copy.deepcopy(engineering);ready(m,'old-real-control',load(L/'receipts/prefix-readiness.json')['controlProof']);validate('old-real-control',m,'scope, sources or completeness differs')
for kind,expected in [('negative','material difference'),('incomplete','scope, sources or completeness differs'),('open','process closure incomplete'),('wrong-history','another history window')]:
    m=copy.deepcopy(engineering);ready(m,f'control-{kind}',trial(m,kind,True,kind));validate(f'control-{kind}',m,expected)
m=copy.deepcopy(confirmation);ready(m,'missing-prefix',good_control);validate('missing-prefix',m,'missing required control or prefix proof')
for kind,expected in [('negative','did not save enough time'),('incomplete','scope, sources or completeness differs')]:
    m=copy.deepcopy(confirmation);ready(m,f'prefix-{kind}',good_control,trial(m,'prefix-'+kind,False,kind));validate(f'prefix-{kind}',m,expected)
m=copy.deepcopy(engineering);m['candidate']['identity']='changed-candidate';validate('changed-candidate',m,'does not bind this implementation')
m=copy.deepcopy(engineering);m['package'][0]['sha256']='0'*64;validate('changed-source',m,'does not bind this implementation')
m=copy.deepcopy(control);m['environment']['JAVA_TOOL_OPTIONS']='-javaagent:diagnostic.jar';validate('java-agent',m,'forbids diagnostic Java agents')
m=copy.deepcopy(confirmation);m['externalCosts']=[];validate('missing-adoption',m,'explicit evidenced external adoption cost')
# Full owner-policy failures need otherwise valid matching trial identities.
for name,kind,expected in [('missing-qualification','missing','lstat'),('old-qualification','old','qualification is incomplete'),('wrong-reader','reader','qualification is incomplete')]:
    policy=load(P/'inputs/qualified-owner-policy.json')
    if kind=='missing':policy['qualification']={'path':'','sha256':''}
    if kind=='old':policy['qualification']=bind(R/'benchmarks/results/buildopt-product-viability-v1/bv006/owner-readiness-v5/owner-qualification.json')
    if kind=='reader':
        changed=root/'wrong-reader.bin';changed.write_bytes(b'wrong compiler state reader')
        policy['metadataClasspath'][0]=bind(changed)
    pp=root/f'{name}-policy.json';save(pp,policy)
    m=copy.deepcopy(engineering);m['outputs']['owner']=bind(pp)
    ready(m,name,trial(m,'trial-'+name));validate(name,m,expected)
assert len(rows)==20,len(rows)
save(P/'analysis/admission-matrix.json',dict(status='verified',cases=rows,runner=bind(runner),ownerPolicy=bind(P/'inputs/qualified-owner-policy.json'),ownerBuilds=0,fixtureWorkflows=0,comparatorJVMs=0,scope='Full actual-runner validate paths with real owner inputs. Trial receipts and adoption costs are synthetic test data only; no live development or confirmation admission is claimed.'))
