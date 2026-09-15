from pathlib import Path
import hashlib,importlib.util,json,sys
sys.dont_write_bytecode=True
P=Path(__file__).resolve().parent;R=P.parents[3]
spec=importlib.util.spec_from_file_location('prior_bindings',P.parent/'bv006-cpu-isolation/bindings.py');binding_module=importlib.util.module_from_spec(spec);spec.loader.exec_module(binding_module)
bind=binding_module.bind
def load(path):return json.loads(Path(path).read_text())
def save(path,x):
 path=Path(path);path.parent.mkdir(parents=True,exist_ok=True)
 with path.open('x') as f:json.dump(x,f,indent=2);f.write('\n')
def atomic(path,x):
 path=Path(path);tmp=path.with_suffix('.tmp');tmp.write_text(json.dumps(x,indent=2)+'\n');tmp.replace(path)
def bytes_used(root):
 return sum(f.stat().st_size for f in Path(root).rglob('*') if f.is_file() and not f.is_symlink())
