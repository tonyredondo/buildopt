package githubqueue

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type recordingStore struct{ events []Event }

func (store *recordingStore) PutGitHubWorkflowJob(_ context.Context, event Event) (PutResult, error) {
	store.events = append(store.events, event)
	return PutAccepted, nil
}

func TestHandlerAuthenticatesAndDecodesExactProviderFields(t *testing.T) {
	secret := []byte("github-webhook-secret-with-32-bytes-minimum")
	store := &recordingStore{}
	handler, err := NewHandler(secret, store)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"action":"completed","repository":{"id":7,"full_name":"owner/repo"},"workflow_job":{"id":101,"run_id":202,"run_attempt":1,"head_sha":"0123456789abcdef0123456789abcdef01234567","name":"build","status":"completed","conclusion":"success","created_at":"2026-07-31T08:00:00Z","started_at":"2026-07-31T08:00:45Z","completed_at":"2026-07-31T08:05:45Z","runner_id":11,"runner_name":"runner-11","runner_group_id":22,"runner_group_name":"linux-builders","labels":["self-hosted","linux","x64"]}}`
	request := httptest.NewRequest(http.MethodPost, WebhookPath, strings.NewReader(body))
	request.Header.Set("X-GitHub-Event", "workflow_job")
	request.Header.Set("X-GitHub-Delivery", "delivery-1")
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(body))
	request.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%x", mac.Sum(nil)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d", response.Code)
	}
	if len(store.events) != 1 || store.events[0].StartedAt.Sub(store.events[0].CreatedAt).Milliseconds() != 45_000 || store.events[0].RunnerGroupID != 22 {
		t.Fatalf("event = %+v", store.events)
	}

	request = httptest.NewRequest(http.MethodPost, WebhookPath, strings.NewReader(body))
	request.Header.Set("X-GitHub-Event", "workflow_job")
	request.Header.Set("X-GitHub-Delivery", "delivery-2")
	request.Header.Set("X-Hub-Signature-256", "sha256="+strings.Repeat("0", 64))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || len(store.events) != 1 {
		t.Fatalf("bad signature status/events = %d/%d", response.Code, len(store.events))
	}
}
