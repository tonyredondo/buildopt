from common import *
policy=load(P/'inputs/measured-owner-policy.json')
matrix=load(P/'receipts/comparator-matrix.json')
assert matrix['tests']==33 and matrix['status']=='verified' and matrix['failures']==matrix['errors']==matrix['skipped']==0
assert load(P/'receipts/reader-binding.json')['exitCode']==0
retained=load(P/'analysis/retained-comparisons.json')
assert retained['status']=='verified' and len(retained['comparisons'])==6
for row in retained['comparisons']:
    assert row['report']['status']=='equivalent' and row['report']['policySHA256']==bind(P/'inputs/measured-owner-policy.json')['sha256']
identities=[policy['schema'],policy['kind'],policy['reusePolicy']]+[policy[k]['sha256'] for k in ['baseContract','dateContract','interpreter','comparator','lexicalProjector','metadataJava']]+[b['sha256'] for b in policy['metadataClasspath']]
readers=hashlib.sha256((json.dumps(identities,indent=2)+'\n').encode()).hexdigest()
q=dict(schema='buildopt.history-replay/owner-qualification/v1',decision='OWNER_OUTPUT_POLICY_QUALIFIED',comparatorSHA256=policy['comparator']['sha256'],readersSHA256=readers,cases=[bind(P/x) for x in ['receipts/comparator-matrix.json','logs/comparator-matrix.log','receipts/reader-binding.json','logs/reader-binding.log','receipts/retained-c5-comparison.json']],nativeReuse=[bind(P/'receipts'/f'{name}-comparison.json') for name in ['native-reuse-000','native-reuse-001','lean-control-0','lean-control-3','screen-native-corrected']])
save(P/'inputs/owner-qualification.json',q)
policy['qualification']=bind(P/'inputs/owner-qualification.json')
save(P/'inputs/qualified-owner-policy.json',policy)
print(json.dumps(dict(policy=bind(P/'inputs/qualified-owner-policy.json'),readersSHA256=readers)))
