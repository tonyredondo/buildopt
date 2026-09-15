from common import *
a=allocation()
for b in load(P/'inputs/entry-bindings.json'): assert bind(b['path'])==b,b
assert load(P/'analysis/retained-comparisons.json')['status']=='verified'
assert load(P/'analysis/admission-matrix.json')['status']=='verified'
assert load(P/'analysis/measurement-freeze.json')['status']=='verified'
assert a['actualOwnerBuilds']==0 and a['actualComparatorJVMs']==a['jvmReservations']==8 and a['goCommands']==1
allocated=sum(f.stat().st_blocks*512 for f in P.rglob('*') if f.is_file())
assert allocated<a['maxBytes']
a.update(status='closed verified',closedUTC=datetime.datetime.now(datetime.timezone.utc).isoformat(),allocatedBytes=allocated)
atomic(P/'allocation.json',a);save(P/'inputs/closed-allocation.json',a)
save(P/'receipts/closeout.json',dict(status='verified',scope='Current comparator qualification and admission prerequisites only; BO-06 remains partial.',allocation=bind(P/'inputs/closed-allocation.json'),ownerBuilds=0,fixtureWorkflows=0,comparatorJVMs=8,comparatorCalls=a['comparatorReservations'],goCommands=1,retries=0,protectedBuilds=0,sourcesUnchanged=True,subprocesses='All eight synchronous metadata JVM calls returned. The Go test and every validate process returned. No workflow or session was launched.',comparator=bind(P/'inputs/owner-qualification.json'),admission=bind(P/'analysis/admission-matrix.json'),freeze=bind(P/'analysis/measurement-freeze.json')))
s=load(P/'task-state.json')
for k in s['steps']:
    if k!='publication':s['steps'][k]='verified'
s['steps']['publication']='in progress';s['sessions']={};s['nextAction']='Publish qualification, admission proof and next measurement proposal. No owner execution in this block.'
atomic(P/'task-state.json',s)
print(json.dumps(dict(status='qualification verified',ownerBuilds=0,comparatorJVMs=8,admissionCases=20,allocatedBytes=allocated)))
