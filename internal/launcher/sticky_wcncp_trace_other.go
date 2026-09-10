//go:build !linux

package launcher

// EIC's CLOCK_BOOTTIME diagnostic belongs to its Linux qualification only.
func beginStickyWCNCPTrace() func(childExecution) { return func(childExecution) {} }
