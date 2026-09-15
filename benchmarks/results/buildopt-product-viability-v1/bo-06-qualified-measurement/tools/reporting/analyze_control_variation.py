"""Describe the retained control difference without changing its decision."""
from pathlib import Path
import sys
sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from common import *
from collections import Counter
from datetime import datetime

summary = load(P / 'analysis/control-summary-v4.json')
host = load(P / 'analysis/control-host-pressure.json')
rows = []
sources = []
for ordinal in range(4):
    arms = {}
    for arm in ('N', 'I'):
        attempt = f'r1-{ordinal:03d}-g0-{arm}'
        graph = P / 'profiles/control/run/attempts' / attempt / 'graph.jsonl'
        sources.append(bind(graph))
        tasks = []
        for line in graph.open():
            event = json.loads(line)
            if event['kind'] != 'task':
                continue
            start = datetime.fromisoformat(event['startedUTC'])
            end = datetime.fromisoformat(event['utc'])
            tasks.append(dict(identity=event['task']['identity'],
                              outcome=event['task']['outcome'],
                              action=event['task']['action'],
                              seconds=(end - start).total_seconds()))
        assert len({t['identity'] for t in tasks}) == len(tasks)
        arms[arm] = tasks
    n = {t['identity']: t for t in arms['N']}
    i = {t['identity']: t for t in arms['I']}
    differences = [dict(identity=k, N=n[k]['seconds'], I=i[k]['seconds'],
                        differenceSeconds=n[k]['seconds'] - i[k]['seconds'])
                   for k in sorted(n.keys() & i.keys())]
    outcomes = lambda tasks: Counter((t['identity'], t['outcome'], t['action']) for t in tasks)
    rows.append(dict(originalOrdinal=ordinal + 17,
                     sameTaskOutcomes=outcomes(arms['N']) == outcomes(arms['I']),
                     taskCount={a: len(t) for a, t in arms.items()},
                     outcomes={a: dict(Counter(t['outcome'] for t in ts)) for a, ts in arms.items()},
                     largestTaskDifferences=sorted(differences, key=lambda t: abs(t['differenceSeconds']), reverse=True)[:12],
                     checkstyle=[t for t in differences if t['identity'].startswith(':server:checkstyle')]))
result = dict(status='verified descriptive analysis',
              decisionUnchanged=summary['decision'],
              graphSources=sources, rows=rows, hostRequests=host['requests'],
              limits='Task durations overlap and cannot be added to explain wall time. Host pressure is a correlation, not attribution to a specific process. No build or comparator was rerun; no observation was excluded.')
save(P / 'analysis/control-variation.json', result)
print(json.dumps(rows[-1], indent=2))
