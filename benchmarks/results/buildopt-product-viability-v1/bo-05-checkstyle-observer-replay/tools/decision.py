"""Prospective short-screen rule; this does not establish product value."""

def classify(pairs):
    assert len(pairs) == 8
    assert {(p['replication'], p['originalOrdinal']) for p in pairs} == {
        (r, o) for r in (1, 2) for o in (17, 18, 19, 20)}
    results = []
    for replication in (1, 2):
        rows = [p for p in pairs if p['replication'] == replication and p['originalOrdinal'] > 17]
        cold = next(p for p in pairs if p['replication'] == replication and p['originalOrdinal'] == 17)
        native = sum(p['N']['customerMS'] for p in rows)
        candidate = sum(p['I']['customerMS'] for p in rows)
        assert native > 0 and candidate > 0
        saving = native - candidate
        mean = saving / 3
        percent = 100 * saving / native
        positive = sum(p['N']['customerMS'] > p['I']['customerMS'] for p in rows)
        active = sum(task['action'] for p in rows for task in p['N']['checkstyle'])
        reusable = sum(state['inferredReusableFiles'] for p in rows for state in p['I']['candidateSuccessState'])
        passed = mean >= 1000 and percent >= 5 and positive >= 2 and active > 0 and reusable > 0
        results.append(dict(replication=replication, scheduledMeasuredPairs=3,
            nativeMS=native, candidateMS=candidate, meanSavedMS=mean, savingPercent=percent,
            positivePairs=positive, nativeCheckingActions=active, candidateReusableFiles=reusable,
            coldNativeMS=cold['N']['customerMS'], coldCandidateMS=cold['I']['customerMS'],
            coldExcessMS=max(0, cold['I']['customerMS']-cold['N']['customerMS']),
            allFourRequestSavingMS=saving+cold['N']['customerMS']-cold['I']['customerMS'],
            passed=passed))
    passed = all(r['passed'] for r in results)
    return dict(replications=results, materialSignal=passed,
        decision='EXPLORATORY_MATERIAL_SIGNAL' if passed else 'NO_MATERIAL_SIGNAL_IN_THIS_SCREEN')
