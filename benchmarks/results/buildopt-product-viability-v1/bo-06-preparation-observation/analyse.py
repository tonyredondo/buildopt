"""Analyse a complete or interrupted preparation observation; never start work."""
from pathlib import Path
from collections import Counter
import gzip
import hashlib
import json
import sys

ROOT = Path(__file__).resolve().parent

def read(name):
    return json.loads((ROOT / name).read_text())

def rows(name):
    path = ROOT / name
    if path.exists():
        return [json.loads(line) for line in path.read_text().splitlines()]
    with gzip.open(str(path)+'.gz', 'rt') as stream:
        return [json.loads(line) for line in stream]

def quiet_reason(samples, policy):
    if not samples:
        return 'no observations'
    last = samples[-1]
    starts = [i for i,s in enumerate(samples) if s['end']['ns'] <= last['end']['ns']-policy['windowNS']]
    if not starts:
        return 'window incomplete'
    first = starts[-1]
    for i in range(first,len(samples)):
        sample = samples[i]
        begin,end = sample['begin'],sample['end']
        if not begin['boot'] or begin['boot'] != end['boot'] or end['boot'] != last['end']['boot'] or begin['ns'] <= 0 or end['ns'] < begin['ns'] or end['ns']-begin['ns'] > policy['maximumReadNS']:
            return 'pressure read delayed or clock changed'
        if set(sample['totalsUS']) != {'cpu','io','memory'} or min(sample['totalsUS'].values()) < 0:
            return 'invalid pressure counters'
        if i == first:
            continue
        previous = samples[i-1]
        gap = end['ns']-previous['end']['ns']
        if gap <= 0 or gap > policy['maximumGapNS'] or begin['ns'] < previous['end']['ns']:
            return 'sample gap or clock discontinuity'
        for key in ('cpu','io','memory'):
            delta = sample['totalsUS'][key]-previous['totalsUS'][key]
            if delta < 0:
                return 'pressure counter reset'
            if delta > gap//10000:
                return 'pressure in window'
    return 'quiet window'

def period(samples, begin, end, policy):
    selected = [r['pressure'] for r in samples if begin <= r['pressure']['begin']['ns'] and r['pressure']['end']['ns'] <= end]
    intervals = []
    for a,b in zip(selected,selected[1:]):
        gap = b['end']['ns']-a['end']['ns']
        assert gap > 0
        delta = {key:b['totalsUS'][key]-a['totalsUS'][key] for key in ('cpu','io','memory')}
        assert min(delta.values()) >= 0
        intervals.append({'ns':gap,'delta':delta})
    total = sum(r['ns'] for r in intervals)
    windows = [{'sampleIndex':i,'endNS':s['end']['ns'],'reason':quiet_reason(selected[:i+1],policy)} for i,s in enumerate(selected)]
    quiet = [r for r in windows if r['reason']=='quiet window']
    return {'beginNS':begin,'endNS':end,'elapsedSeconds':(end-begin)/1e9,'samples':len(selected),'intervals':len(intervals),
            'fullyObservedSeconds':total/1e9,'quietWindowCount':len(quiet),
            'firstQuietSeconds':(quiet[0]['endNS']-begin)/1e9 if quiet else None,
            'finalWindowQuiet':bool(windows and windows[-1]['reason']=='quiet window'),
            'windowReasonCounts':dict(Counter(r['reason'] for r in windows)),
            'pressure':{key:{'weightedPercent':sum(r['delta'][key] for r in intervals)*100000/total if total else None,
                             'maximumPercent':max((r['delta'][key]*100000/r['ns'] for r in intervals),default=None),
                             'intervalsAboveThreshold':sum(r['delta'][key]>r['ns']//10000 for r in intervals)}
                        for key in ('cpu','io','memory')},
            'windows':windows}

def analyse():
    start,end = read('receipts/start.json'),read('receipts/end.json')
    samples,trace = rows('samples.jsonl'),rows('trace.jsonl')
    policy = read('inputs/policy.json')
    assert policy['maximumPressurePPM']==100000
    assert len(samples)==end['samples'] and len(trace)==end['traceRows']
    assert all(row['sequence']==i for i,row in enumerate(samples))
    for row in samples:
        s=row['pressure']
        for key,value in s['raw'].items():
            line=next(x for x in value.splitlines() if x.startswith('some '))
            assert int(dict(x.split('=') for x in line.split()[1:])['total'])==s['totalsUS'][key]
    boundaries=end['boundaries']
    periods={}
    if 'baselineEndNS' in boundaries:
        periods['baseline']=period(samples,boundaries['baselineBeginNS'],boundaries['baselineEndNS'],policy)
    else:
        periods['baseline']=period(samples,boundaries['baselineBeginNS'],end['endedBootNS'],policy)
    if 'preparationExitObservedNS' in boundaries:
        periods['preparation']=period(samples,boundaries['preparationLaunchNS'],boundaries['preparationExitObservedNS'],policy)
    if 'recoveryBeginNS' in boundaries:
        periods['recovery']=period(samples,boundaries['recoveryBeginNS'],boundaries.get('recoveryEndNS',end['endedBootNS']),policy)
    if 'recoveryBeginNS' in boundaries:
        periods['recoveryWithinWaitBudget']=period(samples,boundaries['recoveryBeginNS'],min(boundaries.get('recoveryEndNS',end['endedBootNS']),boundaries['recoveryBeginNS']+policy['maximumWaitNS']),policy)
    open_spans={}
    spans=[]
    stack=[]
    for event in trace:
        key=event['id']
        if event['event']=='BEGIN':
            assert key not in open_spans
            assert event['parent']==(stack[-1] if stack else 0)
            open_spans[key]=event
            stack.append(key)
            continue
        assert event['event']=='RETURN' and stack.pop()==key
        before=open_spans.pop(key)
        assert (before['kind'],before['detail'],before['parent'])==(event['kind'],event['detail'],event['parent'])
        a,b=before['resources'],event['resources']
        assert a['pid']==b['pid'] and a['startTicks']==b['startTicks'] and a['end']['boot']==b['begin']['boot']
        begin_ns,end_ns=a['end']['ns'],b['begin']['ns']
        assert end_ns>=begin_ns
        io_delta={k:b['io'][k]-a['io'][k] for k in a['io']}
        assert min(io_delta.values())>=0
        observation=period(samples,begin_ns,end_ns,policy)
        observation.pop('windows')
        spans.append({'id':key,'parent':event['parent'],'kind':event['kind'],'detail':event['detail'],
                      'beginNS':begin_ns,'endNS':end_ns,'elapsedSeconds':(end_ns-begin_ns)/1e9,
                      'cpuSeconds':sum(b['cpu'][k]-a['cpu'][k] for k in ('userNS','systemNS'))/1e9,
                      'ioBytes':io_delta,'pressure':observation})
    spans.sort(key=lambda r:r['id'])
    if end['status']=='COMPLETE':
        assert not open_spans and not stack and end['preparationExitCode']==0
        assert sum(r['kind']=='copy-tree' for r in spans)==6
        assert sum(r['kind']=='capture-state' for r in spans)==3
        assert sum(r['kind']=='verify-source' for r in spans)==1
        assert sum(r['kind']=='verify-state' for r in spans)==1
    checks=read('process-checks.json')
    pending=[]
    for sample in samples:
        value=sample['storage']['host']['pendingWrites']
        if value['available']:
            pending.append({'phase':sample['phase'],'beginNS':sample['pressure']['begin']['ns'],**value['value']})
    recovery=periods.get('recoveryWithinWaitBudget',{})
    decision='PREPARATION_DIAGNOSIS_INCOMPLETE'
    if end['status']=='COMPLETE':
        decision='QUIET_WINDOWS_AFTER_PREPARATION' if recovery.get('quietWindowCount',0)>0 else 'NO_QUIET_WINDOW_AFTER_PREPARATION'
    return {'schema':'buildopt.preparation-observation/analysis/v1','decision':decision,
            'startedUTC':start['atUTC'],'endedUTC':end['endedUTC'],'observationStatus':end['status'],
            'elapsedSeconds':(end['endedBootNS']-start['bootNS'])/1e9,
            'periods':periods,'spans':spans,'unfinishedSpans':list(open_spans.values()),
            'processChecks':len(checks),'indexerObservations':sum(bool(c['indexers']) for c in checks),
            'processChecksWithUnavailableEntries':sum(bool(c['unavailable']) for c in checks),
            'observerCPUSeconds':(end['observerAfter']['selfCPUNS']-end['observerBefore']['selfCPUNS'])/1e9,
            'minimumFreeBytes':min(r['freeBytes'] for r in samples),
            'maximumObservedFreeSpaceDecreaseBytes':max(0,start['freeBytes']-min(r['freeBytes'] for r in samples)),
            'pendingWrites':{phase:{'maximumDirtyKiB':max((x['Dirty'] for x in pending if x['phase']==phase),default=None),
                                    'maximumWritebackKiB':max((x['Writeback'] for x in pending if x['phase']==phase),default=None)}
                              for phase in ('baseline','preparation','recovery')},
            'projectBuilds':0,'comparisonJVMs':0,'performanceResult':None,
            'historicalCause':'unproven','buildAdmission':False}

if __name__=='__main__':
    result=analyse()
    if sys.argv[1:]==['--verify']:
        assert result==read('analysis.json')
        manifest=ROOT/'evidence-manifest.json'
        if manifest.exists():
            for item in read('evidence-manifest.json')['files']:
                path=(ROOT/item['path']).resolve()
                assert path.is_relative_to(ROOT)
                assert hashlib.sha256(path.read_bytes()).hexdigest()==item['sha256']
        print(result['decision']+'; complete retained observations verified; no build admission.')
    else:
        print(json.dumps(result,indent=2))
