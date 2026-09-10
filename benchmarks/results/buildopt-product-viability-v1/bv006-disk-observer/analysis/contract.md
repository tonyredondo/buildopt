# Disk observation contract and correction

D1 verified on 2026-09-09 against the saved source and prior 751-file seal.
The live worker starts a 100-ms ticker. Its serial callback checks the driver,
free space, every regular-file logical size under RunRoot, then owned processes.
Long scans consume pending ticks; the implementation does not promise a hard
100-ms complete-size observation. It does require continued exact polling,
a fatal error above MaxBytes or below MinimumFreeBytes, and fatal root/I/O loss.
Removed temporary descendants are tolerated; links are not followed and hard
links count by pathname. Sparse files count logical size. Equality to MaxBytes
is admitted. The exact postflight check remains outside native completion.
Cancellation is checked between directory batches and entries. Native end is
stamped immediately after child wait, before the observer join. Existing owned
cgroup closure, parent identity, timeout and evidence contracts remain required.

The correction keeps full-tree scans and the original 100-ms scan ticker, with
one scan at a time. A separate 100-ms loop checks driver identity, free space
and process membership, so a scan cannot block those checks. Both loops are
joined and genuine errors survive child-exit races. Metadata uses fstatat
relative to the open directory instead of reconstructing every full path; all
regular files are still counted, with the same error and link rules.

A fixed consumer regression failed against the extracted original serial loop
and passed against the split implementation. Unit/race proof additionally
covers size growth, low free space, missing roots, process errors, driver loss,
real disk error at child completion and cancellation inside a large directory.
The original static profile attributes 84.13% of CPU samples to system calls;
whole scan CPU was 2.103436 seconds for 10,955,491,091 logical bytes. This supports
reducing metadata path work, not heap or JVM tuning.

A faster full scan may still saturate a CPU because the next tick is already
pending. The fixed owner control therefore reports CPU and native latency even
if they do not improve. No sampling interval, enforcement threshold, product
criterion or JVM setting is relaxed. This phase cannot qualify G0 or C5 value.
