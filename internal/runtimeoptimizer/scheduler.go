// Package runtimeoptimizer owns proof-gated runtime action scheduling.
package runtimeoptimizer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

const schedulerSchemaVersion = 1

var (
	ErrConflict       = errors.New("validation request conflicts with durable state")
	ErrRepositoryBusy = errors.New("repository already has an active validation lease")
	ErrNotLeaseOwner  = errors.New("validation lease owner does not match")
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
)

type Request struct {
	RequestID               string `json:"requestId"`
	RepositoryID            string `json:"repositoryId"`
	ActionID                string `json:"actionId"`
	PolicyDigest            string `json:"policyDigest"`
	BaselineDigest          string `json:"baselineDigest"`
	WorkUnitsFingerprint    string `json:"workUnitsFingerprint"`
	Platform                string `json:"platform"`
	CacheCompatibilityClass string `json:"cacheCompatibilityClass"`
	ValidatesCache          bool   `json:"validatesCache"`
}

type Variant struct {
	Name           string `json:"name"`
	Workspace      string `json:"workspace"`
	Outputs        string `json:"outputs"`
	GradleUserHome string `json:"gradleUserHome"`
	CredentialPath string `json:"credentialPath"`
	ReadNamespace  string `json:"readNamespace"`
	WriteNamespace string `json:"writeNamespace,omitempty"`
	StableWrite    bool   `json:"stableWrite"`
	Authoritative  bool   `json:"authoritative"`
}

type Plan struct {
	AttemptID string  `json:"attemptId"`
	Candidate Variant `json:"candidate"`
	Control   Variant `json:"control"`
	Stable    Variant `json:"stable"`
}

type Entry struct {
	Request    Request    `json:"request"`
	Plan       Plan       `json:"plan"`
	State      string     `json:"state"`
	LeaseOwner string     `json:"leaseOwner,omitempty"`
	LeaseUntil *time.Time `json:"leaseUntil,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type schedulerState struct {
	SchemaVersion int              `json:"schemaVersion"`
	Requests      map[string]Entry `json:"requests"`
}

type Scheduler struct {
	mutex sync.Mutex
	root  string
	path  string
	now   func() time.Time
	state schedulerState
}

func OpenScheduler(root string, now func() time.Time) (*Scheduler, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || now == nil {
		return nil, errors.New("open validation scheduler: invalid configuration")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("open validation scheduler: unsafe state root")
	}
	scheduler := &Scheduler{root: root, path: filepath.Join(root, "scheduler.json"), now: now, state: schedulerState{SchemaVersion: schedulerSchemaVersion, Requests: map[string]Entry{}}}
	if err := scheduler.load(); err != nil {
		return nil, err
	}
	return scheduler, nil
}

func (scheduler *Scheduler) Schedule(request Request) (Entry, bool, error) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()
	if !validRequest(request) {
		return Entry{}, false, errors.New("schedule validation: invalid request")
	}
	if current, ok := scheduler.state.Requests[request.RequestID]; ok {
		if current.Request != request {
			return Entry{}, false, ErrConflict
		}
		return current, false, nil
	}
	plan := scheduler.plan(request)
	if err := provisionVariant(plan.Candidate); err != nil {
		return Entry{}, false, err
	}
	if err := provisionVariant(plan.Control); err != nil {
		return Entry{}, false, err
	}
	entry := Entry{Request: request, Plan: plan, State: "SCHEDULED", UpdatedAt: scheduler.now().UTC()}
	scheduler.state.Requests[request.RequestID] = entry
	if err := scheduler.persist(); err != nil {
		delete(scheduler.state.Requests, request.RequestID)
		return Entry{}, false, err
	}
	return entry, true, nil
}

func (scheduler *Scheduler) Lease(requestID, owner string, lifetime time.Duration) (Entry, error) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()
	if !identifierPattern.MatchString(requestID) || !identifierPattern.MatchString(owner) || lifetime <= 0 || lifetime > time.Hour {
		return Entry{}, errors.New("lease validation: invalid request")
	}
	now := scheduler.now().UTC()
	entry, ok := scheduler.state.Requests[requestID]
	if !ok {
		return Entry{}, os.ErrNotExist
	}
	original := entry
	if entry.State == "LEASED" && entry.LeaseUntil != nil && !entry.LeaseUntil.After(now) {
		entry.State, entry.LeaseOwner, entry.LeaseUntil = "SCHEDULED", "", nil
	}
	if entry.State == "LEASED" && entry.LeaseOwner == owner {
		return entry, nil
	}
	if entry.State != "SCHEDULED" {
		return Entry{}, ErrConflict
	}
	for id, other := range scheduler.state.Requests {
		if id == requestID || other.Request.RepositoryID != entry.Request.RepositoryID || other.State != "LEASED" || other.LeaseUntil == nil {
			continue
		}
		if other.LeaseUntil.After(now) {
			return Entry{}, ErrRepositoryBusy
		}
	}
	until := now.Add(lifetime)
	entry.State, entry.LeaseOwner, entry.LeaseUntil, entry.UpdatedAt = "LEASED", owner, &until, now
	scheduler.state.Requests[requestID] = entry
	if err := scheduler.persist(); err != nil {
		scheduler.state.Requests[requestID] = original
		return Entry{}, err
	}
	return entry, nil
}

func (scheduler *Scheduler) Finish(requestID, owner string, success bool) (Entry, error) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()
	entry, ok := scheduler.state.Requests[requestID]
	if !ok {
		return Entry{}, os.ErrNotExist
	}
	original := entry
	if entry.State != "LEASED" || entry.LeaseOwner != owner {
		return Entry{}, ErrNotLeaseOwner
	}
	if success {
		entry.State = "COMPLETED"
	} else {
		entry.State = "ABORTED"
	}
	entry.LeaseOwner, entry.LeaseUntil, entry.UpdatedAt = "", nil, scheduler.now().UTC()
	scheduler.state.Requests[requestID] = entry
	if err := scheduler.persist(); err != nil {
		scheduler.state.Requests[requestID] = original
		return Entry{}, err
	}
	return entry, nil
}

func (scheduler *Scheduler) plan(request Request) Plan {
	fingerprint := sha256.Sum256([]byte(request.RequestID + "\x00" + request.RepositoryID + "\x00" + request.ActionID + "\x00" + request.PolicyDigest + "\x00" + request.BaselineDigest + "\x00" + request.WorkUnitsFingerprint))
	attempt := hex.EncodeToString(fingerprint[:16])
	base := filepath.Join(scheduler.root, "requests", attempt)
	stable := "stable/" + request.Platform + "/" + request.CacheCompatibilityClass
	candidateRead := stable
	if request.ValidatesCache {
		candidateRead = "quarantine/" + request.ActionID + "/" + attempt
	}
	variant := func(name, read, write string, authoritative bool) Variant {
		root := filepath.Join(base, name)
		return Variant{Name: name, Workspace: filepath.Join(root, "workspace"), Outputs: filepath.Join(root, "outputs"), GradleUserHome: filepath.Join(root, "gradle-user-home"), CredentialPath: filepath.Join(root, "credential"), ReadNamespace: read, WriteNamespace: write, StableWrite: false, Authoritative: authoritative}
	}
	return Plan{AttemptID: attempt, Candidate: variant("candidate", candidateRead, "quarantine/"+request.ActionID+"/"+attempt, false), Control: variant("control", "control/action/"+request.PolicyDigest, "", true), Stable: Variant{Name: "stable", ReadNamespace: stable, WriteNamespace: stable, StableWrite: true, Authoritative: false}}
}

func provisionVariant(variant Variant) error {
	for _, path := range []string{variant.Workspace, variant.Outputs, variant.GradleUserHome} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("provision validation: unsafe private directory")
		}
	}
	credential, err := os.OpenFile(variant.CredentialPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		info, statErr := os.Lstat(variant.CredentialPath)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("provision validation: unsafe credential file")
		}
		return nil
	}
	if err != nil {
		return err
	}
	return credential.Close()
}

func validRequest(request Request) bool {
	return identifierPattern.MatchString(request.RequestID) && identifierPattern.MatchString(request.RepositoryID) && identifierPattern.MatchString(request.ActionID) && identifierPattern.MatchString(request.Platform) && identifierPattern.MatchString(request.CacheCompatibilityClass) && validDigest(request.PolicyDigest) && validDigest(request.BaselineDigest) && validDigest(request.WorkUnitsFingerprint)
}
func validDigest(value string) bool {
	if len(value) != 71 || value[:7] != "sha256:" {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}

func (scheduler *Scheduler) load() error {
	file, err := os.Open(scheduler.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() > 4<<20 {
		return errors.New("open validation scheduler: unsafe state file")
	}
	decoder := json.NewDecoder(io.LimitReader(file, 4<<20))
	decoder.DisallowUnknownFields()
	var state schedulerState
	if err := decoder.Decode(&state); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("open validation scheduler: trailing state")
	}
	if state.SchemaVersion != schedulerSchemaVersion || state.Requests == nil {
		return errors.New("open validation scheduler: unsupported state")
	}
	scheduler.state = state
	return nil
}

func (scheduler *Scheduler) persist() error {
	keys := make([]string, 0, len(scheduler.state.Requests))
	for key := range scheduler.state.Requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := schedulerState{SchemaVersion: schedulerSchemaVersion, Requests: make(map[string]Entry, len(keys))}
	for _, key := range keys {
		ordered.Requests[key] = scheduler.state.Requests[key]
	}
	data, err := json.Marshal(ordered)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(scheduler.root, ".scheduler-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err = file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err = file.Write(append(data, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(temporary)
		return closeErr
	}
	if err := os.Rename(temporary, scheduler.path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	directory, err := os.Open(scheduler.root)
	if err != nil {
		return err
	}
	err = directory.Sync()
	return errors.Join(err, directory.Close())
}

func (entry Entry) ValidateIsolation() error {
	paths := []string{entry.Plan.Candidate.Workspace, entry.Plan.Candidate.Outputs, entry.Plan.Candidate.GradleUserHome, entry.Plan.Candidate.CredentialPath, entry.Plan.Control.Workspace, entry.Plan.Control.Outputs, entry.Plan.Control.GradleUserHome, entry.Plan.Control.CredentialPath}
	for i := range paths {
		for j := i + 1; j < len(paths); j++ {
			if paths[i] == paths[j] {
				return fmt.Errorf("validation paths overlap: %s", paths[i])
			}
		}
	}
	if entry.Plan.Candidate.StableWrite || entry.Plan.Control.StableWrite || entry.Plan.Candidate.WriteNamespace == "" || entry.Plan.Control.WriteNamespace != "" || entry.Plan.Candidate.WriteNamespace == entry.Plan.Stable.WriteNamespace {
		return errors.New("validation namespace isolation is invalid")
	}
	return nil
}
