# Ordinary-build learning economics result

This deterministic POC evidence binds implementation commit
`806116dc254ac56b435ff27cacb63a1bbb465bb2` to a five-match payback horizon.
It proves decision economics and safety behavior; it is not a repository
performance benchmark.

## Result

- A 900 ms learning cost and 400 ms first-pair saving projects three matches
  to payback and 1,100 ms net value over the bounded five-match horizon.
- A four-match compatible lifetime stops after the baseline build and avoids
  the remaining 16 observations of an eight-pair calibration.
- A six-match projected payback stops after the first pair and avoids 14
  remaining observations.
- A regressive first pair also stops after three requested builds.
- A complete robust evidence sequence uses 17 requested ordinary builds:
  one baseline plus eight alternating pairs.
- Measurement-only builds are zero; unsafe measurement-only, structural-drift,
  output-drift and product-failure evidence is rejected in 4/4 cases.
- Product failures are zero.

The result authorizes bounded learning only. It does not automatically qualify
an optimization, weaken the robust eight-pair gate or claim production safety.

```bash
./dev/check-ordinary-learning-economics
```

The canonical evidence is [`summary.json`](./summary.json), and the normative
boundary is
[`specs/poc-ordinary-learning-economics-v1.md`](../../../specs/poc-ordinary-learning-economics-v1.md).
