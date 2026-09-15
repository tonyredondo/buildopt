# BO-06 disk accounting: what must still count

This correction addresses the repeated directory walk identified in the
[storage diagnosis](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/README.md).
It changes resource supervision, not the optimization, output capture, limits,
quiet-start rule or timing decision.

## Writers and accounting boundary

The limit counts the logical length of every regular-file directory entry
beneath the run directory. A sparse file counts by its length. Two hardlinks
count twice, as they did before. Symlinks are not followed. This is an artifact
limit, separate from the filesystem free-space floor.

| Writer | Paths and lifetime | Required treatment |
| --- | --- | --- |
| Native command and its reused processes | Each arm's repository, Gradle home, temporary state and daemon logs | Count changes throughout the request. A completed request does not make the arm immutable. |
| Worker | Attempt stdout/stderr and process receipts; session configuration and closure | Count active logs. Receipts written after exit remain covered by subsequent complete checks. |
| Outer recorder | Seed copies, source changes, state snapshots, costs, checkpoints, comparisons and captured outputs | Keep complete checks at the existing preparation and completion boundaries. Do not omit earlier results from the live total. |
| Output retention | `output-objects`, population directories and hardlinked attempt output copies | Every retained name counts. Native files are copied into the store before retained names share an inode. The native output is not itself a retained hardlink. |
| Diagnostic sampler | Profile-level process samples and candidate-state copies outside the run directory | Retain the separate allocation and diagnostic-byte limits. These paths were not included in the run-directory limit before this correction. |

The writer map comes from `runner.go`, `process_linux.go`, `capture.go`,
`files.go` and the frozen `run_trial.py`. Completed evidence is not assumed
immutable. File watches also detect a size change made through an external
hardlink to a retained inode.

## Changed live check

One observer belongs to one native request. Its first check counts the complete
tree and installs watches on directories and regular inodes. Later checks read
notifications and recount affected directories. Newly created directories are
counted and watched; deleted subtrees are removed from the total. The observer
keeps no file contents and shares no state between native requests.

A directory move, lost event queue, lost directory watch, changed mount or root,
watch limit, unsupported filesystem, or observation error invalidates this
count. The request then uses the previous complete scanner. A failed complete
scan or exceeded ceiling cancels the request. Fallback is recorded; it cannot
be reported as a successful reduction in supervision work.

The fast path supports local Btrfs, ext4, XFS and tmpfs without nested mounts.
Other filesystems retain complete scans. This restriction and the inode watches
address documented notification gaps in
[Linux filesystem notifications](https://man7.org/linux/man-pages/man7/inotify.7.html).
The observer tracks sizes, not content changes. The output checker still owns
content correctness.

The polling interval remains 100 ms, with at most one disk check in progress.
Parent liveness, free space and owned-process checks continue independently.
Cancellation stops directory enumeration between batches and files, then joins
the observer and closes its watches. Complete checks before and after native
execution remain in place. Both versions observe a changing tree; neither
promises an atomic filesystem snapshot or detection of every transient byte
between polls.

## Bounded comparison

The cost fixture has 16 or 32,768 retained files, grouped in directories of up
to 256 files, plus an active directory. Each case performs twenty checks while
growing an active file and creating/removing a temporary directory. Each size
runs old/new, then new/old. There are exactly eight measured cases.

CPU and elapsed time include all checks, notification setup, active mutations
and observer closure. Creating the common input tree is outside both versions.
The first check is also recorded separately. No quiet threshold or project
performance criterion applies to these component timings. The large case must
reduce total CPU in both repetitions. The small-case cost remains visible.

The fixture does not establish the cost on the full retained Elasticsearch
tree, the availability of enough watches there, or the frequency of fallback
under actual Gradle writes. The next control must record those facts before
interpreting an apparent build improvement.

## Storage observations

The external sampler now retains host device counters, pending dirty/writeback
memory, group I/O pressure and counters where readable, and CPU/I/O observations
for itself and the supplied recorder PID. Each read interval is timestamped.
Per-sample CPU cost and the sampler's whole-run resources are separate; the
latter include writing samples and the existing candidate-state capture.

Unavailable optional counters carry an explicit reason. They do not become
zero or establish a quiet build. The existing required pressure reads still
control admission. Device counters describe host activity; they cannot identify
which process caused it. No I/O controller, permission or system setting is
changed, and no unrelated process is inspected.
