from common import *
import copy, subprocess, time
allocation()
assert load(P/'analysis/admission-matrix.json')['status']=='verified'
local=load(L/'receipts/local-proof.json')
local['checks']['readiness-admission']=bind(P/'analysis/admission-matrix.json')
save(P/'receipts/local-proof.json',local)
runner=L/'history-replay-lean'
rows=[]
now=datetime.datetime.now(datetime.timezone.utc)
for kind,old in [('control',L/'profiles/control/manifest.json'),('development',L/'profiles/prefix/manifest.json')]:
    m=load(old)
    m['outputs']['owner']=bind(P/'inputs/qualified-owner-policy.json')
    m['subject']='elasticsearch-checkstyle-qualified-'+kind
    m['runRoot']=str(P/'proposed-runs'/kind)
    m['frozenUTC']=now.isoformat()
    m['limits'].update(deadlineUTC=(now+datetime.timedelta(hours=10)).isoformat(),maxRunNS=36_000_000_000_000,maxBytes=80*1024**3)
    m.pop('measurementReadiness',None)
    preliminary=P/'inputs'/f'{kind}-identity-input.json';save(preliminary,m)
    identity=subprocess.check_output([str(runner),'research-identity',str(preliminary)],text=True,timeout=30).strip()
    readiness=dict(schema='buildopt.lean-research/readiness/v1',mode=m['mode'],identity=identity,localProof=bind(P/'receipts/local-proof.json'))
    path=P/'receipts'/f'{kind}-readiness.json';save(path,readiness);m['measurementReadiness']=bind(path)
    path=P/'manifests'/f'{kind}-proposal.json';save(path,m)
    start=time.monotonic_ns();r=subprocess.run([str(runner),'validate',str(path)],capture_output=True,text=True,timeout=300)
    expected='accepted' if kind=='control' else 'lean research missing required control or prefix proof'
    assert (r.returncode==0) if kind=='control' else (r.returncode!=0 and expected in r.stderr)
    assert not Path(m['runRoot']).exists()
    rows.append(dict(kind=kind,manifest=bind(path),identity=identity,exitCode=r.returncode,stdout=r.stdout,stderr=r.stderr,expected=expected,durationNS=time.monotonic_ns()-start,runRootAbsent=True,syntheticTrialData=False))
assert rows[0]['identity']==rows[1]['identity']
oldidentity=load(L/'receipts/control-readiness.json')['identity'];assert oldidentity!=rows[0]['identity']
control=load(P/'manifests/control-proposal.json');development=load(P/'manifests/development-proposal.json')
fields=['mode','protocol','executable','package','commonGit','command','environment','runtime','baseline','candidate','outputs','acquisition','generatedPaths','affinity','daemonPolicy']
assert all(control[k]==development[k] for k in fields)
assert len(development['history'])==101 and development['executionEnd']==20 and development['prefixEnd']==20
assert [r['commit'] for r in control['history']]==[r['commit'] for r in development['history'][17:21]]
# Bind all measurements' inputs now. Deadline/run allocation is activated in a
# new block; updating those bookkeeping fields cannot change researchIdentity.
frozen=dict(status='verified',scope='Inputs fixed; control admission verified; real development proof remains missing.',measurementIdentity=rows[0]['identity'],previousIdentity=oldidentity,qualification=bind(P/'inputs/owner-qualification.json'),ownerPolicy=bind(P/'inputs/qualified-owner-policy.json'),localProof=bind(P/'receipts/local-proof.json'),sharedFields=fields,proposals=rows,ownerBuilds=0,oldControlCanQualifyNewPolicy=False,protectedHistoryStarts=0,confirmationPrerequisites=['Passing genuine control with this identity.','Passing complete development sequence 0 through 20 with this identity.','Two fresh full confirmation replications after BO-06 closes.','Explicit evidenced preparation/adoption cost for each replication, including measured zero.','Existing correctness and subject bindings; complete scheduled results and independent checks.'])
save(P/'analysis/measurement-freeze.json',frozen)
old=load(L/'inputs/closed-allocation.json')
elapsed=(datetime.datetime.fromisoformat(old['closedUTC'])-datetime.datetime.fromisoformat(old['startedUTC'])).total_seconds()
proposal=dict(status='proposed-not-started',ownerBuilds=50,controlBuilds=8,developmentBuilds=42,comparatorJVMs=50,livePairs=25,independentPairs=25,maxElapsedSeconds=36000,maxRequestSeconds=900,maxNewBytes=80*1024**3,minimumFreeBytes=40*1024**3,maxDiagnosticBytes=256*1024**2,retries=0,replications=1,protectedBuilds=0,measurementIdentity=rows[0]['identity'],previousControlAndCloseoutSeconds=elapsed,previousAllocatedBytes=old['allocatedBytes'],previousAllocation=bind(L/'inputs/closed-allocation.json'),basis='The previous eight-build control and closeout took about 74 minutes and retained 13.2 GiB. Ten hours and 80 GiB retain the earlier bounded proposal; they are stop limits, not a promise that all builds finish.',activation='Create a separate live allocation at launch. Set its deadline once and subtract all preparation/control time before development. Revalidate current input hashes and control proposal; attach only genuine passing control proof to development. Do not use admission-only synthetic records.',controlRule='Stop when absolute aggregate difference is at least 3 seconds AND 5% of the faster side across measured original commits 18-20.',developmentRule='Anchor 0 and all 20 changes, native versus fixed correction. Require at least 5% net saving and 1 second per scheduled non-cold change after required outside-request work. Preserve all outcomes; no retries or replacement commits.',stop='Close this measurement after the development decision. BO-07 protected changes remain a later block.')
save(P/'inputs/next-measurement-proposal.json',proposal)
print(json.dumps(dict(identity=rows[0]['identity'],control='admitted',development='awaits genuine fresh control',ownerBuilds=0)))
