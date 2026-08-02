// Package neutralenvelope records and evaluates paired BuildOpt measurements.
//
// The neutral envelope binds native-baseline and optimization-off wrapper
// observations to the same runner, metric catalog, workload, revision, and
// artifact expectations. It also owns deterministic pilot assignment,
// observation, reporting, validation, and export models. Missing, divergent,
// or improperly paired evidence remains inconclusive rather than being
// attributed as saved build time.
package neutralenvelope
