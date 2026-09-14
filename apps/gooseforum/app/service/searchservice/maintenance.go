package searchservice

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/meilisearch/meilisearch-go"
)

type MaintenanceRequest struct {
	Index  string `json:"index" validate:"required,oneof=all topics users categories courses wiki_pages"`
	Action string `json:"action" validate:"required,oneof=check rebuild"`
}

// MaintenanceProgress also provides the durable audit trail: actor, action,
// index selection, creation time on task_queue, and sanitized results.
type MaintenanceProgress struct {
	MaintenanceRequest
	RequestedBy  uint64       `json:"requestedBy"`
	Phase        string       `json:"phase"`
	CurrentIndex string       `json:"currentIndex"`
	Processed    int          `json:"processed"`
	Reports      []IndexCheck `json:"reports"`
}

type MaintenancePayload struct {
	MaintenanceProgress
	Instance string `json:"instance"`
}

func maintenanceInstance() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(preferences.Get("server.url"))))
}

func maintenanceEnabled() bool { return preferences.GetBool("meilisearch.maintenance_enabled", false) }

type MaintenanceJob struct {
	ID         uint64    `json:"id"`
	Status     uint8     `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	RetryCount uint8     `json:"retryCount"`
	ErrorCode  string    `json:"errorCode"`
	MaintenanceProgress
}

type MaintenanceSubmission struct {
	Job     MaintenanceJob `json:"job"`
	Created bool           `json:"created"`
}

type ManagedIndexStatus struct {
	Index           string      `json:"index"`
	ExpectedVersion int         `json:"expectedVersion"`
	Exists          bool        `json:"exists"`
	Documents       int64       `json:"documents"`
	Indexing        bool        `json:"indexing"`
	LastCheck       *IndexCheck `json:"lastCheck"`
}

type MaintenanceStatus struct {
	MaintenanceEnabled bool                 `json:"maintenanceEnabled"`
	Configured         bool                 `json:"configured"`
	Available          bool                 `json:"available"`
	EngineVersion      string               `json:"engineVersion"`
	Indexes            []ManagedIndexStatus `json:"indexes"`
	Jobs               []MaintenanceJob     `json:"jobs"`
}

func maintenanceJob(row taskQueue.Entity) MaintenanceJob {
	job := MaintenanceJob{ID: row.Id, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.ProcessedAt, RetryCount: row.RetryCount}
	var payload MaintenancePayload
	_ = json.Unmarshal([]byte(row.TaskJson), &payload)
	job.MaintenanceProgress = payload.MaintenanceProgress
	if job.Reports == nil {
		job.Reports = []IndexCheck{}
	}
	if row.Status == taskQueue.StatusFailed || row.Status == taskQueue.StatusRetrying {
		job.ErrorCode = "operation_failed"
	}
	// LastError can contain HTTP addresses or document content; never expose it.
	return job
}

func CreateIndexMaintenance(ctx context.Context, request MaintenanceRequest, userID uint64) (*MaintenanceSubmission, error) {
	if (request.Index != "all" && !managedIndex(request.Index)) || (request.Action != "check" && request.Action != "rebuild") {
		return nil, errors.New("invalid maintenance request")
	}
	if !maintenanceEnabled() || meiliconnect.GetClient() == nil {
		return nil, ErrSearchUnavailable
	}
	payload := MaintenancePayload{MaintenanceProgress: MaintenanceProgress{MaintenanceRequest: request, RequestedBy: userID, Phase: "queued", Reports: []IndexCheck{}}, Instance: maintenanceInstance()}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	row, created, err := taskQueue.CreateSearchMaintenance(ctx, string(raw))
	if err != nil {
		return nil, err
	}
	return &MaintenanceSubmission{Job: maintenanceJob(row), Created: created}, nil
}

func GetIndexMaintenanceStatus(ctx context.Context) (*MaintenanceStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := taskQueue.ListSearchMaintenance(ctx)
	if err != nil {
		return nil, err
	}
	status := &MaintenanceStatus{MaintenanceEnabled: maintenanceEnabled(), Jobs: []MaintenanceJob{}, Indexes: []ManagedIndexStatus{}}
	for _, row := range rows {
		var payload MaintenancePayload
		if json.Unmarshal([]byte(row.TaskJson), &payload) != nil || payload.Instance != maintenanceInstance() {
			continue
		}
		status.Jobs = append(status.Jobs, maintenanceJob(row))
	}
	client := meiliconnect.GetClient()
	status.Configured = client != nil
	var stats *meilisearch.Stats
	if client != nil {
		if version, err := client.VersionWithContext(ctx); err == nil {
			status.EngineVersion = version.PkgVersion
			stats, err = client.GetStatsWithContext(ctx, &meilisearch.StatsParams{})
			status.Available = err == nil
		}
	}
	for _, name := range managedIndexes {
		item := ManagedIndexStatus{Index: name, ExpectedVersion: expectedProjectionVersion(name)}
		if stats != nil {
			if live, ok := stats.Indexes[name]; ok {
				item.Exists = true
				item.Documents = live.NumberOfDocuments
				item.Indexing = live.IsIndexing
			}
		}
		for _, job := range status.Jobs {
			for _, report := range job.Reports {
				if report.Index == name {
					copy := report
					item.LastCheck = &copy
					break
				}
			}
			if item.LastCheck != nil {
				break
			}
		}
		status.Indexes = append(status.Indexes, item)
	}
	return status, nil
}

func RunIndexMaintenance(ctx context.Context, task *taskQueue.Entity) error {
	client := meiliconnect.GetClient()
	return runIndexMaintenance(ctx, client, task)
}

func runIndexMaintenance(ctx context.Context, client meilisearch.ServiceManager, task *taskQueue.Entity) error {
	if task == nil || task.Type != taskQueue.SearchMaintenanceType {
		return errors.New("invalid maintenance task")
	}
	var payload MaintenancePayload
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return err
	}
	if (payload.Index != "all" && !managedIndex(payload.Index)) || (payload.Action != "check" && payload.Action != "rebuild") {
		return errors.New("invalid maintenance payload")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	// Retries restart the bounded reconciliation. Never reuse a partial scan or
	// skip previously written pages: both DB and index may have changed meanwhile.
	payload.Reports = []IndexCheck{}
	progress := func(phase, index string, processed int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Verify ownership at every batch, in addition to the worker heartbeat.
		ok, _, _, err := taskQueue.RenewLease(task.Id, task.LeaseToken)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("maintenance lease lost")
		}
		payload.Phase, payload.CurrentIndex, payload.Processed = phase, index, processed
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return taskQueue.UpdateTaskJsonOwned(task.Id, task.LeaseToken, string(raw))
	}
	names := managedIndexes
	if payload.Index != "all" {
		names = []string{payload.Index}
	}
	// Dev deployments copy main's task table and share its engine. Bind jobs to
	// their configured origin and require explicit maintenance opt-in; copied
	// tasks are consumed without contacting Meili or replaying production writes.
	if !maintenanceEnabled() || payload.Instance != maintenanceInstance() {
		return progress("skipped", "", 0)
	}
	if client == nil {
		return ErrSearchUnavailable
	}
	for _, name := range names {
		if err := progress("starting", name, 0); err != nil {
			return err
		}
		report, err := runManagedIndex(ctx, client, name, payload.Action, progress)
		if err != nil {
			return fmt.Errorf("maintain %s: %w", name, err)
		}
		payload.Reports = append(payload.Reports, *report)
		if err := progress("checked", name, report.Expected); err != nil {
			return err
		}
	}
	return progress("finished", "", 0)
}
