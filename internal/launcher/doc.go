// Package launcher owns the buildopt command's process and invocation boundary.
//
// It executes the original argv without a shell, preserves streams and child
// exit status, forwards cancellation to the complete process tree, and removes
// launcher-owned configuration from the child. Its optional managed path owns
// the authenticated local gateway, runner-slot lifecycle, native L1 lease,
// signed local-authority handoff, Gradle bootstrap cache, and bounded resource
// context.
//
// Optional setup may fail open only to the original command or another
// documented conservative path. Cache publication, authority validation, and
// cross-scope state reuse always fail closed.
package launcher
