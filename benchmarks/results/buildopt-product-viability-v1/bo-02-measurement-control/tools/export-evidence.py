"""Copy the bounded control report into the repository; no Git publication."""
from pathlib import Path
import hashlib
import json
import shutil

P = Path(__file__).resolve().parent
R = P.parents[3]
E = R/'benchmarks/results/buildopt-product-viability-v1/bo-02-measurement-control'

def load(path):
    return json.loads(path.read_text())

verification = load(P/'analysis/verification.json')
assert verification['status'] == 'verified'
profile = load(P/'analysis/profile-supported.json')
order = load(P/'analysis/order-and-idle-age.json')
phases = load(P/'analysis/diagnostic-phases.json')
E.mkdir()
mapping = {
    'analysis/profile-supported.json':'result.json',
    'analysis/verification.json':'verification.json',
    'analysis/host-pressure.json':'host-pressure.json',
    'analysis/diagnostic-phases.json':'task-execution.json',
    'analysis/order-and-idle-age.json':'order-and-idle-age.json',
    'profiles/supported/manifest.json':'manifest.json',
    'profiles/supported/run/result.json':'runner-result.json',
    'allocation.json':'allocation.json',
    'inputs/protocol.json':'protocol.json',
    'inputs/launch-freeze.json':'launch-freeze.json',
    'receipts/preparation.json':'receipts/preparation.json',
    'receipts/static-checks.json':'receipts/static-checks.json',
    'receipts/manifest-validation.json':'receipts/manifest-validation.json',
    'receipts/profile-supported-start.json':'receipts/run-start.json',
    'receipts/profile-supported-end.json':'receipts/run-end.json',
}
for name in ('run-control.py','analyze-control.py','analyze-host.py','analyze-phases.py','analyze-order.py'):
    mapping[name] = 'tools/'+name
    mapping['inputs/'+name+'.diff'] = 'tools/'+name+'.diff'
for name in ('prepare.py','verify-closeout.py','export-evidence.py'):
    mapping[name] = 'tools/'+name
assert sum((P/name).stat().st_size for name in mapping) < 8*2**20
index=[]
for source, target in mapping.items():
    src=P/source; dest=E/target
    dest.parent.mkdir(parents=True,exist_ok=True)
    shutil.copyfile(src,dest)
    digest=hashlib.sha256(src.read_bytes()).hexdigest()
    assert hashlib.sha256(dest.read_bytes()).hexdigest()==digest
    index.append(dict(path=target,source=str(src),bytes=src.stat().st_size,sha256=digest))
(E/'evidence-index.json').write_text(json.dumps(dict(status='verified exact copies',rawStateRoot=str(P),
    files=index,rawEvidenceRetainedLocally=True,
    portability='Reports and execution inputs are included. Full output inventories, retained files and process samples remain at their bound local paths; this export alone cannot rerun C5.'),indent=2)+'\n')

pair=next(pair for pair in profile['pairs'] if not pair['warmup'])
false=profile['materialSignal']
lead=('The same implementation produced a material timing difference. This control confirms that the preserved schedule can show an apparent improvement even when the compiled code is identical.' if false else
      'This control did not reproduce a material timing difference with identical code. The registered control is complete; measurement precision and optimization value remain unproven.')
next_step=('Keep value claims closed. Use the recorded intervals to propose the smallest change to the sequence, then prove output preservation, source checks and recovery before registering another control. The observed association does not by itself identify the cause. BO-03 and BO-04 may continue through evidence review without new timing runs.' if false else
           'Proceed to BO-03: estimate how much of a complete build each retained candidate could realistically save, including code changes where it does nothing. BO-04 can assess the focused Build Impact cases. A later short Checkstyle comparison still needs its own readiness decision and fixed inputs; this control does not start the 17–20 screen or the protected validation history.')
rows=[]
for entry in profile['pairs']:
    for arm in ('N','I'):
        value=entry[arm]
        rows.append(f"| {entry['originalOrdinal']} | {arm} | {'Cold' if entry['warmup'] else 'Measured'} | {value['nativeMS']/1000:.3f} | {value['customerMS']/1000:.3f} |")
orders=[]
for row in order['rows']:
    orders.append(f"| {row['measuredOrder']} | {row['arm']} | {row['idleSinceOwnColdNativeCompletionSeconds']:.1f} | {row['hostFlag']} |")
count_rows=[]
for row in phases['attempts']:
    if row['originalOrdinal']!=7: continue
    for task in row['adapterTasks']:
        count_rows.append(f"| {row['arm']} | {task['task']} | {task['inputFiles']} | {task['processedFiles']} | {task['reusedFiles']} |")
readme=f'''# Identical-code timing control

BO-02 completed on 14 September 2026. Decision: **`{profile['decision']}`**.

{lead}

## What ran

Four Elasticsearch builds replayed the same change from original commit ordinal 6
to ordinal 7. Both sides used the existing ForbiddenPatterns correction and the
same supported V2 Checkstyle correction. Checkstyle checks source formatting and
coding rules; the correction lets it reuse checks on unchanged files while
preserving complete reports.

The command was `:server:precommit --continue`, including its dependencies, with
Gradle's build cache enabled and Configuration Cache disabled. The pinned runner,
JDKs, instrumentation, output contract and schedule were reused unchanged. Native
work used CPUs 0–7, observation CPU 8 and the controller CPU 9. Each side had its
own working directory and cache. The [manifest](./manifest.json) records the full
command, source revisions and input hashes.

Both sides compiled identical effective source. The original runner installs the
N correction during preparation and applies the I correction inside its measured
request; that application cost remains included. These are arm labels, not an
unoptimized-versus-optimized comparison.

## Timings and decision

| Original ordinal | Arm | Role | Native seconds | Whole-request seconds |
| --- | --- | --- | ---: | ---: |
{chr(10).join(rows)}

The signed measured difference, N minus I, is **{profile['signedDifferenceMS']/1000:.3f}s**.
Its absolute size is **{profile['differencePercentOfFaster']:.2f}% of the faster
request**. The threshold fixed before execution was at least one second **and**
5% of the faster request. Cold builds remain in the report but do not enter that
decision. Whole-request time includes recorded work on the build machine outside
the native command where the original accounting requires it.

These timings describe the control. They provide no new estimate of optimization
savings and do not change the previous mixed finalization result.

## Same work and complete results

All four builds succeeded. Both output pairs passed the live comparison and the
independent reconstruction, using all four allowed comparator JVMs. The verifier
checked all tracked source files and the six effective patched files at both
revisions. Both arms produced matching complete outputs under the existing C5
contract. C5 is the retained check of declared files, reports, producers and
permitted normalization; it was not reduced for this control.

Actual Checkstyle work on the measured change was:

| Arm | Task | Input files | Checked again | Reused |
| --- | --- | ---: | ---: | ---: |
{chr(10).join(count_rows)}

The third Checkstyle task, `:server:checkstyleInternalClusterTest`, was already
up to date on both measured builds. Gradle skipped it. Its retained success
state was also verified.

The [verification](./verification.json) records source identity and run limits.
The [task records](./task-execution.json) preserve execution and reuse counts.

## Order and machine activity

| Measured order | Arm | Seconds since its own cold build ended | Host flag |
| --- | --- | ---: | --- |
{chr(10).join(orders)}

The idle intervals include file preservation, checks and the other arm's work.
They are not pure daemon sleep. [Host observations](./host-pressure.json) retain
every request and sampling limitation. CPU affinity does not isolate this shared
workstation; a pressure flag does not identify the cause of a slow build. No
sample was removed and no timing-driven retry was run.

## Limits and next step

One measured pair cannot establish precision, long-term savings or product
viability. G0 and G3 remain unqualified. The protected 21–100 history and other
repositories were not built.

The allocation closed with four owner starts, four comparator JVMs and
{verification['allocatedBytes']/2**30:.2f} GiB of local state, inside the registered
90-minute and 32 GiB limits. Both owned service scopes closed. No historical
allocation or result was overwritten. Changes remain local; no commit or push
was performed.

{next_step}

Follow the [governing tracker](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md)
and the [registered control protocol](../../../../docs/plans/buildopt-checkstyle-measurement-control-v1.md).
The [evidence index](./evidence-index.json) identifies exact copies and the retained
local raw evidence. The export includes reports, inputs and controller changes;
full binary outputs and process samples remain at their bound local paths.
'''
(E/'README.md').write_text(readme)
print(E)
