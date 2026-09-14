"""Export the complete short-screen result; retain large raw inventories locally."""
from pathlib import Path
import hashlib
import json
import shutil

P = Path(__file__).resolve().parent
R = P.parents[3]
E = R / 'benchmarks/results/buildopt-product-viability-v1/bo-05-checkstyle-screen'

def load(path):
    return json.loads(path.read_text())

verification = load(P/'analysis/verification.json')
assert verification['status'] == 'verified'
profile = load(P/'analysis/profile-supported.json')
host = {row['attempt']:row for row in load(P/'analysis/host-pressure.json')['requests']}
phases = load(P/'analysis/diagnostic-phases.json')
order = load(P/'analysis/order-and-costs.json')
mapping = {
    'analysis/profile-supported.json':'result.json',
    'analysis/verification.json':'verification.json',
    'analysis/host-pressure.json':'host-pressure.json',
    'analysis/diagnostic-phases.json':'task-execution.json',
    'analysis/order-and-costs.json':'order-and-costs.json',
    'profiles/supported/manifest.json':'manifest.json',
    'profiles/supported/run/result.json':'runner-result.json',
    'allocation.json':'allocation.json',
    'inputs/allocation-frozen.json':'allocation-frozen.json',
    'inputs/protocol.json':'protocol.json',
    'inputs/protocol.md':'protocol.md',
    'inputs/launch-freeze.json':'launch-freeze.json',
    'inputs/launch-admission.json':'launch-admission.json',
    'inputs/decision-tests.json':'receipts/decision-tests.json',
    'inputs/analysis-bindings.json':'receipts/analysis-bindings.json',
    'receipts/readiness.json':'receipts/readiness.json',
    'receipts/controller-tests.json':'receipts/controller-tests.json',
    'receipts/manifest-validation.json':'receipts/manifest-validation.json',
    'receipts/profile-supported-start.json':'receipts/run-start.json',
    'receipts/profile-supported-end.json':'receipts/run-end.json',
}
for name in ['prepare.py','run-screen.py','decision.py','analyze-screen.py','analyze-host.py',
             'analyze-phases.py','analyze-order.py','verify-closeout.py','export-evidence.py']:
    mapping[name] = 'tools/' + name
    diff = 'inputs/' + name + '.diff'
    if (P/diff).exists():
        mapping[diff] = 'tools/' + name + '.diff'
assert sum((P/name).stat().st_size for name in mapping) < 20*2**20
E.mkdir()
index = []
for source, target in mapping.items():
    src, dest = P/source, E/target
    dest.parent.mkdir(parents=True,exist_ok=True)
    shutil.copyfile(src,dest)
    digest = hashlib.sha256(src.read_bytes()).hexdigest()
    assert hashlib.sha256(dest.read_bytes()).hexdigest() == digest
    index.append(dict(path=target, source=str(src), bytes=src.stat().st_size, sha256=digest))
(E/'evidence-index.json').write_text(json.dumps(dict(status='verified exact copies',
    rawStateRoot=str(P), files=index, rawEvidenceRetainedLocally=True,
    portability='Reports, protocols, tools and all timing rows are included. Full output inventories, generated files and process samples remain at the bound local paths; this export alone cannot rerun the full output comparison.'),indent=2)+'\n')
timings = []
for pair in profile['pairs']:
    for arm in ('N','I'):
        row = pair[arm]
        timings.append(f"| {pair['replication']} | {pair['originalOrdinal']} | {'Cold' if pair['warmup'] else 'Measured'} | {arm} | {row['nativeMS']/1000:.3f} | {row['customerMS']/1000:.3f} | {host[row['attempt']]['contention']} |")
replications = []
for row in profile['replicationDecisions']:
    replications.append(f"| {row['replication']} | {row['nativeMS']/1000:.3f} | {row['candidateMS']/1000:.3f} | {row['meanSavedMS']/1000:.3f} | {row['savingPercent']:.2f}% | {row['positivePairs']}/3 | {'Pass' if row['passed'] else 'Fail'} |")
work = []
for row in phases['attempts']:
    if row['originalOrdinal'] == 17:
        continue
    work.append(f"| {row['replication']} | {row['originalOrdinal']} | {row['arm']} | {row['engineCheckedFiles']} | {sum(t['reusedFiles'] for t in row['adapterTasks'])} |")
cold = []
for row in profile['replicationDecisions']:
    cold.append(f"| {row['replication']} | {row['coldExcessMS']/1000:.3f} | {row['allFourRequestSavingMS']/1000:.3f} |")
passed = profile['materialSignal']
lead = ('The supported Checkstyle correction passed the short comparison in both replications. This supports the remaining readiness work before a longer validation. It does not yet establish sustained net savings.' if passed else
        'The supported Checkstyle correction did not pass the short comparison in both replications. This screen does not justify a longer validation under the current hypothesis. All builds and output checks completed; this is a measured screening result.')
next_step = ('Continue with the remaining BO-06 readiness and confirmation protocol. Formal G0/G3, preparation recovery and the protected validation history remain unqualified or unrun. A positive short screen does not authorize that replay.' if passed else
             'Proceed to the BO-12 decision for the current candidate set. No candidate is admitted to a long replay by this screen. Do not replace commits, relax thresholds or reopen retired routes. The incomplete workflow evidence for Micronaut remains unmeasured, not a failed optimization experiment.')
cost_lines = [f"| {name} | {seconds:.3f} |" for name,seconds in order['costSecondsByClassAndEnvelope'].items()]
text = f'''# Checkstyle on four consecutive changes

BO-05 · 14 September 2026 · **`{profile['decision']}`**.

{lead}

## What we compared

Checkstyle checks source formatting and coding rules. The correction remembers
successful checks on unchanged files while preserving complete reports. Both
arms ran `:server:precommit --continue`, including the same dependencies. N used
native Checkstyle; I used the supported V2 correction. Both retained the earlier
ForbiddenPatterns fix, so the measured difference concerns Checkstyle alone.

Two independent runs followed Elasticsearch changes 17–20. Change 17 started
from cold private state; changes 18–20 retained each arm's state. The command,
native cache policy, tools, eight-worker CPU profile and output rules were fixed.
Configuration Cache remained disabled for this workflow. The [manifest](./manifest.json)
records exact revisions and settings; the [protocol](./protocol.md) was fixed
before timing. No protected validation change was executed.

## Decision across all measured changes

Each replication had to save at least one second per measured pair and 5% in
total, with at least two positive pairs, actual native checking and candidate
reuse. Both replications had to pass. The inactive change remains included.

| Replication | Native total seconds | Corrected total seconds | Mean seconds saved | Saving | Positive pairs | Result |
|---|---:|---:|---:|---:|---:|---|
{chr(10).join(replications)}

These totals cover changes 18–20. Request time includes required machine work
recorded outside the native command. The [result](./result.json) retains each
delta and its component costs. The two replications are repeated runs of one
selected development window, not independent repositories or a precision study.

## Every build

N is native Checkstyle; I is the supported correction. No timing was removed,
replaced or repeated because it was unfavorable.

| Replication | Change | Role | Arm | Native seconds | Whole-request seconds | Host flag |
|---|---|---|---|---:|---:|---|
{chr(10).join(timings)}

The workstation was shared. [Host observations](./host-pressure.json) and
[order and idle intervals](./order-and-costs.json) retain all flags and coverage
limits. They do not identify the cause of an individual slow build. A flag does
not remove a build from the comparison.

## Work avoided and correctness

| Replication | Change | Arm | Files checked by the engine | Reused checks |
|---|---|---|---:|---:|
{chr(10).join(work)}

The [task records](./task-execution.json) retain cold builds, all task outcomes
and exact engine/adapter counts. Task durations are not added together to claim
elapsed savings. All eight live output comparisons and eight independent
reconstructions passed under the existing complete-output contract. All 16
builds succeeded. Source inventories verified that only the declared correction
differed; native Checkstyle acquired no candidate success state.

The [verification](./verification.json) also checks all frozen inputs, four
private worktrees, ownership cleanup and limits. Earlier lifecycle proof was
reused only after its sources and fixture records matched. Its Gradle version
and Configuration Cache limits remain unchanged.

## Cold and preparation costs

| Replication | Extra corrected cold seconds, if any | Seconds saved across all four requests, including cold |
|---|---:|---:|
{chr(10).join(cold)}

The following ledger totals preserve the runner's existing classification.
`customer-machine` names required machine work in that contract; it does not
refer to a customer trial. Inside-request rows are already counted in request
time and must not be added again. Research preparation is retained separately.

| Cost class and location | Seconds |
|---|---:|
{chr(10).join(cost_lines)}

These figures do not establish recovery of adoption or maintenance costs. The
screen used 16 owner builds and 16 comparator JVMs, with no retry or extra
comparison pass. State used {verification['allocatedBytes']/2**30:.2f} GiB within
the 64 GiB allowance. The allocation and all four owned service scopes closed.

## Candidate selection and next step

| Candidate | Current disposition |
|---|---|
| Elasticsearch Checkstyle | {'Short screen passed; remaining BO-06 work is next.' if passed else 'Short screen did not pass; no long replay admitted.'} |
| Micronaut Python task | Whole-workflow frequency and useful recurring cache restores remain unmeasured. BO-03 did not admit timing. |
| Spring architecture checks | The retained opportunity does not justify another timing allocation under the one-second floor. |
| Ktor and Apache Beam Build Impact | BO-04 admitted no distinct continuation of the selected-plan experiments. |

{next_step}

Follow the [governing tracker](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md).
The [evidence index](./evidence-index.json) identifies exact report copies and
retained local raw files. This export includes every timing row and its analysis;
full generated outputs and inventories remain local and are required to rerun
the complete output reconstruction. Product viability remains **NOT_ESTABLISHED**.
'''
(E/'README.md').write_text(text)
print(json.dumps(dict(exportedFiles=len(index)+2, exportRoot=str(E), decision=profile['decision'])))
