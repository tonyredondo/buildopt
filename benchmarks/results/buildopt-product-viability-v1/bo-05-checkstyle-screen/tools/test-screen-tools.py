"""Reproduce decision and observation checks without a Gradle or JVM start."""
from pathlib import Path
import ast
import datetime
import hashlib
import importlib.util
import io
import json
import sys
import time
import types

sys.dont_write_bytecode = True
P = Path(__file__).resolve().parent
OUT = Path(sys.argv[1]).resolve()
OUT.mkdir()
spec = importlib.util.spec_from_file_location('decision', P/'decision.py')
decision = importlib.util.module_from_spec(spec)
spec.loader.exec_module(decision)

def sample():
    return [dict(replication=r, originalOrdinal=o,
        N=dict(customerMS=20000, checkstyle=[dict(action=o in (19,20))]),
        I=dict(customerMS=18000 if o in (19,20) else 20000,
               candidateSuccessState=[dict(inferredReusableFiles=100 if o in (19,20) else 0)]))
        for r in (1,2) for o in (17,18,19,20)]

results = []
def check(name, rows, expected):
    result = decision.classify(rows)
    assert result['materialSignal'] == expected, (name,result)
    results.append(dict(case=name, passed=True, expectedMaterialSignal=expected))

check('two-sequences-active-and-inactive', sample(), True)
for name, transform, expected in [
    ('same-code-zero-savings', lambda r: r['N']['customerMS'], False),
    ('failed-replication-not-pooled', lambda r:21000 if r['replication']==2 and r['originalOrdinal']>17 else r['I']['customerMS'], False),
    ('inactive-regression-stays-in-denominator', lambda r:26000 if r['originalOrdinal']==18 else r['I']['customerMS'], False),
    ('exact-inclusive-threshold', lambda r:18500 if r['originalOrdinal'] in (19,20) else r['I']['customerMS'], True),
    ('below-threshold', lambda r:18501 if r['originalOrdinal'] in (19,20) else r['I']['customerMS'], False),
    ('cold-cost-not-hidden-or-used-as-measured-credit', lambda r:200000 if r['originalOrdinal']==17 else r['I']['customerMS'], True),
    ('one-large-win-is-not-two-positive-pairs', lambda r:21000 if r['originalOrdinal'] in (18,19) else 1000 if r['originalOrdinal']==20 else r['I']['customerMS'], False),
]:
    rows = sample()
    for row in rows:
        row['I']['customerMS'] = transform(row)
    check(name, rows, expected)
    if name.startswith('cold-cost'):
        assert all(r['coldExcessMS']==180000 and r['allFourRequestSavingMS']<0 for r in decision.classify(rows)['replications'])
for name, divisor, extra in [('percent-without-one-second-floor',10,0),('seconds-without-percent-floor',1,100000)]:
    rows = sample()
    for row in rows:
        for arm in ('N','I'):
            row[arm]['customerMS'] = row[arm]['customerMS']/divisor+extra
    check(name,rows,False)
rows = sample()
for row in rows:
    row['N']['checkstyle'][0]['action'] = False
check('no-native-work',rows,False)
rows = sample()
for row in rows:
    row['I']['candidateSuccessState'][0]['inferredReusableFiles'] = 0
check('no-reuse',rows,False)
for name,rows in [('missing-pair',sample()[:-1]),('duplicate-pair',sample()[:-1]+[sample()[0]])]:
    try:
        decision.classify(rows)
    except AssertionError:
        results.append(dict(case=name,passed=True,expected='reject incomplete input'))
    else:
        raise AssertionError(name)

source = (P/'run-screen.py').read_text()
node = next(n for n in ast.parse(source).body if isinstance(n,ast.FunctionDef) and n.name=='observe')
code = compile(ast.Module(body=[node],type_ignores=[]),str(P/'run-screen.py'),'exec')
cases = [
    ('native-cold','N',0,True,False,0,False),
    ('candidate-cold','I',0,True,True,3,False),
    ('native-inactive','N',1,False,False,0,False),
    ('candidate-inactive','I',1,False,False,3,False),
    ('native-must-not-get-state','N',1,False,False,1,True),
    ('candidate-missing-state','I',1,False,False,2,True),
    ('native-cold-missing-phase','N',0,False,False,0,True),
    ('candidate-cold-missing-phase','I',0,True,False,3,True),
]
controller_results = []
for name,arm,ordinal,nativephase,candidatephase,states,rejected in cases:
    root = OUT/name
    run,profile = root/'run',root/'profile'
    attempt = run/'attempts/request'
    attempt.mkdir(parents=True)
    def write(path,obj):
        path.write_text(json.dumps(obj)+'\n')
    write(attempt/'native-finish.json',dict(exitCode=0))
    write(attempt/'start.json',dict(arm=arm,ordinal=ordinal,replication=1))
    logs = 'phase=com/puppycrawl/tools/checkstyle/Checker.process\n' if nativephase else ''
    if candidatephase:
        logs += 'ContentAwareChecker.process CheckstyleSuccessHistory$Commit.execute CheckstyleSuccessHistory.context\n'
    (attempt/'stdout.log').write_text(logs)
    for task in ['checkstyleMain','checkstyleTest','checkstyleInternalClusterTest'][:states]:
        f = run/'r1'/arm/'repo/.gradle/buildopt-checkstyle/server'/task/'success'
        f.parent.mkdir(parents=True)
        f.write_text('fixture state')
    def bind(path):
        return dict(path=str(path),sha256=hashlib.sha256(path.read_bytes()).hexdigest())
    def save(path,data):
        assert not path.exists()
        write(path,data)
    ns = dict(Path=Path,json=json,time=time,hashlib=hashlib,run=run,profile=profile,
        groups={},samples=0,completed=set(),m={'affinity':'0-7'},
        load=lambda q:json.loads(q.read_text()),host_observation=lambda:{},
        bindings=types.SimpleNamespace(bind=bind),save=save,
        now=lambda:datetime.datetime.now(datetime.timezone.utc).isoformat(),offset=17)
    exec(code,ns)
    try:
        ns['observe'](io.StringIO())
        actual = False
    except AssertionError:
        actual = True
    assert actual == rejected,(name,actual,rejected)
    if not rejected:
        record = json.loads((profile/'candidate-state/request/receipt.json').read_text())
        assert record['originalOrdinal']==ordinal+17 and len(record['entries'])==states
    controller_results.append(dict(case=name,passed=True,rejected=actual))
assert len(results)==14 and len(controller_results)==8
report = dict(status='verified',decisionCases=results,controllerCases=controller_results,
    nativeStarts=0,helperJVMs=0,
    inputs={name:hashlib.sha256((P/name).read_bytes()).hexdigest()
            for name in ('decision.py','run-screen.py','test-screen-tools.py')})
(OUT/'result.json').write_text(json.dumps(report,indent=2)+'\n')
print('14 decision cases and 8 actual observation-callback cases passed; no build or JVM started.')
