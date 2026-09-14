"""Check all scheduled pairs without starting another output-comparison JVM."""
from common import *
# A previous report may exist after an accounting-only interruption. Verify it
# byte-for-byte in meaning; never replace its measurements or proof.
original_save=save
def save(path,value):
 if Path(path).exists():
  assert load(path)==value, (path,"existing evidence differs")
 else:original_save(path,value)
import datetime,subprocess,shutil
label=sys.argv[1];assert label in ('control','prefix')
profile=P/'profiles'/label;run=profile/'run';m=load(profile/'manifest.json');count=m['executionEnd']+1;offset=17 if label=='control' else 0
end=load(P/'receipts'/f'{label}-end.json');assert end['exitCode']==0 and end['observationFailure'] is None
result=load(run/'result.json');assert result['manifestSHA256']==bind(run/'manifest.json')['sha256']
assert result['workflowStarts']==result['actualGradleStarts']==result['gradleReservations']==2*count
assert result['nestedStarts']==result['unknownGradleReservations']==0
assert len(result['slots'])==1 and len(result['slots'][0])==count
assert len(list((run/'pairs').glob('*.json')))==count
assert len(list((run/'attempts').glob('*/native-start.json')))==2*count
assert len(list((run/'results').glob('*.json')))==1,'extra independent comparator pass'
costs=[load(f) for f in sorted((run/'costs').glob('*.json')) if 'durationNS' in load(f)]
assert len({x['id'] for x in costs})==len(costs)
for x in costs:assert x['durationNS']==x['endNS']-x['startNS']>=0
outside=sum(x['durationNS'] for x in costs if x['class']=='customer-machine' and not x['insideEnvelope'])
rows=[];proofs=[bind(run/'result.json')];native_masks=set();process_gaps=[];observations=[]
for line in (profile/'process-samples.jsonl').open():observations.append(json.loads(line))
workers={}
for cfgpath in (run/'sessions').glob('*/worker-config.json'):
 cfg=load(cfgpath);closed=load(cfgpath.parent/'closed.json')
 assert not closed['remaining'] and not (Path('/sys/fs/cgroup')/closed['cgroup'].lstrip('/')).exists()
 assert cfg['nativeAffinity']=='0-7' and cfg['observerAffinity']=='8'
 workers[cfg['unit']]=cfg;proofs += [bind(cfgpath),bind(cfgpath.parent/'closed.json')]
assert len(workers)==2
worker_ids=set()
for nativepath in (run/'attempts').glob('*/native-finish.json'):
 supervisor=load(nativepath)['supervisor'];worker_ids.add((supervisor['pid'],supervisor['startTicks']))
assert len(worker_ids)==2
for ordinal in range(count):
 pair=load(run/'pairs'/f'r1-{ordinal:03d}.json');slot=pair['slot'];assert slot==result['slots'][0][ordinal]
 assert slot['class'] in ('COMPARABLE','NATIVE_RETAINED') and slot['extraCandidateNS']==0
 assert slot['ownerMetadataJVMStarts']==1
 row=dict(originalOrdinal=ordinal+offset,commit=m['history'][ordinal]['commit'],cold=ordinal==0,class_=slot['class'],nativeNS=slot['nativeNS'],candidateNS=slot['candidateNS'],savingNS=slot['nativeNS']-slot['candidateNS'],attempts=[])
 order=[]
 for binding in pair['attempts']:
  assert bind(binding['path'])==binding
  folder=Path(binding['path']).parent;start=load(folder/'start.json');finish=load(binding['path']);native=load(finish['native']['path']);capture=load(finish['capture']['path']);after=load(finish['after']['path'])
  assert native['exitCode']==0 and native['outcome']=='EXITED'
  for b in [finish['native'],finish['capture'],finish['after']]:assert bind(b['path'])==b
  assert finish['durationNS']==finish['end']['ns']-finish['begin']['ns']>0
  assert finish['begin']['ns']<=native['start']['ns']<=native['end']['ns']<=finish['end']['ns']
  assert bind(folder/'stdout.log')['sha256']==native['stdoutSHA256'] and bind(folder/'stderr.log')['sha256']==native['stderrSHA256']
  assert len(capture['gradleBuilds'])==1
  order.append((native['start']['ns'],start['arm']))
  entries={e['path']:e for e in after['entries']}
  retained=load(profile/'candidate-state'/start['id']/'receipt.json')
  assert retained['native']==finish['native']
  for e in retained['entries']:assert bind(e['copy'])['sha256']==e['sha256']==entries[e['path']]['sha256']
  sample_times=[];missing=[]
  for sample in observations:
   if not native['start']['ns']<=sample['beginBootNS']<=native['end']['ns']:continue
   sample_times.append(sample['beginBootNS'])
   for group in sample['groups']:
    for process in group['processes']:
     if (process['identity']['pid'],process['identity']['startTicks']) in worker_ids:continue
     for mask in process['threadMasks']:native_masks.add(mask);assert mask=='0,1,2,3,4,5,6,7'
     missing+=process['unavailableObservations']
  times=[native['start']['ns']]+sample_times+[native['end']['ns']]
  row['attempts'].append(dict(arm=start['arm'],id=start['id'],requestNS=finish['durationNS'],nativeNS=native['end']['ns']-native['start']['ns'],candidateApplied=finish['candidateApplied'],successStateFiles=len(retained['entries']),samples=len(sample_times),maximumSampleGapNS=max(b-a for a,b in zip(times,times[1:])),missingDiagnosticObservations=missing))
  proofs.append(bind(folder/'end.json'))
 assert [arm for _,arm in sorted(order)]==(['N','I'] if ordinal%2==0 else ['I','N'])
 rows.append(row);proofs.append(bind(run/'pairs'/f'r1-{ordinal:03d}.json'))
assert native_masks,'No observed native affinity; required invocation proof remains separate'
measured=rows[1:];n=sum(x['nativeNS'] for x in measured);i=sum(x['candidateNS'] for x in measured)
if label=='control':
 diff=abs(n-i);passed=not (diff>=3e9 and diff>=.05*min(n,i));decision='CONTROL_PASSED' if passed else 'CONTROL_MATERIAL_DIFFERENCE'
else:
 diff=n-i-outside;passed=diff>=20e9 and diff>=.05*n;decision='PREFIX_PASSED' if passed else 'PREFIX_INSUFFICIENT_SAVING'
summary=dict(status='verified',decision=decision,passed=passed,recordedWorkflowOnly=True,ordinaryGradleClaim=False,requests=2*count,livePairs=count,independentPairs=count,nativeSeconds=n/1e9,candidateSeconds=i/1e9,outsideRequestSeconds=outside/1e9,netSavedSeconds=(n-i-outside)/1e9,netSavingPercent=100*(n-i-outside)/n,netSecondsPerScheduled=(n-i-outside)/len(measured),controlAbsoluteDifferenceSeconds=abs(n-i)/1e9,controlDifferencePercentOfFaster=100*abs(n-i)/min(n,i),includingColdNativeSeconds=sum(x['nativeNS'] for x in rows)/1e9,includingColdCandidateSeconds=sum(x['candidateNS'] for x in rows)/1e9,rows=rows,costs=costs,diagnosticSamples=len(observations),nativeMasksObserved=sorted(native_masks),continuousIsolationProven=False)
save(P/'analysis'/f'{label}.json',summary)
save(P/'receipts'/f'{label}-closure.json',dict(schema='buildopt.lean-research/closure/v1',manifestSHA256=bind(run/'manifest.json')['sha256'],exitCode=0,livePairs=count,independentPairs=count,closed=True,evidence=proofs))
trial=dict(manifest=bind(run/'manifest.json'),result=bind(run/'result.json'),closure=bind(P/'receipts'/f'{label}-closure.json'),costs=bind(run/'costs'),outsideRequestNS=outside)
save(P/'receipts'/f'{label}-trial-proof.json',trial)
a=load(P/'allocation.json');a['actualHelpers']=sum(2*load(x)['livePairs'] for x in [P/'receipts'/f'{stage}-closure.json' for stage in ('control','prefix') if (P/'receipts'/f'{stage}-closure.json').exists()]);a['actualOwnerStarts']=sum(len(list(x.glob('run/attempts/*/native-start.json'))) for x in (P/'profiles').iterdir());assert a['actualHelpers']<=50 and a['actualOwnerStarts']<=50
allocated=int(subprocess.check_output(['du','-s','-B1',str(P)],text=True).split()[0]);assert allocated<=a['maxNewBytes'] and shutil.disk_usage(P).free>=a['minimumFreeBytes'];a['allocatedBytes']=allocated;atomic(P/'allocation.json',a)
state=load(P/'task-state.json');state['steps']['identical-code control' if label=='control' else 'development prefix']='verified';state['decision']=decision;state.update(actualOwnerStarts=a['actualOwnerStarts'],actualHelpers=a['actualHelpers']);atomic(P/'task-state.json',state)
print(json.dumps({k:v for k,v in summary.items() if k not in ('rows','costs')}),flush=True)
