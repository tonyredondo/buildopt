"""Exercise current comparator bytes with real readers; never run a build."""
from common import *
import importlib.util, os, subprocess, time, unittest

os.environ['BUILDOPT_REPLAY_TEST_ROOT'] = str(P / 'scratch/comparator-matrix')
os.environ['BUILDOPT_REPLAY_OWNER_POLICY'] = str(P / 'inputs/measured-owner-policy.json')
source = P / 'tests/runner-source'
spec = importlib.util.spec_from_file_location('retained_cases', source / 'owner_compare_test.py')
cases = importlib.util.module_from_spec(spec); spec.loader.exec_module(cases)
owner = cases.owner
policy = load(P / 'inputs/measured-owner-policy.json')
assert bind(source / 'owner_compare.py')['sha256'] == policy['comparator']['sha256']
real_compare = owner.compare
real_run = owner.subprocess.run
events = []
def counted_compare(request):
    reserve('comparatorReservations', 'maxComparatorCalls')
    return real_compare(request)
def observed_run(command, **kwargs):
    is_java = command[0] == policy['metadataJava']['path']
    if is_java: reserve('jvmReservations', 'maxComparatorJVMs')
    start = time.monotonic_ns()
    try:
        result = real_run(command, **kwargs)
    except Exception as error:
        events.append(dict(java=is_java, executable=command[0], startNS=start, endNS=time.monotonic_ns(), error=repr(error)))
        raise
    events.append(dict(java=is_java, executable=command[0], startNS=start, endNS=time.monotonic_ns(), exitCode=result.returncode))
    if is_java:
        a = allocation(); a['actualComparatorJVMs'] += 1; atomic(P / 'allocation.json', a)
    return result
owner.compare = counted_compare
owner.subprocess.run = observed_run

class ControlCases(cases.OwnerTests):
    def change_capture(self, end, change):
        value = owner.read(end['capture']['path']); change(value)
        end['capture'] = cases.save(Path(end['capture']['path']), value)
        cases.save(Path(end['capture']['path']).parent / 'end.json', end)

    def test_control_marker_is_boolean(self):
        request = self.adapted_pair()
        for value in (1, 'true'):
            with self.subTest(value=value):
                request['controlAdapted'] = value
                with self.assertRaisesRegex(ValueError, 'invalid control adapter declaration'): owner.compare(request)

    def test_control_generated_config_mode(self):
        request = self.adapted_pair()
        self.change_capture(request['native'], lambda c: next(o for o in c['outputs'] if o['entry']['path'] == owner.EXTRA)['entry'].update(mode=493))
        with self.assertRaisesRegex(ValueError, 'generated config mode differs'): owner.compare(request)

    def test_control_generated_config_producer(self):
        request = self.adapted_pair()
        self.change_capture(request['native'], lambda c: next(o for o in c['outputs'] if o['entry']['path'] == owner.EXTRA).update(producer=':server:checkstyleMain'))
        with self.assertRaisesRegex(ValueError, 'unapproved candidate-only output'): owner.compare(request)

    def test_control_task_dependencies(self):
        request = self.adapted_pair(); end = request['native']
        def change(capture):
            path = Path(capture['graph'][0]['path'])
            events = [owner.parse(line) for line in path.read_text().splitlines()]
            events[0]['tasks'][1]['dependencies'] = [owner.EXTRA_OWNER]
            path.write_text(''.join(json.dumps(e) + '\n' for e in events))
            capture['graph'] = [cases.binding(path)]
        self.change_capture(end, change)
        with self.assertRaisesRegex(ValueError, 'required task contract differs'): owner.compare(request)

    def test_control_missing_report(self):
        request = self.adapted_pair()
        self.change_capture(request['native'], lambda c: c.update(outputs=[o for o in c['outputs'] if o['entry']['path'] != 'server/build/reports/checkstyle/main.xml']))
        with self.assertRaisesRegex(ValueError, 'required output coverage differs'): owner.compare(request)

    def test_control_exit_difference(self):
        request = self.adapted_pair(); end = request['native']
        native = owner.read(end['native']['path']); native['exitCode'] = 1
        end['native'] = cases.save(Path(end['native']['path']), native)
        with self.assertRaisesRegex(ValueError, 'workflow exit differs'): owner.compare(request)

    def test_failed_controls_still_compare_reports(self):
        request = self.adapted_pair()
        for arm in ('native', 'candidate'):
            end = request[arm]; native = owner.read(end['native']['path']); native['exitCode'] = 1
            end['native'] = cases.save(Path(end['native']['path']), native)
        self.assertEqual(owner.compare(request)['status'], 'equivalent')
        self.mutate(request['native'], 'server/build/reports/checkstyle/main.xml', lambda raw: raw.replace(b'</file>', b'<error line="2" message="new finding"/></file>'))
        with self.assertRaisesRegex(ValueError, 'non-permitted projected difference'): owner.compare(request)

    def test_control_reuses_same_arm_only(self):
        self.adapted_pair()
        for ordinal, outcome in ((1, 'UP-TO-DATE'), (2, 'FROM-CACHE')):
            request = dict(policy=self.policy_binding, runRoot=str(self.root), native=self.arm('N', ordinal, outcome, control_adapted=True), candidate=self.arm('I', ordinal, outcome), controlAdapted=True)
            self.assertEqual(owner.compare(request)['counts']['causalOrigins'], 2)

suite = unittest.defaultTestLoader.loadTestsFromTestCase(cases.OwnerTests)
suite.addTests(ControlCases(name) for name in sorted(ControlCases.__dict__) if name.startswith('test_'))
start = time.monotonic_ns()
result = unittest.TextTestRunner(verbosity=2).run(suite)
save(P / 'receipts/comparator-matrix.json', dict(status='verified' if result.wasSuccessful() else 'failed', tests=result.testsRun, failures=len(result.failures), errors=len(result.errors), skipped=len(result.skipped), durationNS=time.monotonic_ns()-start, source=bind(source / 'owner_compare.py'), testsSource=bind(source / 'owner_compare_test.py'), additionalTests=bind(Path(__file__)), subprocesses=events, actualJavaStarts=sum(e['java'] for e in events), builds=0))
sys.exit(0 if result.wasSuccessful() else 1)
