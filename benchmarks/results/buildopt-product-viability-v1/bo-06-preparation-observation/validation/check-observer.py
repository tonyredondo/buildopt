from pathlib import Path
import hashlib,json,os,shutil,subprocess
P=Path(__file__).resolve().parents[1]
rows=[]
for name,script,expected in [('success','printf \'{"fixture":true}\\n\'\n',0),('timeout','sleep 30\n',1)]:
 root=P/'validation'/('observer-'+name);root.mkdir()
 for folder in ('tools','inputs','receipts','logs'): (root/folder).mkdir()
 for source in (P/'tools').glob('*.py'):shutil.copyfile(source,root/'tools'/source.name)
 binary=root/'preparation-observer';binary.write_text('#!/bin/sh\n'+script);binary.chmod(0o700)
 shutil.copyfile(P/'inputs/original-manifest.json',root/'inputs/original-manifest.json')
 (root/'task-state.json').write_text(json.dumps({'steps':{}}))
 protocol={'baselineSeconds':1,'recoverySeconds':1,'maxObservationSeconds':5,'observerCPU':9,'minimumFreeBytes':40*2**30,'maxFreeSpaceDecreaseBytes':16*2**30,'maxDiagnosticBytes':2**20}
 (root/'inputs/observation-protocol.json').write_text(json.dumps(protocol))
 files=[{'path':str(f.relative_to(root)),'sha256':hashlib.sha256(f.read_bytes()).hexdigest()} for f in sorted(root.rglob('*')) if f.is_file()]
 (root/'inputs/diagnostic-freeze.json').write_text(json.dumps({'files':files}))
 result=subprocess.run(['python3','-B',str(root/'tools/observe_preparation.py')],text=True,capture_output=True,timeout=20)
 (root/'output.txt').write_text(result.stdout+result.stderr)
 assert result.returncode==expected,(name,result.stdout,result.stderr)
 end=json.loads((root/'receipts/end.json').read_text())
 assert end['status']==('COMPLETE' if name=='success' else 'INCOMPLETE')
 child=json.loads((root/'receipts/preparation-start.json').read_text())['process']
 assert not Path('/proc',str(child['pid'])).exists()
 try: os.killpg(child['pid'],0)
 except ProcessLookupError: group_gone=True
 else: group_gone=False
 assert group_gone
 rows.append({'case':name,'exitCode':result.returncode,'status':end['status'],'error':end['error'],'processGroupGone':group_gone})
print(json.dumps(rows,indent=2))
(P/'validation/observer-checks.json').write_text(json.dumps(rows,indent=2)+'\n')
