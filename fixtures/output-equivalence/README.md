# Output-equivalence contracts

These files are explicit owner-review inputs for the public workflow POC. They
contain no repository identity or task-selection logic. Exact bytes remain the
default for every required output that a rule does not match.

- `groovy-jar.json` compares canonical ZIP contents and permits only the
  `BuildTime` value in Groovy's release-properties entry to vary.
- `kafka-checkstyle.json` replaces only the isolated repository-root prefix in
  the UTF-8 Checkstyle reports; findings, paths below that root, and all other
  report bytes remain bound.
- `kafka-shadowjar.json` compares canonical ZIP entry names, modes, sizes, and
  uncompressed payloads while ignoring container order, timestamps, comments,
  extra fields, and compression encoding.

The conformance suite mutates non-declared payloads and requires rejection.
These contracts authorize POC measurement only, never automatic activation or
production use.
