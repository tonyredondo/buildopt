# Task-bound CPU isolation contract

The native CPU mask stays in the original manifest. Authenticated worker-config
adds nativeAffinity and observerAffinity; the supervisor is launched with
systemd CPUAffinity=8 for isolation, or the native mask for the shared control.
Both arms use one identical prototype executable. No launcher wrapper, native
argument, heap setting, worker-count change or production source change occurs.

The request goroutine locks its OS thread before changing affinity. Go's pinned
runtime starts its template thread while state is still unchanged. The exact
native mask is set and read back before Cmd.Start, then the original observer
mask is restored and read back immediately afterward. The thread stays locked
until child wait and observer closure, preserving Pdeathsig lifetime. All real
errors fail closed. A restoration failure kills and reaps the child, exits the
worker and leaves its unsafe thread locked so it cannot reenter the runtime.
Native completion is still stamped immediately after Wait before observer join.

Primary contracts inspected: pinned Go src/os/exec/exec.go:624-627,
src/runtime/proc.go:5607-5628 and src/syscall/exec_linux.go:92-96;
https://man7.org/linux/man-pages/man2/sched_setaffinity.2.html
(affinity per thread, inheritance through fork/exec, silently intersected masks).
The fixed owner compares native CPUs 0-7 with shared observer 0-7 versus
isolated observer CPU8, a different physical core (sibling24). Live guards,
100-ms cadence, memory/disk/timeout limits and native state remain unchanged.

The first compile rejected a shadowed append variable; no child or Gradle ran.
The corrected race proof verifies actual native child thread masks under
concurrent thread creation, observer restoration, launch and set errors,
restoration-error child reaping and disposal of the unsafe OS thread, and
cancellation. Three actual non-Gradle child starts; six reservations including
the initial compile failure. Existing liveness/disk race tests also pass.
