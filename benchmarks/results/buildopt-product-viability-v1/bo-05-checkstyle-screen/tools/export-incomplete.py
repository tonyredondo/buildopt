"""Publish the incomplete record without changing the frozen experiment."""
from pathlib import Path
import hashlib
import json
import shutil

P=Path(__file__).resolve().parent
E=P.parents[3]/'benchmarks/results/buildopt-product-viability-v1/bo-05-checkstyle-screen'
def load(p): return json.loads(p.read_text())
v=load(P/'analysis/verification.json')
assert v['status']=='verified recovery of incomplete evidence'
r=load(P/'analysis/incomplete-profile-supported.json')
host={x['attempt']:x for x in load(P/'analysis/host-pressure.json')['requests']}
order=load(P/'analysis/order-and-costs.json')
mapping={
 'analysis/incomplete-profile-supported.json':'result.json',
 'analysis/verification.json':'verification.json',
 'analysis/host-pressure.json':'host-pressure.json',
 'analysis/diagnostic-phases.json':'task-execution.json',
 'analysis/order-and-costs.json':'order-and-costs.json',
 'profiles/supported/manifest.json':'manifest.json',
 'allocation.json':'allocation.json',
 'inputs/allocation-frozen.json':'allocation-frozen.json',
 'inputs/protocol.json':'protocol.json',
 'inputs/protocol.md':'protocol.md',
 'inputs/launch-freeze.json':'launch-freeze.json',
 'inputs/launch-admission.json':'launch-admission.json',
}
for name in ['prepare.py','run-screen.py','run-screen-v2.py','decision.py',
 'analyze-screen.py','analyze-incomplete.py','analyze-host.py','analyze-host-incomplete.py',
 'analyze-phases.py','analyze-phases-incomplete.py','analyze-phases-v2.py',
 'analyze-order.py','analyze-order-incomplete.py','verify-closeout.py','verify-incomplete.py',
 'test-screen-tools.py','test-screen-tools-v2.py','test-observer-repair.py','status.py','status-v2.py','export-incomplete.py']:
 mapping[name]='tools/'+name
 if (P/'inputs'/f'{name}.diff').exists(): mapping['inputs/'+name+'.diff']='tools/'+name+'.diff'
for name in ['readiness','controller-tests','manifest-validation','profile-supported-start',
 'profile-supported-end','observer-repair-design','observer-repair-tests','observer-test-fixture-correction',
 'incomplete-report-correction','postprocessing-design','export-design-amendment',
 'verification-performance-correction','status-helper-proof']:
 mapping['receipts/'+name+'.json']='receipts/'+name+'.json'
for path,target in [('receipts/reproducible-tool-tests/result.json','receipts/original-tool-tests.json'),
 ('receipts/revised-controller-tool-tests/result.json','receipts/revised-controller-tool-tests.json'),
 ('inputs/decision-tests.json','receipts/prospective-decision-tests.json'),
 ('inputs/analysis-bindings.json','receipts/analysis-bindings.json'),
 ('inputs/test-observer-repair-first.py','tools/test-observer-repair-first.py'),
 ('inputs/verify-incomplete-initial.py','tools/verify-incomplete-initial.py'),
 ('inputs/export-evidence-initial.py','tools/export-evidence-initial.py'),
 ('export-evidence.py','tools/export-evidence-unused.py')]: mapping[path]=target
for f in sorted((P/'profiles/supported/run/pairs').glob('*.json')):
 mapping[str(f.relative_to(P))]='raw-pairs/'+f.name
assert sum((P/path).stat().st_size for path in mapping)<20*2**20
E.mkdir()
index=[]
for src,dest in mapping.items():
 source=P/src;target=E/dest;target.parent.mkdir(parents=True,exist_ok=True)
 shutil.copyfile(source,target);digest=hashlib.sha256(source.read_bytes()).hexdigest()
 assert hashlib.sha256(target.read_bytes()).hexdigest()==digest
 index.append(dict(path=dest,source=str(source),bytes=source.stat().st_size,sha256=digest))
(E/'evidence-index.json').write_text(json.dumps(dict(files=index,rawStateRoot=str(P),
 rawEvidenceRetainedLocally=True,portability='All observed timing rows, all sixteen scheduled statuses, completed cost rows, unclosed cost markers, protocols, proof and analysis sources are included. Full inventories, generated outputs and process samples remain local. Independent output reconstruction did not run.'),indent=2)+'\n')
timings=[]
for pair in r['pairs']:
 for arm in ('N','I'):
  row=pair[arm]
  timings.append(f"| {pair['originalOrdinal']} | {'Cold' if pair['warmup'] else 'Measured'} | {arm} | {row['nativeMS']/1000:.3f} | {row['customerMS']/1000:.3f} | {host[row['attempt']]['contention']} |")
summary=r['summary']['customerMS']
all_saved=sum(x['customerSavingMS'] for x in r['pairs'])/1000
costs='\n'.join(f"| {key.replace('customer-machine','Required machine work').replace('research','Research preparation and checking')} | {seconds:.3f} |" for key,seconds in order['costSecondsByClassAndEnvelope'].items())
text=f'''# Checkstyle comparison on four changes

BO-05 · 14 September 2026 · **INCOMPLETE_SCREEN**.

Eight builds finished and all four live comparisons of their outputs passed.
A permissions error in the measurement script stopped the experiment before the
second sequence began. BO-05 remains incomplete. This result neither admits the
correction to long validation nor rejects it for lack of savings.

## What ran

Checkstyle checks source formatting and coding rules. The correction remembers
successful checks on unchanged files while preserving complete reports.
Both versions ran Elasticsearch's `:server:precommit --continue` command.
N used native Checkstyle; I used the supported V2 correction. Both retained
the earlier ForbiddenPatterns correction, which checks for disallowed text,
so this comparison concerns Checkstyle alone.

The [frozen protocol](./protocol.md) specifies two independent sequences of
changes 17–20, sixteen builds in total. Each starts from cold private state and
preserves that state through the following changes. The [manifest](./manifest.json)
fixes the revisions, eight-worker CPU profile, tools and output checks. Native
build caching is enabled; Configuration Cache remains disabled for this workflow.

## Every completed build

These are all eight builds in the first sequence. Whole-request time also
includes required machine work recorded outside the native command.

| Change | Role | Version | Native seconds | Whole-request seconds | Host flag |
|---|---|---|---:|---:|---|
{chr(10).join(timings)}

Across changes 18–20, the observed mean fell from
{summary['baselineMeanMS']/1000:.3f} to {summary['candidateMeanMS']/1000:.3f} seconds:
{summary['savingPercent']:.2f}% less time, or {summary['pairedMeanSavingMS']/1000:.3f}
seconds per build. All three differences favored the correction, including the
change with little opportunity. Including the cold build, the first sequence's
request times differed by {all_saved:.3f} seconds in favor of the correction.

These observations are provisional. The required second sequence and independent
output reconstruction are missing. No passing screen decision is assigned.
The [result](./result.json) retains every difference and its components; the
[task records](./task-execution.json) retain the work performed and reused.

The shared workstation showed pressure during several builds. All
[host flags and sampling gaps](./host-pressure.json) are retained. They do not
identify the cause of an individual timing, and no observation was dropped.
Sampled CPU masks do not establish uninterrupted isolation or general precision.

## What did not run

| Second sequence change | Native build | Corrected build |
|---|---|---|
| 17 | Not run | Preparation started; native command not started |
| 18 | Not run | Not run |
| 19 | Not run | Not run |
| 20 | Not run | Not run |

The [verification record](./verification.json) retains all sixteen scheduled
statuses. The independent check that reconstructs comparisons from their saved
files was not reached. The four live output checks remain evidence for their
original scope; the complete correctness requirement has not passed.

## Why it stopped and what was corrected

The controller recorded `PermissionError(13, 'Permission denied')` while
observing processes during preparation of the second sequence. It then stopped
the runner. Its error record did not retain the failing PID or file, so the
exact original read cannot be identified.

A [focused Linux test](./receipts/observer-repair-tests.json) reproduces the same
failure class: a process can remain visible while access to its disk counters
is denied. The original collector aborts on that read. The revised collector
records inaccessible disk counters and wait information as explicit gaps while
retaining process identity and CPU affinity. Other IO errors and failures to
read required identity or command information still fail. Future fatal errors
retain their file and traceback.

The resource report now keeps unavailable disk measurements empty instead of
inventing zeroes. The status helper also distinguishes a stopped controller
from an unfinished phase marker. Fourteen focused repair cases passed, along
with the [fourteen decision and eight collection checks](./receipts/revised-controller-tool-tests.json)
on the revised controller. One test-fixture correction is retained with its
explanation. No Gradle build or comparator JVM was started by these tests.

The revised collector has **not** run this build experiment. Its exact changes
are available in the [collector diff](./tools/run-screen-v2.py.diff) and
[resource-report diff](./tools/analyze-phases-v2.py.diff). The original frozen
files remain unchanged. Recovery analyses were added after the interruption;
they preserve the original timing, cost and host-flag rules and assign no
performance pass.

## Costs and closure

Sixteen build reservations remain charged. Eight builds and four live-comparison
JVMs actually ran; eight builds and all eight independent-comparison JVMs did
not. There were no retries. The allocation is closed and its unused reservations
are not reusable. The [cost and order record](./order-and-costs.json) retains all
44 completed cost rows. The unfinished preparation phase remains explicit in
the verification record, with no invented completion time.

| Recorded work and location | Seconds |
|---|---:|
{costs}

Rows inside a request are already included in its time and must not be added
again. Preparation and checking costs are retained separately. The observations
do not establish that future savings would recover preparation or maintenance.

The state uses {v['allocatedBytes']/2**30:.2f} GiB within the 64 GiB limit. All
three created services are inactive, and the controller and runner have ended.
Two services have their original closure records; the third is verified through
an external service-state observation. No missing owner record was fabricated.
All four worktrees and raw evidence remain available.

## Next step

Checkstyle remains the candidate awaiting a complete short comparison. Micronaut's
whole-workflow opportunity remains unmeasured; Spring and the reviewed Ktor/Beam
continuations have no admitted timing allocation. This interruption changes
none of those decisions.

Before another measured attempt, register a separate allocation and freeze the
revised observation sources. The proposed attempt retains the same correction,
commands, changes and criteria: two fresh sequences, sixteen builds, at most
sixteen comparator JVMs, one CPU profile, three hours and 64 GiB. Preserve this
interrupted attempt separately and do not choose the better result from repeated
sequences. The frozen protocol permits no automatic replacement trial, so none
has started. Protected changes 21–100 remain untouched.

Follow the [execution tracker](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md).
The [evidence index](./evidence-index.json) identifies exact copies and the local
raw files needed for further verification. Full generated outputs, inventories
and process samples remain local. Product viability remains **NOT_ESTABLISHED**.
'''
(E/'README.md').write_text(text)
print(json.dumps(dict(exportRoot=str(E),files=len(index)+2,bytes=sum(x['bytes'] for x in index),decision='INCOMPLETE_SCREEN')))
