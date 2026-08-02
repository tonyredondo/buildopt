// Package sessioningest transports provisional build-session observations.
//
// The launcher sends one bounded strict record through an authenticated
// loopback HTTP boundary. Requests bind the active gateway generation and use
// the session identity as their idempotency key; identical replay is accepted
// while conflicting reuse is rejected. Transport failure is diagnostic and
// cannot replace the Gradle command's exit status.
package sessioningest
