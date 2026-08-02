// Package metricscatalog loads and validates BuildOpt's versioned metric catalog.
//
// The catalog fixes identity, unit, sign, aggregation, measurement method, and
// availability semantics before a producer may report a metric. Consumers use
// these definitions to avoid silently comparing incompatible measurements or
// treating UNAVAILABLE observations as zero.
package metricscatalog
