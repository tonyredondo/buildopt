// Package githubqueue authenticates GitHub workflow_job webhooks and turns
// provider timestamps into durable, exact queue observations.
package githubqueue

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	WebhookPath      = "/internal/v1/ci/github/workflow-jobs"
	MaximumBodyBytes = 1 << 20
)

var (
	ErrConflict     = errors.New("GitHub workflow job conflicts with durable state")
	deliveryPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)
)

type Event struct {
	DeliveryID      string
	BodyDigest      string
	Action          string
	RepositoryID    int64
	RepositoryName  string
	JobID           int64
	RunID           int64
	RunAttempt      int64
	HeadSHA         string
	Name            string
	Status          string
	Conclusion      string
	CreatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	RunnerID        int64
	RunnerName      string
	RunnerGroupID   int64
	RunnerGroupName string
	Labels          []string
}

type PutResult int

const (
	PutAccepted PutResult = iota
	PutDuplicate
)

type Store interface {
	PutGitHubWorkflowJob(context.Context, Event) (PutResult, error)
}

type Handler struct {
	secret []byte
	store  Store
}

func NewHandler(secret []byte, store Store) (*Handler, error) {
	if len(secret) < 32 || store == nil {
		return nil, errors.New("invalid GitHub queue adapter configuration")
	}
	return &Handler{secret: append([]byte(nil), secret...), store: store}, nil
}

func (handler *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if request.URL.Path != WebhookPath {
		http.NotFound(response, request)
		return
	}
	if request.Header.Get("X-GitHub-Event") != "workflow_job" {
		http.Error(response, "unsupported GitHub event", http.StatusBadRequest)
		return
	}
	delivery := request.Header.Get("X-GitHub-Delivery")
	if !deliveryPattern.MatchString(delivery) {
		http.Error(response, "invalid GitHub delivery", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, MaximumBodyBytes))
	if err != nil {
		http.Error(response, "invalid webhook body", http.StatusRequestEntityTooLarge)
		return
	}
	if !validSignature(handler.secret, body, request.Header.Get("X-Hub-Signature-256")) {
		http.Error(response, "invalid webhook signature", http.StatusUnauthorized)
		return
	}
	event, err := decodeEvent(delivery, body)
	if err != nil {
		http.Error(response, "invalid workflow_job payload", http.StatusBadRequest)
		return
	}
	result, err := handler.store.PutGitHubWorkflowJob(request.Context(), event)
	if errors.Is(err, ErrConflict) {
		http.Error(response, "workflow_job conflict", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(response, "workflow_job persistence unavailable", http.StatusServiceUnavailable)
		return
	}
	if result == PutDuplicate {
		response.WriteHeader(http.StatusNoContent)
		return
	}
	response.WriteHeader(http.StatusAccepted)
}

func validSignature(secret, body []byte, signature string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) || len(signature) != len(prefix)+sha256.Size*2 {
		return false
	}
	want, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return hmac.Equal(mac.Sum(nil), want)
}

type webhookPayload struct {
	Action     string `json:"action"`
	Repository struct {
		ID       int64  `json:"id"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	WorkflowJob struct {
		ID              int64      `json:"id"`
		RunID           int64      `json:"run_id"`
		RunAttempt      int64      `json:"run_attempt"`
		HeadSHA         string     `json:"head_sha"`
		Name            string     `json:"name"`
		Status          string     `json:"status"`
		Conclusion      *string    `json:"conclusion"`
		CreatedAt       time.Time  `json:"created_at"`
		StartedAt       *time.Time `json:"started_at"`
		CompletedAt     *time.Time `json:"completed_at"`
		RunnerID        *int64     `json:"runner_id"`
		RunnerName      *string    `json:"runner_name"`
		RunnerGroupID   *int64     `json:"runner_group_id"`
		RunnerGroupName *string    `json:"runner_group_name"`
		Labels          []string   `json:"labels"`
	} `json:"workflow_job"`
}

func decodeEvent(delivery string, body []byte) (Event, error) {
	var payload webhookPayload
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&payload); err != nil {
		return Event{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return Event{}, err
	}
	job := payload.WorkflowJob
	if payload.Repository.ID <= 0 || payload.Repository.FullName == "" ||
		job.ID <= 0 || job.RunID <= 0 || job.RunAttempt <= 0 ||
		len(job.HeadSHA) != 40 || job.Name == "" || job.CreatedAt.IsZero() {
		return Event{}, errors.New("missing workflow job identity")
	}
	if !validLifecycle(payload.Action, job.Status) {
		return Event{}, errors.New("invalid workflow job lifecycle")
	}
	if job.StartedAt != nil && job.StartedAt.Before(job.CreatedAt) {
		return Event{}, errors.New("runner started before eligibility")
	}
	if job.CompletedAt != nil && (job.StartedAt == nil || job.CompletedAt.Before(*job.StartedAt)) {
		if job.StartedAt != nil || job.Status != "completed" {
			return Event{}, errors.New("invalid completion time")
		}
	}
	if (job.Status == "in_progress" || (job.Status == "completed" && job.StartedAt != nil)) && (job.StartedAt == nil || job.RunnerID == nil || job.RunnerGroupID == nil) {
		return Event{}, errors.New("in-progress job lacks runner assignment")
	}
	if job.Status == "completed" && job.CompletedAt == nil {
		return Event{}, errors.New("completed job lacks completion time")
	}
	digest := sha256.Sum256(body)
	event := Event{
		DeliveryID: delivery, BodyDigest: fmt.Sprintf("sha256:%x", digest),
		Action: payload.Action, RepositoryID: payload.Repository.ID,
		RepositoryName: payload.Repository.FullName, JobID: job.ID,
		RunID: job.RunID, RunAttempt: job.RunAttempt, HeadSHA: job.HeadSHA,
		Name: job.Name, Status: job.Status, CreatedAt: job.CreatedAt.UTC(),
		StartedAt: utc(job.StartedAt), CompletedAt: utc(job.CompletedAt),
		Labels: append([]string(nil), job.Labels...),
	}
	if job.Conclusion != nil {
		event.Conclusion = *job.Conclusion
	}
	if job.RunnerID != nil {
		event.RunnerID = *job.RunnerID
	}
	if job.RunnerName != nil {
		event.RunnerName = *job.RunnerName
	}
	if job.RunnerGroupID != nil {
		event.RunnerGroupID = *job.RunnerGroupID
	}
	if job.RunnerGroupName != nil {
		event.RunnerGroupName = *job.RunnerGroupName
	}
	return event, nil
}

func validLifecycle(action, status string) bool {
	return (action == "queued" && status == "queued") ||
		(action == "in_progress" && status == "in_progress") ||
		(action == "completed" && status == "completed")
}

func utc(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}
