package wcncpobserve

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// UploadOutcome records whether a post-child batch reached the backend. The
// Gradle result is preserved regardless; an unavailable backend leaves items
// queued for a later invocation.
type UploadOutcome struct {
	Attempted bool
	Uploaded  int
	Queued    bool
	Reason    string
}

// UploadBatch attempts one bounded batch upload after the child completes
// under a strict post-child deadline. It never runs before the child and
// never delays the native result beyond the deadline.
func UploadBatch(ctx context.Context, client *http.Client, url, token string, items []json.RawMessage, timeout time.Duration) UploadOutcome {
	if len(items) == 0 {
		return UploadOutcome{Reason: "empty"}
	}
	if timeout <= 0 || timeout > PostChildUploadDeadlineMs*time.Millisecond {
		timeout = PostChildUploadDeadlineMs * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := json.Marshal(items)
	if err != nil {
		return UploadOutcome{Queued: true, Reason: "encode"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return UploadOutcome{Queued: true, Reason: "request"}
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return UploadOutcome{Queued: true, Reason: "unavailable"}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		// Corrupt or rejecting backends fail closed without losing the local
		// queue: the items stay queued for a later invocation.
		return UploadOutcome{Attempted: true, Queued: true, Reason: "rejected"}
	}
	return UploadOutcome{Attempted: true, Uploaded: len(items)}
}
