"""Exercise actual collector control flow and OS permission denial without Gradle."""
from pathlib import Path
import ast
import collections
import hashlib
import io
import json
import os
import subprocess
import sys
import tempfile
import textwrap
import time
import traceback

P = Path(__file__).resolve().parent
results = []

def extract(path, names, namespace):
    tree = ast.parse(path.read_text())
    nodes = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name in names]
    assert {node.name for node in nodes} == set(names)
    exec(compile(ast.Module(body=nodes, type_ignores=[]), str(path), 'exec'), namespace)

def check(name, action, expected=None):
    try:
        action()
    except Exception as error:
        assert expected and isinstance(error, expected), (name, repr(error))
        results.append(dict(name=name, status='passed', observed=type(error).__name__,
                            filename=getattr(error, 'filename', None)))
    else:
        assert expected is None, (name, 'expected failure did not occur')
        results.append(dict(name=name, status='passed'))

with tempfile.TemporaryDirectory(prefix='observer-repair-', dir=P/'receipts') as tmp:
    root = Path(tmp)
    cgroup = root/'fixture'; cgroup.mkdir()
    (cgroup/'cpu.stat').write_text('usage_usec 0\n')
    child = subprocess.Popen([sys.executable, '-B', '-c',
        'import ctypes,sys; libc=ctypes.CDLL(None,use_errno=True); assert libc.prctl(4,0,0,0,0)==0; print("ready",flush=True); input(); assert libc.prctl(4,1,0,0,0)==0; print("readable",flush=True); input()'],
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
    try:
        assert child.stdout.readline().strip() == 'ready'
        (cgroup/'cgroup.procs').write_text(str(child.pid))
        class Denied:
            def __init__(self, path, error): self.path, self.error = path, error
            def __str__(self): return str(self.path)
            def read_text(self): raise self.error(5 if self.error is OSError else 13, 'test access failure', str(self.path))
            def read_bytes(self): raise self.error(5 if self.error is OSError else 13, 'test access failure', str(self.path))
        def observe(version, denied=None, error=PermissionError):
            def path(first, *rest):
                p = Path(first, *rest)
                if p == Path('/sys/fs/cgroup'): return root
                if denied and p == Path('/proc', str(child.pid), denied): return Denied(p, error)
                return p
            ns = dict(Path=path, time=time, os=os, json=json, subprocess=subprocess,
                      run=root/'run', m={'affinity':'0-7'}, groups={'fixture':{'path':'/fixture'}},
                      samples=0, completed=set())
            names = ['stat','host_observation','observe']
            if version == 'v2': names.append('optional_process_text')
            extract(P/('run-screen-v2.py' if version=='v2' else 'run-screen.py'), names, ns)
            output = io.StringIO(); ns['observe'](output)
            return json.loads(output.getvalue())['groups'][0]['processes']
        check('original collector aborts on real Linux proc permission denial',
              lambda: observe('v1'), PermissionError)
        def gap():
            row, = observe('v2')
            assert row['identity']['pid']==child.pid and row['threadMasks']
            assert row['io'] is None
            assert any(x['path']==f'/proc/{child.pid}/io' and x['errno']==13 for x in row['unavailableObservations'])
        check('revised collector retains identity and affinity with explicit inaccessible IO', gap)
        child.stdin.write('\n'); child.stdin.flush()
        assert child.stdout.readline().strip() == 'readable'
        def normal():
            row, = observe('v2')
            assert row['io']['read_bytes'] >= 0 and not row['unavailableObservations']
        check('readable process retains normal IO counters', normal)
        def wait_gap():
            row, = observe('v2','wchan')
            assert row['mainThreadWaitChannel'] is None and row['io'] is not None
            assert row['unavailableObservations'][0]['path'].endswith('/wchan')
        check('inaccessible wait channel is an explicit optional gap', wait_gap)
        check('process identity access remains required',lambda:observe('v2','stat'),PermissionError)
        check('process command access remains required',lambda:observe('v2','cmdline'),PermissionError)
        check('other IO failures still abort collection',lambda:observe('v2','io',OSError),OSError)
        def disappeared(): assert observe('v2','io',FileNotFoundError)==[]
        check('process disappearance retains prior race handling', disappeared)
    finally:
        child.terminate(); child.wait(timeout=5)

source = (P/'analyze-phases-v2.py').read_text()
block = textwrap.dedent(source[source.index('    resources = {}'):source.index('    adapter_tasks = []')])
for missing in [(), (0,), (1,), (2,), (0,1,2)]:
    def resource_case(missing=missing):
        samples=[]
        for index in range(3):
            identity=dict(pid=1,startTicks=1,cpuTicks=index*100,majorFaults=index,state='S')
            process=dict(identity=identity,command='GradleDaemon',io=None if index in missing else dict(read_bytes=index*100,write_bytes=index*50))
            samples.append(dict(endBootNS=index*10**9,groups=[dict(unit='fixture',processes=[process])]))
        ns=dict(selected=samples,native={'unit':'fixture','start':{'ns':0}},hz=100,phases=[],collections=collections)
        exec(compile(block,'analyze-phases-v2 resource block','exec'),ns)
        value,=ns['resources'].values()
        assert value['observedCPUSeconds']==2 and value['missingIOSamples']==len(missing)
        assert value['observedReadBytes']==(None if missing else 200)
        assert value['observedWriteBytes']==(None if missing else 100)
        assert value['ioObservationComplete']==(not missing)
    check('resource accounting with missing IO at '+str(missing),resource_case)

controller=(P/'run-screen-v2.py').read_text()
a=controller.index("        receipt['observationFailureDetail']")
b=controller.index('        if child.poll()',a)
ns={'receipt':{},'traceback':traceback}
try:
    raise PermissionError(13,'test permission denial','/proc/example/io')
except PermissionError as error:
    ns['error']=error
    exec(textwrap.dedent(controller[a:b]),ns)
assert ns['receipt']['observationFailureDetail']['filename']=='/proc/example/io'
assert 'PermissionError' in ns['receipt']['observationFailureDetail']['traceback']
results.append(dict(name='fatal errors retain filename and traceback',status='passed'))
result=dict(status='verified focused observer repair',cases=results,passed=len(results),
    ownerBuilds=0,comparatorJVMs=0,measuredTrialWithRevisedCollector=False,
    limits=['Real OS permission denial reproduces the failure class; the original failing path and PID were not retained.',
            'Only optional IO and wait-channel access changes; critical identity, command and other failures still propagate.'],
    sources={name:hashlib.sha256((P/name).read_bytes()).hexdigest() for name in ['run-screen.py','run-screen-v2.py','analyze-phases-v2.py','test-observer-repair.py']})
out=P/'receipts/observer-repair-tests.json'
with out.open('x') as f: f.write(json.dumps(result,indent=2)+'\n')
print(json.dumps(result))
