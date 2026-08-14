# Groovy console transition fingerprint

The first corrected Groovy capture on revision `bfb7578` completed all eight
pairs and the full fallback. Every pair was positive and semantically
equivalent, with a 54,295.75 ms / 72.91% mean saving. It did not qualify because
pair 7 alone recorded `:compileJava` first without an outcome suffix and then
as `FROM-CACHE`; the other seven pairs printed only the terminal line. The
initial correction preserved this render transition in the task fingerprint,
so it incorrectly appeared to be execution-shape drift.

This is not a performance failure and the timing is not promoted as terminal
evidence. The generic parser now retains ordered outcome transitions as
diagnostics, uses the last emission as the terminal task outcome, and computes
the execution fingerprint from terminal outcomes only. No task is omitted and
malformed or non-terminal evidence still fails closed.

All terminal captures restart from zero on the next correction revision. No
output rule, workflow, threshold, pairing, fallback, or POC boundary changed.
