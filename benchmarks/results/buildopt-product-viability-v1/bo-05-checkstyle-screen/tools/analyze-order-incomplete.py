"""Completed requests only; the separate incomplete record retains all unrun requests."""
"""Preserve execution order, idle intervals and every machine-cost row."""
from pathlib import Path
import hashlib
import json

P = Path(__file__).resolve().parent
RUN = P / 'profiles/supported/run'

def load(path):
    return json.loads(path.read_text())

def bound(binding):
    path = Path(binding['path'])
    assert hashlib.sha256(path.read_bytes()).hexdigest() == binding['sha256']
    return load(path)

host = {row['attempt']:row for row in load(P/'analysis/host-pressure.json')['requests']}
rows = []
for folder in sorted((RUN/'attempts').iterdir()):
    if not folder.is_dir() or not (folder / 'end.json').exists():
        continue
    start = load(folder/'start.json')
    native = bound(load(folder/'end.json')['native'])
    rows.append(dict(attempt=start['id'], replication=start['replication'],
        localOrdinal=start['ordinal'], originalOrdinal=start['ordinal']+17,
        arm=start['arm'], startBootNS=native['start']['ns'], endBootNS=native['end']['ns'],
        nativeSeconds=(native['end']['ns']-native['start']['ns'])/1e9,
        hostFlag=host[start['id']]['contention']))
assert len(rows) == 8
last = {}
ordered = sorted(rows,key=lambda r:r['startBootNS'])
for index, row in enumerate(ordered):
    if index:
        previous = ordered[index-1]
        assert previous['endBootNS'] <= row['startBootNS']
    key = (row['replication'], row['arm'])
    row['idleSinceOwnPreviousBuildSeconds'] = None if key not in last else (row['startBootNS']-last[key])/1e9
    last[key] = row['endBootNS']
    row['overallOrder'] = index+1
for replication in (1,):
    for ordinal in range(4):
        pair = sorted([r for r in rows if r['replication']==replication and r['localOrdinal']==ordinal],key=lambda r:r['startBootNS'])
        expected = ['N','I'] if (replication+ordinal)%2==1 else ['I','N']
        assert [r['arm'] for r in pair] == expected
costs = []
for file in sorted((RUN/'costs').glob('*.json')):
    row = load(file)
    if 'durationNS' in row and 'purpose' in row:
        costs.append(row)
assert len({r['id'] for r in costs}) == len(costs)
totals = {}
for row in costs:
    key = row['class'] + (' / inside request' if row['insideEnvelope'] else ' / outside request')
    totals[key] = totals.get(key,0)+row['durationNS']/1e9
report = dict(status='verified', rows=sorted(rows,key=lambda r:r['overallOrder']),
    costs=costs, costSecondsByClassAndEnvelope=totals,
    limits=['Inside-envelope rows must not be added to the request a second time',
            'Idle intervals include preservation, comparison and the other arm; they do not identify a slowdown cause',
            'Preparation and cold costs are retained; this short screen does not prove their recovery'])
with (P/'analysis/order-and-costs.json').open('x') as output:
    output.write(json.dumps(report,indent=2)+'\n')
print(json.dumps(dict(verified=True, requests=len(rows), costRows=len(costs), costSecondsByClassAndEnvelope=totals)))
