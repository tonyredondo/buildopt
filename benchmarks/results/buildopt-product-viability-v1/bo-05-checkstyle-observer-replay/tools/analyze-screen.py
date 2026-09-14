"""Reconstruct exploratory profile metrics from qualified runner evidence."""
from pathlib import Path
import datetime, hashlib, importlib.util, json, os, statistics, subprocess, sys

P = Path(__file__).resolve().parent
R = P.parents[3]
I = P.parent / 'bv006-cpu-isolation'
CHECKSTYLE_TASKS = {':server:checkstyleMain', ':server:checkstyleTest', ':server:checkstyleInternalClusterTest'}
def load(f): return json.loads(Path(f).read_text())
def sha(f): return hashlib.sha256(Path(f).read_bytes()).hexdigest()
def bound(b):
    assert sha(b['path']) == b['sha256'], b['path']
    return load(b['path'])
def save(f, x):
    assert not f.exists(), f
    f.write_text(json.dumps(x, indent=2) + '\n')
def utc(value): return datetime.datetime.fromisoformat(value.replace('Z', '+00:00')).timestamp()
def properties(path):
    data = Path(path).read_bytes()
    assert data[64:65] == b'\n' and hashlib.sha256(data[65:]).hexdigest().encode() == data[:64]
    def unescape(value):
        result = ''; index = 0
        while index < len(value):
            char = value[index]; index += 1
            if char != '\\': result += char; continue
            assert index < len(value)
            char = value[index]; index += 1
            if char == 'u':
                result += chr(int(value[index:index+4], 16)); index += 4
            else: result += {'n':'\n','t':'\t','r':'\r','f':'\f'}.get(char, char)
        return result
    result = {}
    for line in data[65:].decode('iso-8859-1').splitlines():
        if not line or line.startswith(('#', '!')): continue
        escaped = False
        for index, char in enumerate(line):
            if not escaped and char == '=':
                key = unescape(line[:index]); assert key not in result
                result[key] = unescape(line[index+1:]); break
            escaped = not escaped if char == '\\' else False
        else: raise ValueError('Unsupported property record')
    assert result['version'] == '1'
    return result
def cgroup_cpu(resource):
    fields = dict(line.split() for line in resource['cgroupFiles']['cpu.stat'].splitlines())
    return int(fields['usage_usec']) / 1000
def summary(pairs, native, candidate):
    left = [pair['N'][native] for pair in pairs]
    right = [pair['I'][candidate] for pair in pairs]
    delta = [a - b for a, b in zip(left, right)]
    return dict(baselineMeanMS=statistics.mean(left), candidateMeanMS=statistics.mean(right), pairedMeanSavingMS=statistics.mean(delta), pairedMedianSavingMS=statistics.median(delta), savingPercent=100 * sum(delta) / sum(left), minimumPairedSavingMS=min(delta), maximumPairedSavingMS=max(delta), individualDeltasMS=delta)

def validate_masks(samples, workers, cpus, readiness):
    # A failed worker-isolation observation must remain failed in the report.
    # Native timing/output evidence can still support an exploratory negative decision.
    masks = dict(observerThreads=0, nativeThreads=0, nativeProcesses=0,
                 preExecWorkerThreads=0, preExecObservations=[], workerAffinityObservations=[])
    expected = ','.join(map(str, range(cpus)))
    for sample in samples:
        for group in sample['groups']:
            scope = (group['cgroup'], group['unit'])
            for process in group['processes']:
                identity = (process['identity']['pid'], process['identity']['startTicks'])
                observer = identity in workers
                if observer:
                    assert workers[identity] == scope, (identity, scope)
                assert process['threadMasks'], (identity, 'missing masks')
                threads = sum(map(len, process['threadMasks'].values()))
                if observer and any(mask != '8' for mask in process['threadMasks']):
                    masks['workerAffinityObservations'].append(dict(
                        beginBootNS=sample['beginBootNS'], endBootNS=sample['endBootNS'],
                        scope=list(scope), identity=process['identity'],
                        threadMasks=process['threadMasks'],
                        commandSHA256=hashlib.sha256(process['command'].encode()).hexdigest(),
                        qualification='FAILED_WORKER_EXCLUSIVE_AFFINITY'))
                    continue
                required = '8' if observer else expected
                assert all(mask == required for mask in process['threadMasks']), (identity, required)
                masks['observerThreads' if observer else 'nativeThreads'] += threads
                if not observer:
                    masks['nativeProcesses'] += 1
    assert masks['observerThreads'] and masks['nativeThreads']
    masks['nativeThreadAffinityVerified'] = True
    masks['workerExclusiveAffinityVerified'] = not masks['workerAffinityObservations']
    masks['strictAllTimeAffinityQualified'] = False
    masks['qualificationLimit'] = 'Sampled affinity only; no continuous isolation or G0/G3 qualification'
    masks['nativeAndRunningWorkerAffinityVerified'] = masks['workerExclusiveAffinityVerified']
    return masks

def analyze(label):
    assert label == 'supported'
    cpus = 8
    offset = 17
    profile = P / 'profiles' / label
    run = profile / 'run'
    m = load(profile / 'manifest.json')
    pair_count = len(m['history']) * m['replications']
    starts = 2 * pair_count
    receipt = load(P / f'receipts/profile-{label}-end.json')
    assert receipt['exitCode'] == 0 and receipt['observationFailure'] is None
    assert sha(P / f'logs/profile-{label}.log') == receipt['logSHA256']
    assert sha(profile / 'process-samples.jsonl') == receipt['processSamplesSHA256']
    result = load(run / 'result.json')
    assert result['decision'] == 'FIXTURE_VERIFIED' and result['workflowStarts'] == result['actualGradleStarts'] == result['gradleReservations'] == starts
    assert result['nestedStarts'] == result['unknownGradleReservations'] == 0
    slots = [slot for replication in result['slots'] for slot in replication]
    assert len(slots) == pair_count and all(slot['class'] == 'COMPARABLE' for slot in slots)
    # run already reconstructs every pair independently after the live comparison.
    # Binding its final JSON proves which independent result was actually emitted.
    emitted = json.loads((P / f'logs/profile-{label}.log').read_text())
    assert emitted == result
    records = {(record['replication'], record['ordinal']): record for f in (run / 'pairs').glob('*.json') if (record := load(f))}
    assert sorted(records) == [(rep, ordinal) for rep in range(1, m['replications']+1) for ordinal in range(len(m['history']))]
    costs = [value for f in (run / 'costs').glob('*.json') if 'durationNS' in (value := load(f)) and 'purpose' in value]
    samples = [json.loads(line) for line in (profile / 'process-samples.jsonl').read_text().splitlines()]
    assert samples
    workers = {}
    readiness = {}
    for record in records.values():
        for binding in record['attempts']:
            native = bound(bound(binding)['native'])
            supervisor = native['supervisor']
            identity = (supervisor['pid'], supervisor['startTicks'])
            scope = (supervisor['cgroup'], native['unit'])
            assert identity not in workers or workers[identity] == scope
            workers[identity] = scope
            ready = native['resourcesBefore']
            assert ready['cpuAffinity'] == '8' and ready['at']['ns'] < native['start']['ns']
            if identity not in readiness or ready['at']['ns'] < readiness[identity]['atNS']:
                readiness[identity] = dict(atNS=ready['at']['ns'],nativeStartNS=native['start']['ns'],cpuAffinity=ready['cpuAffinity'])
    assert len(workers) == 4
    masks = validate_masks(samples, workers, cpus, readiness)
    pairs = []
    previous_states = {}
    daemon_pids = {(rep,arm):set() for rep in range(1,m['replications']+1) for arm in ('N','I')}
    for (replication, ordinal), pair_record in sorted(records.items()):
        slot = pair_record['slot']
        assert slot == result['slots'][replication-1][ordinal], (replication, ordinal, 'live and independent slots differ')
        pair = dict(replication=replication, localOrdinal=ordinal, originalOrdinal=ordinal + offset, commit=m['history'][ordinal]['commit'], warmup=ordinal == 0, ownerOutputEquivalent=True)
        for binding in pair_record['attempts']:
            end = bound(binding)
            directory = Path(binding['path']).parent
            start = load(directory / 'start.json')
            arm = start['arm']
            assert start['replication'] == replication and start['ordinal'] == ordinal and start['reservedGradleStarts'] == 1
            assert end['class'] == 'COMPARABLE' and end['candidateApplied'] == (arm == 'I')
            native = bound(end['native'])
            capture = bound(end['capture'])
            after = bound(end['after'])
            assert native['exitCode'] == 0 and native['outcome'] == 'EXITED'
            assert sha(directory / 'stdout.log') == native['stdoutSHA256'] and sha(directory / 'stderr.log') == native['stderrSHA256']
            assert len(capture['gradleBuilds']) == 1
            daemon = capture['gradleBuilds'][0]['pid']
            daemon_pids[(replication,arm)].add(daemon)
            coverage = [sample['beginBootNS'] for sample in samples if native['start']['ns'] <= sample['beginBootNS'] <= native['end']['ns'] and any(process['identity']['pid'] == daemon for group in sample['groups'] for process in group['processes'])]
            assert len(coverage) >= 2, (ordinal, arm, 'missing daemon observations')
            cpu = load(directory / 'supervisor-cpu.json')
            assert all(cpu['end'][key] >= cpu['begin'][key] for key in ('userNS', 'systemNS'))
            own_cpu = cgroup_cpu(native['resourcesAfter']) - cgroup_cpu(native['resourcesBefore'])
            assert own_cpu >= 0
            row = dict(attempt=start['id'], nativeMS=(native['end']['ns']-native['start']['ns'])/1e6, customerMS=end['durationNS']/1e6, ownedServiceCPUMS=own_cpu, supervisorCPUMS=sum(cpu['end'][key]-cpu['begin'][key] for key in ('userNS','systemNS'))/1e6, daemonPID=daemon, observedDaemonSpanMS=(max(coverage)-min(coverage))/1e6, maximumDaemonGapMS=max(b-a for a,b in zip([native['start']['ns']]+coverage, coverage+[native['end']['ns']]))/1e6, checkstyle=[], candidateSuccessState=[])
            outside = [cost for cost in costs if cost['class'] == 'customer-machine' and not cost['insideEnvelope'] and cost['replication'] == replication and cost['ordinal'] == ordinal and cost['arm'] == arm]
            row['requestMS'] = row['customerMS']
            row['outsideRequestCustomerMS'] = sum(cost['durationNS'] for cost in outside) / 1e6
            row['outsideRequestCustomerCosts'] = [cost['id'] for cost in outside]
            row['customerMS'] += row['outsideRequestCustomerMS']
            graphs = []
            for b in capture['graph']:
                assert sha(b['path']) == b['sha256']
                graphs.extend(json.loads(line) for line in Path(b['path']).read_text().splitlines())
            for event in graphs:
                if event['kind'] == 'task' and event['task']['identity'] in CHECKSTYLE_TASKS:
                    row['checkstyle'].append(dict(task=event['task']['identity'], outcome=event['task']['outcome'], action=event['task']['action'], wallMS=(utc(event['utc'])-utc(event['startedUTC']))*1000))
            assert row['checkstyle'], (ordinal, arm)
            retained = load(profile / 'candidate-state' / start['id'] / 'receipt.json')
            assert retained['native'] == end['native']
            entries = {entry['path']:entry for entry in after['entries']}
            for entry in retained['entries']:
                assert sha(entry['copy']) == entry['sha256'] == entries[entry['path']]['sha256']
                state = properties(entry['copy'])
                files = {key[5:]:value for key,value in state.items() if key.startswith('file:')}
                task = ':server:' + Path(entry['path']).parent.name
                reports = [out for out in capture['outputs'] if out['producer'] == task and out['entry']['path'].endswith('.xml')]
                assert len(reports) == 1 and reports[0]['raw']['sha256'] == state['reportHash']
                previous = previous_states.get((replication, arm, task))
                inferred = 0
                executed = any(item['task'] == task and item['action'] for item in row['checkstyle'])
                if executed and previous and previous['identity'] == state['identity']:
                    inferred = sum(previous.get('file:' + name) == digest for name,digest in files.items())
                row['candidateSuccessState'].append(dict(task=task, taskExecuted=executed, files=len(files), contextIdentity=state['identity'], inferredReusableFiles=inferred, reuseInference='same context and earlier complete success/report retained by runner; execution counts come from task records'))
                previous_states[(replication, arm, task)] = state
            if arm == 'N':
                assert not row['candidateSuccessState']
            if arm == 'I':
                assert {x['task'] for x in row['candidateSuccessState']} == CHECKSTYLE_TASKS and all(x['files'] > 0 for x in row['candidateSuccessState'])
            pair[arm] = row
        assert all(arm in pair for arm in ('N','I'))
        pair['customerSavingMS'] = pair['N']['customerMS'] - pair['I']['customerMS']
        pair['customerSavingPercent'] = 100 * pair['customerSavingMS'] / pair['N']['customerMS']
        pairs.append(pair)
    assert all(len(pids) == 1 for pids in daemon_pids.values()), daemon_pids
    for configpath in (run / 'sessions').glob('*/worker-config.json'):
        cfg = load(configpath); closure = load(configpath.parent / 'closed.json')
        assert not closure['remaining'] and not (Path('/sys/fs/cgroup') / closure['cgroup'].lstrip('/')).exists()
        assert cfg['nativeAffinity'] == m['affinity'] and cfg['observerAffinity'] == '8'
    measured = [pair for pair in pairs if not pair['warmup']]
    metrics = {metric: summary(measured, metric, metric) for metric in ('customerMS','nativeMS','ownedServiceCPUMS','supervisorCPUMS')}
    helper = 2 * sum(slot['ownerMetadataJVMStarts'] for slot in slots)
    assert helper <= starts
    active = sum(task['action'] for pair in measured for task in pair['N']['checkstyle'])
    reusable = sum(state['inferredReusableFiles'] for pair in measured for state in pair['I']['candidateSuccessState'])
    from decision import classify
    decision = classify(pairs)
    report = dict(schema='buildopt.checkstyle-native-screen/v1', status='verified',
        profile=label, workers=cpus, armLabels={'N':'native Checkstyle', 'I':'supported V2 correction'},
        gradleActual=starts, metadataJVMActual=helper, warmupBuilds=4, measuredPairs=len(measured),
        pairs=pairs, summary=metrics, replicationDecisions=decision['replications'],
        decision=decision['decision'], materialSignal=decision['materialSignal'],
        baselineCheckstyleActions=active, measuredReusableFiles=reusable,
        allThreeCandidateTaskStatesVerified=True, masks=masks,
        cpuIsolationQualified=False,
        readinessQualified=False, valueQualified=False, productViability='NOT_ESTABLISHED',
        metadataAccounting='Eight live and eight independently reconstructed output comparisons; no third pass',
        analyzer=dict(path=str(Path(__file__).resolve()),sha256=sha(__file__)),
        rawResult=dict(path=str(run/'result.json'),sha256=sha(run/'result.json')),
        limits=['Six post-cold pairs across two replications; no precision or sustained-value claim',
                'Fresh state starts at original17; the earlier development state is not reconstructed',
                'Cold and setup costs remain separate and are not assumed recovered',
                'Shared host and the symmetric phase agent can affect timing; retain every row and flag',
                'Pinned Gradle9.7.1 with Configuration Cache disabled; formal G0 and G3 remain unqualified'])
    save(P / f'analysis/profile-{label}.json', report)
    print(json.dumps({k:v for k,v in report.items() if k != 'pairs'}))

if __name__ == '__main__': analyze(sys.argv[1])
