package sharedcache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/tonyredondo/buildopt/internal/githubqueue"
)

// GitHubQueueObservation is the provider-authenticated queue measurement
// consumed by phase B. QueueMilliseconds is populated only for EXACT rows.
type GitHubQueueObservation struct {
	JobID             int64    `json:"jobId"`
	RunID             int64    `json:"runId"`
	RunAttempt        int64    `json:"runAttempt"`
	Availability      string   `json:"availability"`
	UnavailableReason string   `json:"unavailableReason,omitempty"`
	QueueMilliseconds int64    `json:"ciQueueMs,omitempty"`
	RunnerGroupID     int64    `json:"runnerGroupId,omitempty"`
	RunnerGroupName   string   `json:"runnerGroupName,omitempty"`
	RunnerID          int64    `json:"runnerId,omitempty"`
	RunnerName        string   `json:"runnerName,omitempty"`
	Labels            []string `json:"labels"`
}

func (storage *Storage) PutGitHubWorkflowJob(
	ctx context.Context,
	event githubqueue.Event,
) (githubqueue.PutResult, error) {
	if storage == nil {
		return 0, errors.New("persist GitHub workflow job: nil storage")
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return 0, err
	}
	defer finish()
	return putGitHubWorkflowJob(ctx, storage.control.database, event)
}

func (storage *Storage) GitHubQueueObservation(
	ctx context.Context,
	jobID int64,
) (GitHubQueueObservation, error) {
	if storage == nil {
		return GitHubQueueObservation{}, errors.New("read GitHub queue: nil storage")
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return GitHubQueueObservation{}, err
	}
	defer finish()
	return readGitHubQueueObservation(ctx, storage.control.database, jobID)
}

func putGitHubWorkflowJob(ctx context.Context, database *sql.DB, event githubqueue.Event) (githubqueue.PutResult, error) {
	if ctx == nil || database == nil || event.JobID <= 0 || event.DeliveryID == "" || event.BodyDigest == "" {
		return 0, errors.New("persist GitHub workflow job: invalid event")
	}
	labels := append([]string(nil), event.Labels...)
	slices.Sort(labels)
	labels = slices.Compact(labels)
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return 0, err
	}
	tx, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var existingDigest string
	err = tx.QueryRowContext(ctx, `SELECT body_digest FROM github_webhook_deliveries WHERE delivery_id = ?`, event.DeliveryID).Scan(&existingDigest)
	if err == nil {
		if existingDigest != event.BodyDigest {
			return 0, githubqueue.ErrConflict
		}
		return githubqueue.PutDuplicate, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	var existing githubJobRow
	err = scanGitHubJob(tx.QueryRowContext(ctx, `SELECT repository_id, repository_name, run_id, run_attempt, head_sha, name, status, conclusion, created_at_unix_ms, started_at_unix_ms, completed_at_unix_ms, runner_id, runner_name, runner_group_id, runner_group_name, labels_json FROM github_workflow_jobs WHERE job_id = ?`, event.JobID), &existing)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err == nil {
		if existing.repositoryID != event.RepositoryID || existing.repositoryName != event.RepositoryName || existing.runID != event.RunID || existing.runAttempt != event.RunAttempt || existing.headSHA != event.HeadSHA || existing.name != event.Name || existing.createdAt != event.CreatedAt.UnixMilli() {
			return 0, githubqueue.ErrConflict
		}
		if lifecycleRank(event.Status) < lifecycleRank(existing.status) {
			if !compatibleStale(existing, event) {
				return 0, githubqueue.ErrConflict
			}
			event.Status, event.Conclusion = existing.status, existing.conclusion
			event.StartedAt, event.CompletedAt = millisTime(existing.startedAt), millisTime(existing.completedAt)
			event.RunnerID, event.RunnerName = existing.runnerID.Int64, existing.runnerName.String
			event.RunnerGroupID, event.RunnerGroupName = existing.runnerGroupID.Int64, existing.runnerGroupName.String
			labelsJSON = []byte(existing.labelsJSON)
		} else if !compatibleAdvance(existing, event) {
			return 0, githubqueue.ErrConflict
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO github_workflow_jobs (job_id, repository_id, repository_name, run_id, run_attempt, head_sha, name, status, conclusion, created_at_unix_ms, started_at_unix_ms, completed_at_unix_ms, runner_id, runner_name, runner_group_id, runner_group_name, labels_json, updated_at_unix_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(job_id) DO UPDATE SET status=excluded.status, conclusion=excluded.conclusion, started_at_unix_ms=excluded.started_at_unix_ms, completed_at_unix_ms=excluded.completed_at_unix_ms, runner_id=excluded.runner_id, runner_name=excluded.runner_name, runner_group_id=excluded.runner_group_id, runner_group_name=excluded.runner_group_name, labels_json=excluded.labels_json, updated_at_unix_ms=excluded.updated_at_unix_ms`, event.JobID, event.RepositoryID, event.RepositoryName, event.RunID, event.RunAttempt, event.HeadSHA, event.Name, event.Status, event.Conclusion, event.CreatedAt.UnixMilli(), timeMillis(event.StartedAt), timeMillis(event.CompletedAt), nullablePositive(event.RunnerID), queueNullableString(event.RunnerName), nullablePositive(event.RunnerGroupID), queueNullableString(event.RunnerGroupName), string(labelsJSON), time.Now().UTC().UnixMilli()); err != nil {
		return 0, fmt.Errorf("persist GitHub workflow job: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO github_webhook_deliveries (delivery_id, body_digest, job_id, received_at_unix_ms) VALUES (?, ?, ?, ?)`, event.DeliveryID, event.BodyDigest, event.JobID, time.Now().UTC().UnixMilli()); err != nil {
		return 0, fmt.Errorf("persist GitHub delivery: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return githubqueue.PutAccepted, nil
}

type githubJobRow struct {
	repositoryID, runID, runAttempt, createdAt                    int64
	repositoryName, headSHA, name, status, conclusion, labelsJSON string
	startedAt, completedAt, runnerID, runnerGroupID               sql.NullInt64
	runnerName, runnerGroupName                                   sql.NullString
}

type rowScanner interface{ Scan(...any) error }

func scanGitHubJob(row rowScanner, value *githubJobRow) error {
	return row.Scan(&value.repositoryID, &value.repositoryName, &value.runID, &value.runAttempt, &value.headSHA, &value.name, &value.status, &value.conclusion, &value.createdAt, &value.startedAt, &value.completedAt, &value.runnerID, &value.runnerName, &value.runnerGroupID, &value.runnerGroupName, &value.labelsJSON)
}
func lifecycleRank(status string) int {
	if status == "queued" {
		return 1
	}
	if status == "in_progress" {
		return 2
	}
	if status == "completed" {
		return 3
	}
	return 0
}
func compatibleStale(existing githubJobRow, event githubqueue.Event) bool {
	return event.StartedAt == nil || !existing.startedAt.Valid || event.StartedAt.UnixMilli() == existing.startedAt.Int64
}
func compatibleAdvance(existing githubJobRow, event githubqueue.Event) bool {
	if existing.startedAt.Valid && (event.StartedAt == nil || event.StartedAt.UnixMilli() != existing.startedAt.Int64) {
		return false
	}
	if existing.completedAt.Valid && (event.CompletedAt == nil || event.CompletedAt.UnixMilli() != existing.completedAt.Int64) {
		return false
	}
	if existing.runnerID.Valid && event.RunnerID != existing.runnerID.Int64 {
		return false
	}
	if existing.runnerGroupID.Valid && event.RunnerGroupID != existing.runnerGroupID.Int64 {
		return false
	}
	if existing.status == "completed" && event.Conclusion != existing.conclusion {
		return false
	}
	return true
}
func timeMillis(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UnixMilli()
}
func millisTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	result := time.UnixMilli(value.Int64).UTC()
	return &result
}
func nullablePositive(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}
func queueNullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func readGitHubQueueObservation(ctx context.Context, database *sql.DB, jobID int64) (GitHubQueueObservation, error) {
	var observation GitHubQueueObservation
	var created int64
	var started, runnerGroupID, runnerID sql.NullInt64
	var runnerGroupName, runnerName sql.NullString
	var labelsJSON string
	err := database.QueryRowContext(ctx, `SELECT job_id, run_id, run_attempt, created_at_unix_ms, started_at_unix_ms, runner_group_id, runner_group_name, runner_id, runner_name, labels_json FROM github_workflow_jobs WHERE job_id = ?`, jobID).Scan(&observation.JobID, &observation.RunID, &observation.RunAttempt, &created, &started, &runnerGroupID, &runnerGroupName, &runnerID, &runnerName, &labelsJSON)
	if err != nil {
		return GitHubQueueObservation{}, err
	}
	observation.RunnerGroupID, observation.RunnerGroupName = runnerGroupID.Int64, runnerGroupName.String
	observation.RunnerID, observation.RunnerName = runnerID.Int64, runnerName.String
	if err := json.Unmarshal([]byte(labelsJSON), &observation.Labels); err != nil {
		return GitHubQueueObservation{}, err
	}
	if !started.Valid {
		observation.Availability = "UNAVAILABLE"
		observation.UnavailableReason = "CI_RUNNER_NOT_STARTED"
		return observation, nil
	}
	observation.Availability = "EXACT"
	observation.QueueMilliseconds = started.Int64 - created
	return observation, nil
}
