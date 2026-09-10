package wcncpobserve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Observer is shared by the standalone and installed entrypoints. Call Record
// only after the native child has finished. A failed upload preserves the outbox.
// The caller supplies an already scope-bound credential and redirect-safe client.
type Observer struct {
	RepositoryScope  string
	RouteScopeSHA256 string
	OutboxDir        string
	BackendURL       string
	Token            string
	Client           *http.Client
}

func (observer Observer) httpClient() *http.Client {
	if observer.Client != nil {
		return observer.Client
	}
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Record validates, queues, and attempts one upload with a 100ms network deadline.
// Fact collection and bounded local queue I/O are separate from that deadline.
func (observer Observer) Record(getenv func(string) string, cwd string, args []string, result PassthroughResult) error {
	if observer.RepositoryScope == "" || observer.OutboxDir == "" {
		return ErrObservationInvalid
	}
	redacted, err := RedactArguments(observer.RepositoryScope, args)
	if err != nil {
		return err
	}
	runnerID := loadOrCreateRunnerID(observer.OutboxDir)
	if runnerID == "" {
		return ErrObservationInvalid
	}
	facts := BuildObservation(observer.RepositoryScope, runnerID, getenv, cwd, args, redacted, result)
	if err := facts.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(facts)
	if err != nil {
		return err
	}
	outbox := Outbox{Dir: observer.OutboxDir}
	filename := fmt.Sprintf("obs-%d-%s.json", time.Now().UTC().UnixNano(), runnerID[:8])
	if err := outbox.Enqueue(filename, raw, time.Now().UTC()); err != nil {
		return err
	}
	if observer.BackendURL == "" || observer.Token == "" {
		return nil
	}
	endpoint, err := observer.endpoint("WCNCP_OBSERVATION/batch")
	if err != nil {
		return err
	}
	pending, err := outbox.Pending(32, 1<<20)
	if err != nil || len(pending) == 0 {
		return err
	}
	batch := make([]json.RawMessage, 0, len(pending))
	for _, item := range pending {
		batch = append(batch, json.RawMessage(item.Raw))
	}
	outcome := UploadBatch(context.Background(), observer.httpClient(), endpoint, observer.Token, batch, 100*time.Millisecond)
	if outcome.Uploaded == len(pending) {
		return outbox.Acknowledge(pending)
	}
	return nil
}

// Status reads local state and a bounded authenticated projection without writes.
func (observer Observer) Status() Status {
	queued := 0
	if observer.OutboxDir != "" {
		if items, err := (Outbox{Dir: observer.OutboxDir}).Pending(OutboxMaxItems, OutboxMaxBytes); err == nil {
			queued = len(items)
		}
	}
	state, verified := "", false
	if observer.BackendURL != "" && observer.RepositoryScope != "" && observer.Token != "" {
		if endpoint, err := observer.endpoint("status"); err == nil {
			state, verified = FetchStatus(context.Background(), observer.httpClient(), endpoint, observer.Token, 100*time.Millisecond)
		}
	}
	return DeriveStatus(queued, state, verified)
}

func (observer Observer) endpoint(resource string) (string, error) {
	if observer.RouteScopeSHA256 != "" {
		return EndpointForScope(observer.BackendURL, observer.RouteScopeSHA256, resource)
	}
	return Endpoint(observer.BackendURL, observer.RepositoryScope, resource)
}
