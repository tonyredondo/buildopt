// Package localauthority verifies and persists local cache authority.
//
// Authority combines canonical signed policy, cumulative revocation state,
// repository and component bindings, expiry, and monotonic generations. The
// package enforces private regular-file inputs and durable anti-rollback state;
// a credential, checksum, cache blob, or previously accepted document cannot
// independently authorize a current operation.
package localauthority
