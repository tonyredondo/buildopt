"""Describe observed order/idle-age association; do not assign causal savings."""
from pathlib import Path
import datetime, hashlib, json, statistics

D=Path(__file__).resolve().parent
RUN=D/'profiles/supported/run'
def load(path):return json.loads(path.read_text())
profile=load(D/'analysis/profile-supported.json')
phases=load(D/'analysis/diagnostic-phases.json')
host=load(D/'analysis/host-pressure.json')
phase={row['attempt']:row for row in phases['attempts']}
pressure={row['attempt']:row for row in host['requests']}
rows=[]
for rep in (1,):
    measured=[]
    for arm in ('N','I'):
        name=f'r{rep}-001-g0-{arm}'
        cold=load(RUN/'attempts'/f'r{rep}-000-g0-{arm}'/'native-finish.json')
        native=load(RUN/'attempts'/name/'native-finish.json')
        pair=next(p for p in profile['pairs'] if p['replication']==rep and not p['warmup'])
        task_data=phase[name]['tasks']
        resources=list(phase[name]['resources'].values())
        measured.append(dict(attempt=name,replication=rep,arm=arm,startBootNS=native['start']['ns'],
            endBootNS=native['end']['ns'],nativeSeconds=(native['end']['ns']-native['start']['ns'])/1e9,
            customerSeconds=pair[arm]['customerMS']/1000,
            idleSinceOwnColdNativeCompletionSeconds=(native['start']['ns']-cold['end']['ns'])/1e9,
            hostFlag=pressure[name]['contention'],hostObservation=pressure[name],
            observedProcessReadBytes=sum(p['observedReadBytes'] for p in resources),
            observedProcessMajorFaults=sum(p['observedMajorFaults'] for p in resources),
            daemonObservedReadBytes=sum(p['observedReadBytes'] for p in resources if p['role']=='daemon'),
            daemonObservedMajorFaults=sum(p['observedMajorFaults'] for p in resources if p['role']=='daemon'),
            selectedTaskSeconds={name:task_data[name]['seconds'] for name in (
                ':server:compileJava',':server:compileTestJava',':server:collectTransportVersionReferences',
                ':server:checkstyleMain',':server:checkstyleTest') if name in task_data}))
    measured.sort(key=lambda row:row['startBootNS'])
    for order,row in enumerate(measured,1):
        row['measuredOrder']=order
        row['gapFromPriorMeasuredNativeSeconds']=None if order==1 else (row['startBootNS']-measured[0]['endBootNS'])/1e9
    rows.extend(measured)
first=[row['customerSeconds'] for row in rows if row['measuredOrder']==1]
second=[row['customerSeconds'] for row in rows if row['measuredOrder']==2]
result=dict(status='verified descriptive order and idle-age reconstruction for one A/A pair; causal attribution unverified',
    primaryDecision=profile['decision'],rows=rows,firstMeasuredCustomerSeconds=first,secondMeasuredCustomerSeconds=second,
    firstMeanCustomerSeconds=statistics.mean(first),secondMeanCustomerSeconds=statistics.mean(second),
    secondRequestSlower=rows[1]['customerSeconds']>rows[0]['customerSeconds'],
    sameActualCheckingWork=phases['actualProcessedAndReusedCountsEquivalent'],
    observation='Same-code 6-to-7 control; order and idle intervals are descriptive, not a causal explanation.',
    implication='Apply the predeclared A/A threshold to the whole request. No sample removal, timing retry or value gate is permitted.',
    nextControl='No control rerun authorized by this result. Record the measurement disposition before another timing allocation.',
    limits=['Only one replication; no variance or causal estimate',
            'Cold-to-measured interval contains capture, comparison and other-arm work; it is not pure daemon sleep',
            'Sampled process reads/faults omit unobserved boundary work and do not identify a cache layer',
            'This control cannot establish measurement precision or optimization value'],
    analyzerSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest())
output=D/'analysis/order-and-idle-age.json';assert not output.exists();output.write_text(json.dumps(result,indent=2)+'\n')
for row in rows:print(row['attempt'],'order',row['measuredOrder'],'customer',round(row['customerSeconds'],3),'idle',round(row['idleSinceOwnColdNativeCompletionSeconds'],1),'host',row['hostFlag'],'daemon major faults',row['daemonObservedMajorFaults'])
