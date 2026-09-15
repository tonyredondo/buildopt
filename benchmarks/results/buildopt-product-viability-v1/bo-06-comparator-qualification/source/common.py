from pathlib import Path
import datetime, hashlib, importlib.util, json, shutil, sys
sys.dont_write_bytecode = True
P = Path(__file__).resolve().parent
R = P.parents[3]
L = P.parent / 'bo-06-lean-measurement-2026-09-14'
spec = importlib.util.spec_from_file_location('retained_bindings', P.parent / 'bv006-cpu-isolation/bindings.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
bind = module.bind
def load(path): return json.loads(Path(path).read_text())
def save(path, value):
    path = Path(path); path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('x') as stream: json.dump(value, stream, indent=2); stream.write('\n')
def atomic(path, value):
    path = Path(path); temporary = path.with_suffix('.tmp')
    temporary.write_text(json.dumps(value, indent=2) + '\n'); temporary.replace(path)
def allocation():
    value = load(P / 'allocation.json')
    assert value['status'] == 'allocated'
    assert datetime.datetime.now(datetime.timezone.utc) < datetime.datetime.fromisoformat(value['deadlineUTC'])
    assert shutil.disk_usage(P).free >= value['minimumFreeBytes']
    return value
def reserve(kind, limit, count=1):
    value = allocation(); assert value[kind] + count <= value[limit], (kind, value[kind], count)
    value[kind] += count; atomic(P / 'allocation.json', value)
