package localauthority

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// StateContractVersion identifies the durable anti-rollback record.
const StateContractVersion = "buildopt.local-authority-state/v1"

// State is the minimum durable anti-rollback record. It deliberately stores
// authenticated generations and digests, never the data-plane credential.
type State struct {
	SchemaVersion               string             `json:"schemaVersion"`
	ScopeDigest                 string             `json:"scopeDigest"`
	Repository                  RepositoryIdentity `json:"repository"`
	PolicyID                    string             `json:"policyId"`
	PolicyVersion               int64              `json:"policyVersion"`
	PolicyDigest                string             `json:"policyDigest"`
	ConfigurationPolicyDigest   string             `json:"configurationPolicyDigest"`
	RevocationEpoch             int64              `json:"revocationEpoch"`
	RevocationDigest            string             `json:"revocationDigest"`
	L1SecurityGeneration        int64              `json:"l1SecurityGeneration"`
	GatewayConnectionGeneration int64              `json:"gatewayConnectionGeneration"`
	Namespace                   string             `json:"namespace"`
	NamespaceGeneration         int64              `json:"namespaceGeneration"`
	PolicyExpiresAt             string             `json:"policyExpiresAt"`
	InstalledAt                 string             `json:"installedAt"`
}

// StateFromVerified projects authenticated authority into its monotonic
// persistent form.
func StateFromVerified(verified Verified, installedAt time.Time) State {
	document := verified.document
	return State{
		SchemaVersion:               StateContractVersion,
		ScopeDigest:                 ScopeDigest(document.Repository),
		Repository:                  document.Repository,
		PolicyID:                    document.Policy.PolicyID,
		PolicyVersion:               document.Policy.PolicyVersion,
		PolicyDigest:                document.Policy.PolicyDigest,
		ConfigurationPolicyDigest:   document.Policy.ConfigurationPolicyDigest,
		RevocationEpoch:             document.Revocation.RevocationEpoch,
		RevocationDigest:            document.Revocation.CumulativeStateDigest,
		L1SecurityGeneration:        document.Revocation.L1SecurityGeneration,
		GatewayConnectionGeneration: document.Policy.GatewayConnectionGeneration,
		Namespace:                   document.Policy.RemoteCache.Namespace,
		NamespaceGeneration:         document.Policy.RemoteCache.NamespaceGeneration,
		PolicyExpiresAt:             document.Policy.ExpiresAt,
		InstalledAt: installedAt.UTC().
			Format(time.RFC3339Nano),
	}
}

// Advance validates a new authenticated state against the highest durable
// state. Exact policy/revocation replay is allowed; every security generation
// is otherwise monotonic, and a revocation epoch advance must rotate L1.
func Advance(current State, next State) error {
	if err := validateState(next); err != nil {
		return err
	}
	if current == (State{}) {
		return nil
	}
	if err := validateState(current); err != nil {
		return fmt.Errorf("%w: persisted state is invalid", ErrRollback)
	}
	if current.ScopeDigest != next.ScopeDigest ||
		current.Repository != next.Repository ||
		current.PolicyID != next.PolicyID {
		return fmt.Errorf("%w: policy scope changed", ErrRollback)
	}
	if next.PolicyVersion < current.PolicyVersion {
		return fmt.Errorf("%w: policy version decreased", ErrRollback)
	}
	if next.PolicyVersion == current.PolicyVersion &&
		(next.PolicyDigest != current.PolicyDigest ||
			next.ConfigurationPolicyDigest !=
				current.ConfigurationPolicyDigest) {
		return fmt.Errorf(
			"%w: policy version was reused with different content",
			ErrRollback,
		)
	}
	if next.RevocationEpoch < current.RevocationEpoch {
		return fmt.Errorf("%w: revocation epoch decreased", ErrRollback)
	}
	if next.RevocationEpoch == current.RevocationEpoch {
		if next.RevocationDigest != current.RevocationDigest ||
			next.L1SecurityGeneration != current.L1SecurityGeneration {
			return fmt.Errorf(
				"%w: revocation epoch was reused with different state",
				ErrRollback,
			)
		}
	} else if next.L1SecurityGeneration <=
		current.L1SecurityGeneration {
		return fmt.Errorf(
			"%w: revocation advance did not rotate L1",
			ErrRollback,
		)
	}
	if next.L1SecurityGeneration < current.L1SecurityGeneration {
		return fmt.Errorf("%w: L1 generation decreased", ErrRollback)
	}
	if next.GatewayConnectionGeneration <
		current.GatewayConnectionGeneration {
		return fmt.Errorf(
			"%w: gateway connection generation decreased",
			ErrRollback,
		)
	}
	if next.NamespaceGeneration < current.NamespaceGeneration {
		return fmt.Errorf(
			"%w: namespace generation decreased",
			ErrRollback,
		)
	}
	if next.NamespaceGeneration == current.NamespaceGeneration &&
		next.Namespace != current.Namespace {
		return fmt.Errorf(
			"%w: namespace generation was rebound",
			ErrRollback,
		)
	}
	return nil
}

// ScopeDigest creates an opaque stable path/database identity without treating
// the digest as authentication.
func ScopeDigest(repository RepositoryIdentity) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("buildopt-local-authority-scope-v1"))
	for _, value := range []string{
		repository.Tenant,
		repository.Repository,
		repository.TrustDomain,
	} {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(value)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(value))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func validateState(state State) error {
	if state.SchemaVersion != StateContractVersion ||
		len(state.ScopeDigest) != 64 ||
		!isLowerHex(state.ScopeDigest) ||
		state.ScopeDigest != ScopeDigest(state.Repository) ||
		!identifierPattern.MatchString(state.Repository.Tenant) ||
		!identifierPattern.MatchString(state.Repository.Repository) ||
		!identifierPattern.MatchString(state.Repository.TrustDomain) ||
		!identifierPattern.MatchString(state.PolicyID) ||
		state.PolicyVersion < 1 ||
		!validSHA256(state.PolicyDigest) ||
		!validSHA256(state.ConfigurationPolicyDigest) ||
		state.RevocationEpoch < 1 ||
		!validSHA256(state.RevocationDigest) ||
		state.L1SecurityGeneration < 1 ||
		state.GatewayConnectionGeneration < 1 ||
		!namespacePattern.MatchString(state.Namespace) ||
		state.NamespaceGeneration < 1 {
		return errors.New("invalid local authority state")
	}
	if _, err := parseTimestamp(state.PolicyExpiresAt); err != nil {
		return errors.New("invalid local authority policy expiry")
	}
	if _, err := parseTimestamp(state.InstalledAt); err != nil {
		return errors.New("invalid local authority install time")
	}
	return nil
}
