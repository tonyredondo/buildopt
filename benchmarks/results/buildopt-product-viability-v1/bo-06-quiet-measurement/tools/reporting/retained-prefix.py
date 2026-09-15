"""Account for the interrupted prefix without assigning savings to missing builds."""
from pathlib import Path
import sys
sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from common import P, load, save, bind

run = P / 'profiles/prefix/run'
result = load(run / 'result.json')
end = load(P / 'receipts/prefix-end.json')
assert end['exitCode'] == 1 and end['observationFailure'] is None
assert result['decision'] == 'INCOMPLETE_EVIDENCE'
assert result == load(run / 'results/0001.json')
assert len(list((run / 'results').glob('*.json'))) == 1
assert result['actualGradleStarts'] == result['workflowStarts'] == 17
assert result['gradleReservations'] == 18 and result['unknownGradleReservations'] == 1
assert result['nestedStarts'] == 0
slots = result['slots'][0]
assert len(slots) == 21
coverage = []
for ordinal, slot in enumerate(slots):
    assert slot['ordinal'] == ordinal
    if ordinal < 8:
        pair = load(run / 'pairs' / f'r1-{ordinal:03}.json')
        assert pair['slot'] == slot and slot['class'] == 'COMPARABLE'
        assert slot['ownerMetadataJVMStarts'] == 1
    elif ordinal == 8:
        assert slot['class'] == 'HARNESS_INVALID'
        assert slot['reason'] == 'partial pair retained without saving'
    else:
        assert slot['class'] == 'NOT_RUN_DEPENDENCY'
    for arm in ('N', 'I'):
        attempt = run / 'attempts' / f'r1-{ordinal:03}-g0-{arm}'
        if (attempt / 'native-finish.json').exists():
            finish = load(attempt / 'native-finish.json')
            envelope = load(attempt / 'end.json')
            assert finish['exitCode'] == 0 and finish['outcome'] == 'EXITED'
            assert ordinal < 8 or (ordinal == 8 and arm == 'N')
            outcome = 'COMPLETED_PAIR' if ordinal < 8 else 'COMPLETED_UNPAIRED'
            evidence = [bind(attempt / name) for name in ('start.json', 'native-finish.json', 'end.json')]
            seconds = envelope['durationNS'] / 1e9
        elif attempt.exists():
            assert ordinal == 8 and arm == 'I'
            assert sorted(x.name for x in attempt.iterdir()) == ['quiet-start.json', 'start.json']
            assert not (run / 'costs' / (attempt.name + '-request.json')).exists()
            assert not (run / 'costs' / (attempt.name + '-request-start.json')).exists()
            outcome, seconds = 'NOT_RUN_QUIET', None
            evidence = [bind(attempt / name) for name in ('start.json', 'quiet-start.json')]
        else:
            assert ordinal > 8
            outcome, seconds, evidence = 'NOT_RUN_DEPENDENCY', None, []
        coverage.append(dict(ordinal=ordinal, arm=arm, outcome=outcome,
                             wholeRequestSeconds=seconds, evidence=evidence))
assert len(list((run / 'pairs').glob('*.json'))) == 8
assert len(list((run / 'attempts').glob('*/native-start.json'))) == 17
assert len(list((run / 'attempts').glob('*/native-finish.json'))) == 17
quiet_path = run / 'attempts/r1-008-g0-I/quiet-start.json'
quiet = load(quiet_path)
assert quiet['decision'] == 'NOT_QUIET_TIMEOUT'
policy = load(quiet['policy']['path'])
intervals = []
for before, after in zip(quiet['samples'], quiet['samples'][1:]):
    ns = after['end']['ns'] - before['end']['ns']
    assert ns > 0
    fractions = {key: (after['totalsUS'][key] - before['totalsUS'][key]) * 1000 / ns
                 for key in ('cpu', 'io', 'memory')}
    intervals.append(dict(seconds=ns / 1e9, pressure=fractions))
assert len(intervals) == 179
assert all(row['pressure']['io'] > .10 for row in intervals)
seconds = sum(row['seconds'] for row in intervals)
pressure = {key: dict(intervalsAboveThreshold=sum(row['pressure'][key] > .10 for row in intervals),
                      weightedFraction=sum(row['pressure'][key] * row['seconds'] for row in intervals) / seconds,
                      maximumFraction=max(row['pressure'][key] for row in intervals))
            for key in ('cpu', 'io', 'memory')}
summary = dict(status='verified incomplete accounting', decision='INCOMPLETE_DEVELOPMENT_QUIET_TIMEOUT',
               complete=False, requests=17, scheduledBuilds=42, unstartedBuilds=25,
               livePairs=8, independentPairs=8, coverage=coverage,
               result=bind(run / 'result.json'), independentResult=bind(run / 'results/0001.json'),
               runnerDecision=result['decision'], runnerUnknownReservations=1,
               unknownReservationExplanation='The runner conservatively retains the unlaunched I8 reservation as unknown. The timeout receipt and frozen pre-launch return path identify why it has no native invocation; the runner result is not rewritten.',
               refusal=dict(evidence=bind(quiet_path), seconds=(quiet['end']['ns']-quiet['begin']['ns'])/1e9,
                            intervals=len(intervals), pressure=pressure, intervalsData=intervals),
               netSaving=None, interpretation='The scheduled sequence is incomplete. Completed pairs cannot qualify a complete-prefix or product saving. Storage pressure is observed, not attributed to a particular process; PSI is stall time, not disk utilization.')
save(P / 'analysis/prefix-retained-summary.json', summary)
print({key: summary[key] for key in ('decision','requests','unstartedBuilds','livePairs','independentPairs')})
