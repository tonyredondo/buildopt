package sessioningest

import (
	"errors"
	"reflect"
	"sort"
	"sync"
)

// ErrSessionConflict reports reuse of a session ID with different content.
var ErrSessionConflict = errors.New(
	"session ID was already accepted with different content",
)

// PutResult describes whether an ingest created or replayed a session.
type PutResult int

const (
	// PutCreated means the store accepted a new session.
	PutCreated PutResult = iota + 1
	// PutDuplicate means the store accepted an identical idempotent replay.
	PutDuplicate
)

// Store is the in-memory WS-005 ingest boundary. Durable control state remains
// separate; accepted records may be handed to the A0-008 JSON/JSONL exporter.
type Store struct {
	mutex    sync.RWMutex
	sessions map[string]Record
}

// NewStore creates an empty concurrency-safe in-memory ingest store.
func NewStore() *Store {
	return &Store{sessions: make(map[string]Record)}
}

// Put validates and idempotently stores a session record.
func (store *Store) Put(record Record) (PutResult, error) {
	if err := record.Validate(); err != nil {
		return 0, err
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()

	existing, found := store.sessions[record.SessionID]
	if found {
		if !reflect.DeepEqual(existing, record) {
			return 0, ErrSessionConflict
		}
		return PutDuplicate, nil
	}
	store.sessions[record.SessionID] = cloneRecord(record)
	return PutCreated, nil
}

// Snapshot returns the stored sessions ordered by session ID.
func (store *Store) Snapshot() []Record {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	result := make([]Record, 0, len(store.sessions))
	for _, record := range store.sessions {
		result = append(result, cloneRecord(record))
	}
	sort.Slice(result, func(left int, right int) bool {
		return result[left].SessionID < result[right].SessionID
	})
	return result
}

func cloneRecord(record Record) Record {
	cloned := record
	if record.ExportContext != nil {
		context := *record.ExportContext
		context.RequestedTasks = append(
			[]string(nil),
			record.ExportContext.RequestedTasks...,
		)
		cloned.ExportContext = &context
	}
	if record.GradleInvocation != nil {
		invocation := *record.GradleInvocation
		cloned.GradleInvocation = &invocation
	}
	return cloned
}
