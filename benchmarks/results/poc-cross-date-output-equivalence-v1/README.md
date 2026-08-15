# Reviewed cross-date output equivalence

This bundle closes the date-dependent output gap found by the public installed
profile replay. It is correctness and transfer evidence, not a new timing
experiment.

## Result

| Workflow and change | Cross-date evidence | Result | Historical qualification retained |
| --- | --- | --- | ---: |
| Groovy `jar`, leaf source | Real JAR plus controlled `BuildDate` change | match | 52,911.75 ms / 73.82% |
| Groovy `jar`, shared source | Real JAR plus controlled `BuildDate` change | match | 51,589 ms / 65.73% |
| Kafka Checkstyle, metadata source | Natural independent capture | match | 23,703.125 ms / 28.44% |
| Kafka Checkstyle, client-utils source | Natural independent capture | match | 28,356.125 ms / 31.68% |
| Kafka `shadowJar`, clients source | Natural independent capture | match | 29,638.125 ms / 66.40% |
| Kafka `shadowJar`, generator source | Natural independent capture | match | 29,121.625 ms / 79.15% |

All six cells are comparable across the relevant capture/date boundary. The
four Kafka rows preserve natural cross-capture matches from the public replay.
For each Groovy row, the runner built a real JAR from the frozen public
revision and deterministic source change, then copied it and changed only the
owner-declared `BuildDate`. The previous `BuildTime`-only contract rejected
both date changes; the reviewed `BuildDate + BuildTime` contract matched both.

The negative probes remain fail-closed. Changing undeclared
`ImplementationVersion` produced a different semantic digest in 2/2 Groovy
cells, as did changing a non-metadata `.class` payload. Original and mutated
JAR byte hashes also differ in every probe. Product-attributable failures are
zero.

## Interpretation

The Groovy date probe is intentionally controlled so it can be repeated on any
day. It demonstrates the exact semantic boundary on a real artifact; it is not
a claim that two performance builds were timed on separate dates. Prior
same-checkout native-versus-BuildOpt equivalence established that the date was
the remaining cross-capture difference.

No historical profile was edited. All six eight-pair qualification objects,
evidence digests, savings, intervals, and target revisions are copied from and
checked against their original terminal profiles. The reviewed fixture is the
default only for future qualification evidence.

Validate the checked bundle and its tamper tests with:

```bash
./dev/check-cross-date-output-equivalence-result
./dev/test-cross-date-output-equivalence-result
```

The checker binds the previous public replay and both contract digests,
recomputes each result from raw files, compares every historical qualification
to its original profile, verifies both build-log hashes, and rejects summary or
probe tampering. No new timing, averaged percentage, repository-name rule,
automatic or production authority, soak, design-partner dependency, or Test
Optimization scope is introduced.
