package runtimeoptimizer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
)

// ConfigurationCachePhase is the proof-gated adoption state.
type ConfigurationCachePhase string

const (
	ConfigurationCacheDisabled          ConfigurationCachePhase = "DISABLED"
	ConfigurationCacheCanary            ConfigurationCachePhase = "CI_CANARY"
	ConfigurationCacheCompatibilityOnly ConfigurationCachePhase = "COMPATIBILITY_ONLY"
	ConfigurationCachePromoted          ConfigurationCachePhase = "PROMOTED"
	ConfigurationCacheSuspended         ConfigurationCachePhase = "SUSPENDED"
)

// ConfigurationCacheCompatibility is the pre-enable compatibility report.
type ConfigurationCacheCompatibility struct {
	RepositoryID              string
	PipelineClass             string
	GradleVersion             string
	ContractVersion           string
	ConfigurationPolicyDigest string
	Compatible                bool
	UnsuppressedProblems      int
	ProblemMode               string
	NoncriticalCI             bool
	PersistentWorkspace       bool
	PreservesEncryptionKey    bool
}

// ConfigurationCacheObservation records one idempotently identified natural result.
type ConfigurationCacheObservation struct {
	ObservationID       string
	Natural             bool
	EntryCreated        bool
	EntryReused         bool
	Correct             bool
	AttributableFailure bool
}

// ConfigurationCacheRollout retains server-side adoption evidence.
type ConfigurationCacheRollout struct {
	Phase                     ConfigurationCachePhase
	RepositoryID              string
	PipelineClass             string
	GradleVersion             string
	ContractVersion           string
	BasePolicyDigest          string
	ConfigurationPolicyDigest string
	RequiredCorrectHits       int
	CorrectNaturalHits        int
	InvalidationGeneration    uint64
	ExpectedSavings           bool
	SeenObservationIDs        map[string]struct{}
}

// ConfigurationCacheDecision is the complete distributable surface. It
// deliberately contains no entry bytes, workspace paths, or encryption keys.
type ConfigurationCacheDecision struct {
	Enabled                   bool   `json:"enabled"`
	ContractVersion           string `json:"contractVersion"`
	ConfigurationPolicyDigest string `json:"configurationPolicyDigest"`
	InvalidationGeneration    uint64 `json:"invalidationGeneration"`
}

// LocalConfigurationCacheIdentity binds an entry to one trusted local environment.
type LocalConfigurationCacheIdentity struct {
	Workspace          string
	HostID             string
	TrustDomain        string
	RepositoryID       string
	GradleVersion      string
	EncryptionStrategy string
}

// LocalConfigurationCache describes an in-place entry that is never distributed.
type LocalConfigurationCache struct {
	Directory      string
	IdentityDigest string
	Reusable       bool
}

// StartConfigurationCacheRollout applies the strict initial CI adoption gate.
func StartConfigurationCacheRollout(compatibility ConfigurationCacheCompatibility, requiredCorrectHits int) (ConfigurationCacheRollout, ConfigurationCacheDecision, error) {
	if !validConfigurationCacheCompatibility(compatibility) || requiredCorrectHits < 1 || requiredCorrectHits > 100 {
		return ConfigurationCacheRollout{}, ConfigurationCacheDecision{}, errors.New("start configuration cache rollout: invalid compatibility report")
	}
	rollout := ConfigurationCacheRollout{
		Phase:                  ConfigurationCacheDisabled,
		RepositoryID:           compatibility.RepositoryID,
		PipelineClass:          compatibility.PipelineClass,
		GradleVersion:          compatibility.GradleVersion,
		ContractVersion:        compatibility.ContractVersion,
		BasePolicyDigest:       compatibility.ConfigurationPolicyDigest,
		RequiredCorrectHits:    requiredCorrectHits,
		InvalidationGeneration: 1,
		SeenObservationIDs:     map[string]struct{}{},
	}
	if compatibility.Compatible && compatibility.NoncriticalCI {
		rollout.Phase = ConfigurationCacheCompatibilityOnly
		if compatibility.PersistentWorkspace && compatibility.PreservesEncryptionKey {
			rollout.Phase = ConfigurationCacheCanary
			rollout.ExpectedSavings = true
		}
	}
	rollout.ConfigurationPolicyDigest = configurationCacheDigest(rollout.BasePolicyDigest, rollout.enabled(), rollout.InvalidationGeneration)
	return rollout, rollout.Decision(), nil
}

// Observe applies one natural result idempotently and may promote or suspend.
func (rollout ConfigurationCacheRollout) Observe(observation ConfigurationCacheObservation) (ConfigurationCacheRollout, ConfigurationCacheDecision, error) {
	if err := rollout.validate(); err != nil || !validConfigurationCacheObservation(observation) {
		return ConfigurationCacheRollout{}, ConfigurationCacheDecision{}, errors.New("observe configuration cache: invalid state or observation")
	}
	if _, seen := rollout.SeenObservationIDs[observation.ObservationID]; seen {
		return rollout, rollout.Decision(), nil
	}
	seen := make(map[string]struct{}, len(rollout.SeenObservationIDs)+1)
	for id := range rollout.SeenObservationIDs {
		seen[id] = struct{}{}
	}
	seen[observation.ObservationID] = struct{}{}
	rollout.SeenObservationIDs = seen
	if rollout.Phase == ConfigurationCacheDisabled || rollout.Phase == ConfigurationCacheSuspended {
		return rollout, rollout.Decision(), nil
	}
	if observation.AttributableFailure {
		rollout.Phase = ConfigurationCacheSuspended
		rollout.CorrectNaturalHits = 0
		rollout.ExpectedSavings = false
		rollout.InvalidationGeneration++
		rollout.ConfigurationPolicyDigest = configurationCacheDigest(rollout.BasePolicyDigest, false, rollout.InvalidationGeneration)
		return rollout, rollout.Decision(), nil
	}
	if rollout.Phase == ConfigurationCacheCanary && observation.Natural && observation.EntryReused && observation.Correct {
		rollout.CorrectNaturalHits++
		if rollout.CorrectNaturalHits >= rollout.RequiredCorrectHits {
			rollout.Phase = ConfigurationCachePromoted
		}
	}
	return rollout, rollout.Decision(), nil
}

// Decision returns the entry-free policy surface safe to distribute.
func (rollout ConfigurationCacheRollout) Decision() ConfigurationCacheDecision {
	return ConfigurationCacheDecision{
		Enabled:                   rollout.enabled(),
		ContractVersion:           rollout.ContractVersion,
		ConfigurationPolicyDigest: rollout.ConfigurationPolicyDigest,
		InvalidationGeneration:    rollout.InvalidationGeneration,
	}
}

// MaterializeLocalConfigurationCache derives an in-place identity and directory.
func MaterializeLocalConfigurationCache(decision ConfigurationCacheDecision, identity LocalConfigurationCacheIdentity) (LocalConfigurationCache, error) {
	if !validDecision(decision) || !filepath.IsAbs(identity.Workspace) || filepath.Clean(identity.Workspace) != identity.Workspace ||
		!identifierPattern.MatchString(identity.HostID) || !identifierPattern.MatchString(identity.TrustDomain) ||
		!identifierPattern.MatchString(identity.RepositoryID) || !identifierPattern.MatchString(identity.GradleVersion) ||
		!identifierPattern.MatchString(identity.EncryptionStrategy) {
		return LocalConfigurationCache{}, errors.New("materialize configuration cache: invalid local identity")
	}
	encoded, _ := json.Marshal([]any{
		identity.Workspace,
		identity.HostID,
		identity.TrustDomain,
		identity.RepositoryID,
		identity.GradleVersion,
		identity.EncryptionStrategy,
		decision.ContractVersion,
		decision.ConfigurationPolicyDigest,
		decision.InvalidationGeneration,
	})
	digest := sha256.Sum256(encoded)
	return LocalConfigurationCache{
		Directory:      filepath.Join(identity.Workspace, ".gradle", "configuration-cache"),
		IdentityDigest: "sha256:" + hex.EncodeToString(digest[:]),
		Reusable:       decision.Enabled,
	}, nil
}

func (rollout ConfigurationCacheRollout) enabled() bool {
	return rollout.Phase == ConfigurationCacheCanary || rollout.Phase == ConfigurationCacheCompatibilityOnly || rollout.Phase == ConfigurationCachePromoted
}

func (rollout ConfigurationCacheRollout) validate() error {
	if !identifierPattern.MatchString(rollout.RepositoryID) || !identifierPattern.MatchString(rollout.PipelineClass) ||
		!identifierPattern.MatchString(rollout.GradleVersion) || !identifierPattern.MatchString(rollout.ContractVersion) ||
		!validDigest(rollout.BasePolicyDigest) || !validDigest(rollout.ConfigurationPolicyDigest) ||
		rollout.ConfigurationPolicyDigest != configurationCacheDigest(rollout.BasePolicyDigest, rollout.enabled(), rollout.InvalidationGeneration) ||
		rollout.RequiredCorrectHits < 1 || rollout.CorrectNaturalHits < 0 || rollout.InvalidationGeneration == 0 || rollout.SeenObservationIDs == nil {
		return errors.New("invalid configuration cache rollout")
	}
	for id := range rollout.SeenObservationIDs {
		if !identifierPattern.MatchString(id) {
			return errors.New("invalid configuration cache observation identity")
		}
	}
	switch rollout.Phase {
	case ConfigurationCacheDisabled, ConfigurationCacheCanary, ConfigurationCacheCompatibilityOnly, ConfigurationCachePromoted, ConfigurationCacheSuspended:
		return nil
	default:
		return errors.New("invalid configuration cache phase")
	}
}

func validConfigurationCacheCompatibility(compatibility ConfigurationCacheCompatibility) bool {
	return identifierPattern.MatchString(compatibility.RepositoryID) &&
		identifierPattern.MatchString(compatibility.PipelineClass) &&
		identifierPattern.MatchString(compatibility.GradleVersion) &&
		identifierPattern.MatchString(compatibility.ContractVersion) &&
		validDigest(compatibility.ConfigurationPolicyDigest) &&
		compatibility.UnsuppressedProblems >= 0 && compatibility.ProblemMode == "FAIL" &&
		(!compatibility.Compatible || compatibility.UnsuppressedProblems == 0)
}

func validConfigurationCacheObservation(observation ConfigurationCacheObservation) bool {
	if !identifierPattern.MatchString(observation.ObservationID) {
		return false
	}
	if observation.EntryCreated && observation.EntryReused || (observation.EntryCreated || observation.EntryReused) && !observation.Natural {
		return false
	}
	if observation.Correct && !observation.EntryReused || observation.EntryReused && !observation.Correct && !observation.AttributableFailure {
		return false
	}
	return true
}

func validDecision(decision ConfigurationCacheDecision) bool {
	return identifierPattern.MatchString(decision.ContractVersion) && validDigest(decision.ConfigurationPolicyDigest) && decision.InvalidationGeneration > 0
}

func configurationCacheDigest(base string, enabled bool, generation uint64) string {
	encoded, _ := json.Marshal([]any{base, enabled, generation})
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
