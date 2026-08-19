# Economic prequalification POC contract

## Purpose

BuildOpt must not spend more time learning an optimization than the
optimization can plausibly return. This contract adds a cheap decision before
automatic structural discovery and eight-pair calibration when a verified
central graph exists but the currently qualified profile does not match the
changed project.

The decision is a POC learning gate, not a production policy and not a claim
that past changes predict future changes exactly.

## Inputs

The precheck may use only information already available before discovery:

- the exact Gradle workflow entrypoints;
- a previously verified graph whose Wrapper, executable, repository and build
  logic bindings still match;
- the current changed paths and their unambiguous owners in that graph; and
- at most 64 first-parent commits from local Git history.

It must not run Gradle, inspect outputs, borrow the measured saving or useful
lifetime of a different profile, or branch on a repository name.

## Conservative economic lower bound

Qualification requires eight paired observations. Each pair contains one
optimized-native control. Even if the candidate arm took zero milliseconds,
the calibration could not repay in fewer than eight matching builds because
the eight controls alone must be recovered.

Therefore the generic precheck requires all of the following before allowing
measurement:

1. the current owner and change family resolve without ambiguity;
2. the verified graph can omit at least one project;
3. the configured maximum break-even window is at least eight builds; and
4. at least eight commits in the bounded recent history have the same owner and
   change family without global/build-logic changes.

Failure or unavailable evidence means `REJECT`, not optimistic discovery.
Passing means only `MEASURE`: the existing output-equivalence, fallback,
eight-pair, positive-interval and break-even gates still decide qualification.

## Observable result

`buildopt optimize` includes `prequalification` in its result. It records the
decision, reason, evidence source, entrypoints, family, owners, graph shape,
history counts, theoretical minimum payback and decision duration. A rejection
sets discovery reason `ECONOMIC_PREQUALIFICATION_REJECTED`, emits no discovery
files and performs no calibration.

An exact existing profile bypasses this gate because its learning cost has
already been paid and its evidence is revalidated independently.

## POC boundaries

- This block validates a generic rejection mechanism; it does not claim a
  production-grade forecast of future change frequency.
- It does not weaken native fallback, output equivalence or profile bindings.
- It does not require a soak, a design partner or Test Optimization.
- A future POC may replace the conservative recurrence rule only with measured
  evidence that improves realized cumulative wall time.
