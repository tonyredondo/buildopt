import java.util.Arrays;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;
import org.gradle.internal.operations.DefaultBuildOperationRef;
import org.gradle.internal.operations.OperationIdentifier;
import org.gradle.internal.resources.DefaultResourceLockCoordinationService;
import org.gradle.internal.resources.ResourceLock;
import org.gradle.internal.work.AsyncWorkCompletion;
import org.gradle.internal.work.AsyncWorkTracker.ProjectLockRetention;
import org.gradle.internal.work.DefaultAsyncWorkTracker;
import org.gradle.internal.work.DefaultWorkerLeaseService;
import org.gradle.internal.work.ResourceLockStatistics;
import org.gradle.internal.work.WorkerLimits;
import org.gradle.util.GradleVersion;
import org.gradle.util.Path;

/** Controlled causality proof using the pinned Gradle services, no Gradle build or project edits. */
public final class LockRetentionProof {
    private static void await(CountDownLatch latch) {
        try {
            if (!latch.await(5, TimeUnit.SECONDS)) throw new AssertionError("Latch timed out");
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt(); throw new AssertionError(e);
        }
    }
    private static void check(ProjectLockRetention mode, boolean expectedEarlyReturn) throws Exception {
        var coordination = new DefaultResourceLockCoordinationService();
        var limits = new WorkerLimits() {
            public int getMaxWorkerCount() { return 4; }
            public int getMaxUnconstrainedWorkerCount() { return 4; }
        };
        var leases = new DefaultWorkerLeaseService(coordination, limits, ResourceLockStatistics.NO_OP);
        leases.startProjectExecution(true);
        ResourceLock projectLock = leases.getProjectLock(Path.ROOT, Path.path(":server"));
        var tracker = new DefaultAsyncWorkTracker(leases);
        var operation = new DefaultBuildOperationRef(new OperationIdentifier(1), null);
        var checkerHasLock = new CountDownLatch(1);
        var blockerHasLock = new CountDownLatch(1);
        var workFinished = new CountDownLatch(1);
        var releaseBlocker = new CountDownLatch(1);
        var complete = new AtomicBoolean();
        var earlyReturn = new AtomicBoolean();
        var failure = new AtomicReference<Throwable>();
        long[] stamps = new long[3];
        Thread checker = new Thread(() -> {
            try {
                leases.runAsWorkerThread((Runnable) () -> {
                    coordination.withStateLock(DefaultResourceLockCoordinationService.lock(projectLock));
                    try {
                        checkerHasLock.countDown();
                        tracker.registerWork(operation, new AsyncWorkCompletion() {
                            public boolean isComplete() { return complete.get(); }
                            public void cancel() { throw new AssertionError("Unexpected cancellation"); }
                            public void waitForCompletion() {
                                await(blockerHasLock);
                                complete.set(true);
                                stamps[0] = System.nanoTime();
                                workFinished.countDown();
                            }
                        });
                        tracker.waitForCompletion(operation, mode);
                        stamps[1] = System.nanoTime();
                        earlyReturn.set(releaseBlocker.getCount() == 1);
                    } finally {
                        coordination.withStateLock((Runnable) () -> {
                            if (projectLock.isLockedByCurrentThread()) projectLock.unlock();
                        });
                    }
                });
            } catch (Throwable e) { failure.compareAndSet(null, e); releaseBlocker.countDown(); workFinished.countDown(); }
        }, "proof-checker-" + mode);
        Thread blocker = new Thread(() -> {
            try {
                await(checkerHasLock);
                leases.runAsWorkerThread((Runnable) () -> leases.withLocks(List.of(projectLock), (Runnable) () -> {
                    blockerHasLock.countDown();
                    await(releaseBlocker);
                }));
            } catch (Throwable e) { failure.compareAndSet(null, e); releaseBlocker.countDown(); workFinished.countDown(); }
        }, "proof-project-lock-holder");
        checker.start(); blocker.start();
        await(workFinished);
        Thread.sleep(250);
        List<String> stack = Arrays.stream(checker.getStackTrace())
            .map(frame -> frame.getClassName() + "." + frame.getMethodName()).toList();
        stamps[2] = System.nanoTime();
        releaseBlocker.countDown();
        checker.join(5000); blocker.join(5000);
        if (checker.isAlive() || blocker.isAlive()) throw new AssertionError("Proof thread did not close");
        if (failure.get() != null) throw new AssertionError("Service proof failed", failure.get());
        if (!complete.get() || earlyReturn.get() != expectedEarlyReturn) throw new AssertionError("Wrong completion/lock ordering for " + mode);
        if (!expectedEarlyReturn && stack.stream().noneMatch(frame -> frame.contains("DefaultWorkerLeaseService"))) {
            throw new AssertionError("Missing actual Gradle lock-wait stack: " + stack);
        }
        if (expectedEarlyReturn && stamps[1] >= stamps[2]) throw new AssertionError("Native-style return waited for project release");
        if (!expectedEarlyReturn && stamps[1] < stamps[2]) throw new AssertionError("Final action returned before project release");
        System.out.println("LOCK_PROOF mode=" + mode + " returnedBeforeProjectRelease=" + earlyReturn.get()
            + " afterWorkNS=" + (stamps[1] - stamps[0]) + " heldAfterWorkNS=" + (stamps[2] - stamps[0]));
        for (String frame : stack) System.out.println("WAIT_FRAME " + frame);
        leases.finishProjectExecution(); leases.stop(); coordination.close();
    }
    public static void main(String[] args) {
        try {
            if (!GradleVersion.current().getVersion().equals("9.7.1")) throw new AssertionError("Wrong Gradle version");
            check(ProjectLockRetention.RELEASE_PROJECT_LOCKS, true);
            check(ProjectLockRetention.RELEASE_AND_REACQUIRE_PROJECT_LOCKS, false);
            System.out.println("LOCK_RETENTION_PROOF_VERIFIED Gradle=9.7.1 nativeBuildStarts=0 actualServices=true");
            System.exit(0);
        } catch (Throwable error) { error.printStackTrace(); System.exit(1); }
    }
}
