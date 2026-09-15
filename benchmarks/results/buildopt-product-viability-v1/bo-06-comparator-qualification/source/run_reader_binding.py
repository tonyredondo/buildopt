from common import *
import os, subprocess, time
reserve('goCommands','maxGoCommands')
command=['./dev/run','--toolchain','go','--','go','test','-count=1','-v','-tags=replay_integration,replay_owner','-run=^TestOwnerQualificationCannotBorrowDifferentReaders$',str(P.relative_to(R)/'tests/runner-source')]
command[-1]='./'+command[-1]
env=os.environ.copy(); env['BUILDOPT_REPLAY_OWNER_POLICY']=str(P/'inputs/measured-owner-policy.json'); env['GOMAXPROCS']='4'
start=time.monotonic_ns()
with (P/'logs/reader-binding.log').open('x') as log:
    result=subprocess.run(command,cwd=R,env=env,stdout=log,stderr=subprocess.STDOUT,timeout=300)
receipt=dict(command=command,exitCode=result.returncode,durationNS=time.monotonic_ns()-start,log=bind(P/'logs/reader-binding.log'),source=bind(P/'tests/runner-source'),ownerPolicy=bind(P/'inputs/measured-owner-policy.json'),ownerBuilds=0,fixtureWorkflows=0)
save(P/'receipts/reader-binding.json',receipt)
assert result.returncode==0
print('reader binding passed')
