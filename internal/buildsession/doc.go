// Package buildsession produces and persists BuildOpt build-session evidence.
//
// It maps authenticated launcher/session-ingest observations into the
// normative BUILD_SESSION v1 model, emits immutable private JSON documents,
// and owns the bounded deterministic JSONL lifecycle. Persistence is atomic,
// replay is idempotent, and startup recovery may create an explicit partial
// record but never fabricates missing observations.
//
// Schema validation remains outside this package so the producer cannot make
// its own output authoritative; repository conformance checks validate every
// emitted document against contracts/jsonschema/build-session.v1.schema.json.
package buildsession
