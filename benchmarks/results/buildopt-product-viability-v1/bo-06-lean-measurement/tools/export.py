"""Export compact evidence; retain complete owner outputs in local state."""
from common import *
import gzip, shutil, tarfile

D = R / 'benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement'
entries = []
def copy(source, relative=None):
    source = Path(source)
    relative = relative or str(source.relative_to(P))
    destination = D / relative
    assert source.stat().st_size < 4 * 2**20, source
    destination.parent.mkdir(parents=True, exist_ok=True)
    if destination.exists():
        assert source.read_bytes() == destination.read_bytes()
    else:
        shutil.copyfile(source, destination)
    entries.append(dict(export=relative, original=bind(source), sha256=bind(destination)['sha256'], bytes=destination.stat().st_size))

for folder in ('receipts', 'inputs', 'analysis'):
    for f in sorted((P / folder).iterdir()):
        if f.is_file(): copy(f)
for f in sorted(P.glob('*.py')): copy(f, 'tools/' + f.name)
for f in sorted((P / 'logs').iterdir()):
    if f.is_file(): copy(f)
for label in ('control', 'prefix'):
    copy(P / 'profiles' / label / 'manifest.json')
run = P / 'profiles/control/run'
for name in ('manifest.json', 'result.json'): copy(run / name)
for folder in ('pairs', 'costs', 'results'):
    for f in sorted((run / folder).glob('*.json')): copy(f)
for folder in (run / 'sessions').iterdir():
    for name in ('worker-config.json', 'closed.json'): copy(folder / name)
for folder in sorted((run / 'attempts').iterdir()):
    for name in ('start.json', 'end.json', 'native-start.json', 'native-pid.json', 'native-finish.json', 'stdout.log', 'stderr.log', 'supervisor-cpu.json'):
        copy(folder / name)
m = load(run / 'manifest.json')
copy(Path(m['outputs']['owner']['path']), 'inputs/measured-owner-policy.json')
policy = load(m['outputs']['owner']['path'])
copy(Path(policy['comparator']['path']), 'inputs/measured-owner-compare.py')
sample = P / 'profiles/control/process-samples.jsonl'
dest = D / 'profiles/control/process-samples.jsonl.gz'
with sample.open('rb') as source, dest.open('xb') as output:
    with gzip.GzipFile(filename='', mode='wb', fileobj=output, mtime=0) as compressed:
        shutil.copyfileobj(source, compressed)
entries.append(dict(export=str(dest.relative_to(D)), original=bind(sample), sha256=bind(dest)['sha256'], bytes=dest.stat().st_size, compression='gzip'))
members = []
with tarfile.open(D / 'runner-source.tar.gz') as archive:
    for member in archive:
        if not member.isfile(): continue
        content = archive.extractfile(member).read()
        members.append(dict(path=member.name, sha256=hashlib.sha256(content).hexdigest(), bytes=len(content)))
save(D / 'source-archive-members.json', members)
for relative in ('runner-source.tar.gz', 'runner-extension.patch', 'approved-method.md', 'source-archive-members.json'):
    path = D / relative
    entries.append(dict(export=relative, sha256=bind(path)['sha256'], bytes=path.stat().st_size))
index = dict(schema='buildopt.bo06-lean-evidence/v1', result='INCOMPLETE_PREFIX_NOT_ADMITTED',
    originalRoot=str(P), files=sorted(entries, key=lambda x: x['export']),
    supersededReports={'analysis/control.json': 'netSecondsPerScheduled has nanoseconds; use analysis/control-summary-v4.json'},
    omitted='Complete output trees, state inventories, caches and fixture scratch directories remain local. Full output reconstruction requires these files. The portable verifier checks exported records and arithmetic only.')
save(D / 'evidence-manifest.json', index)
print(json.dumps(dict(exportedFiles=len(entries), exportedBytes=sum(e['bytes'] for e in entries))))
