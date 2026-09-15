"""Export complete control and interrupted development records; large output trees remain local."""
from pathlib import Path
import sys
sys.dont_write_bytecode=True
sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from common import *
import gzip,shutil,tarfile,io
close=load(P/'receipts/allocation-closeout.json');assert close['status']=='verified'
D=R/'benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-measurement';D.mkdir(exist_ok=True)
entries=[]
missing=[]
def copy(source,relative=None,compress=False):
    source=Path(source);relative=relative or str(source.relative_to(P))
    if compress:relative+='.gz'
    destination=D/relative;destination.parent.mkdir(parents=True,exist_ok=True)
    assert not destination.exists(),destination
    if compress:
        with source.open('rb') as src,destination.open('xb') as out:
            with gzip.GzipFile(filename='',mode='wb',fileobj=out,mtime=0) as zipped:shutil.copyfileobj(src,zipped)
    else:
        assert source.stat().st_size<8*2**20,source
        shutil.copyfile(source,destination)
    row=dict(export=relative,original=bind(source),sha256=bind(destination)['sha256'],bytes=destination.stat().st_size)
    if compress:row['compression']='gzip'
    entries.append(row)
for directory in ('inputs','receipts','analysis','logs'):
    for f in sorted((P/directory).iterdir()):
        if f.is_file():copy(f,compress=f.stat().st_size>=4*2**20)
for f in sorted(P.glob('*.py')):copy(f,'tools/'+f.name)
for f in sorted((P/'reporting').glob('*.py')):copy(f,'tools/reporting/'+f.name)
for trial in close['trials']:
    profile=P/'profiles'/trial['label'];run=profile/'run'
    copy(profile/'manifest.json');copy(profile/'worktree-identities.json')
    for name in ('manifest.json','result.json','run-start.json','ownership.json','checkpoint.json'):copy(run/name)
    for directory in ('pairs','costs','results','recorder-resources','lifecycle'):
        for f in sorted((run/directory).glob('*.json')):copy(f)
    for directory in sorted((run/'sessions').iterdir()):
        for name in ('worker-config.json','closed.json'):copy(directory/name)
    for directory in sorted((run/'attempts').iterdir()):
        for name in ('start.json','end.json','native-start.json','native-pid.json','native-finish.json','stdout.log','stderr.log','supervisor-cpu.json','quiet-start.json'):
            f=directory/name
            if f.exists():copy(f,compress=f.stat().st_size>=4*2**20)
            else:
                assert trial['label']=='prefix' and directory.name=='r1-008-g0-I'
                missing.append(dict(path=str(f.relative_to(P)), reason='Quiet-start refusal before native launch'))
        graph=directory/'graph.jsonl'
        if graph.exists():copy(graph,compress=True)
        else:
            assert trial['label']=='prefix' and directory.name=='r1-008-g0-I'
            missing.append(dict(path=str(graph.relative_to(P)), reason='Quiet-start refusal before native launch'))
    copy(profile/'process-samples.jsonl',compress=True)
m=load(P/'profiles/control/manifest.json');policy=load(m['outputs']['owner']['path'])
copy(m['quietStart']['path'],'inputs/quiet-start-policy.json')
copy(m['outputs']['owner']['path'],'inputs/measured-owner-policy.json')
copy(policy['comparator']['path'],'inputs/measured-owner-compare.py')
copy(policy['qualification']['path'],'inputs/owner-qualification.json')
source=Path(m['package'][0]['path']);members=[]
with tarfile.open(D/'runner-source.tar.gz','x:gz') as archive:
    for f in sorted(source.rglob('*')):
        assert not f.is_symlink()
        if not f.is_file():continue
        data=f.read_bytes();name=str(f.relative_to(source));info=tarfile.TarInfo(name);info.size=len(data);info.mode=f.stat().st_mode&0o777;info.mtime=0;archive.addfile(info,io.BytesIO(data))
        members.append(dict(path=name,sha256=hashlib.sha256(data).hexdigest(),bytes=len(data),mode=info.mode))
save(D/'source-archive-members.json',members)
for path in (D/'runner-source.tar.gz',D/'source-archive-members.json'):entries.append(dict(export=path.name,sha256=bind(path)['sha256'],bytes=path.stat().st_size))
index=dict(schema='buildopt.quiet-measurement/evidence/v1',decision=close['decision'],originalRoot=str(P),files=entries,missingAttemptFiles=missing,omitted='Complete output trees, inventories, caches and candidate state remain local. The portable audit checks exported records, bindings and arithmetic; replaying all output comparisons needs those retained inputs and consumes additional comparator starts.')
save(D/'evidence-manifest.json',index)
print(json.dumps(dict(files=len(entries),bytes=sum(e['bytes'] for e in entries))))
