package searchservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/meilisearch/meilisearch-go"
)

func TestMaintenanceDigest(t *testing.T) {
	decode := func(raw string) projectionDocument {
		var doc projectionDocument
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	a := decode(`{"id":9007199254740993,"values":["b","a"],"_projectionVersion":1}`)
	b := decode(`{"values":["a","b"],"_projectionVersion":1,"id":9007199254740993}`)
	aa, _ := digestDocument(a)
	bb, _ := digestDocument(b)
	if aa != bb {
		t.Fatal("unordered projection arrays should compare equal")
	}
	id, _ := documentID(a)
	if id != "9007199254740993" {
		t.Fatal("uint64 ID precision lost")
	}
	delete(b, "_projectionVersion")
	bb, _ = digestDocument(b)
	if bb.Hash == aa.Hash || bb.Version != "0" {
		t.Fatal("legacy document should be outdated/unmarked")
	}
	if _, err := documentID(projectionDocument{}); err == nil {
		t.Fatal("missing id accepted")
	}
}

func TestMaintenanceRejectsNoncanonicalDocumentIDs(t *testing.T) {
	setupCourseSearchTestDB(t)
	if err := dbconnect.Connect().Create(&course.Entity{Id: 1, PrimaryCode: "ONE", Status: course.StatusVisible}).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"001", "-1", "1.0", "1x", "90071992547409930000000"} {
		doc, err := currentProjection(context.Background(), CourseIndex, id)
		if err != nil || doc != nil {
			t.Fatalf("ghost ID %q resolved to a valid source: %v %v", id, doc, err)
		}
	}
	doc, err := currentProjection(context.Background(), CourseIndex, "1")
	if err != nil || doc == nil {
		t.Fatalf("valid source rejected: %v", err)
	}
}

func TestMaintenanceExclusiveTaskAndRedaction(t *testing.T) {
	setupCourseSearchTestDB(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	created := make(chan bool, 12)
	ids := make(chan uint64, 12)
	for range 12 {
		wg.Go(func() {
			row, fresh, err := taskQueue.CreateSearchMaintenance(ctx, `{"index":"all","action":"check","requestedBy":1,"reports":[]}`)
			if err != nil {
				t.Error(err)
				return
			}
			created <- fresh
			ids <- row.Id
		})
	}
	wg.Wait()
	close(created)
	close(ids)
	count := 0
	for fresh := range created {
		if fresh {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("concurrent requests created %d tasks", count)
	}
	var id uint64
	for current := range ids {
		if id != 0 && current != id {
			t.Fatal("duplicate did not return active task")
		}
		id = current
	}
	row, claimed, err := taskQueue.ClaimTask(id)
	if err != nil || !claimed {
		t.Fatalf("claim: %v", err)
	}
	if err := taskQueue.UpdateStatusOwned(id, taskQueue.StatusFailed, row.LeaseToken, errors.New("secret endpoint, private document content")); err != nil {
		t.Fatal(err)
	}
	row, err = taskQueue.GetByID(id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(maintenanceJob(row))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "lease") || !strings.Contains(string(raw), "operation_failed") {
		t.Fatalf("unsafe job response: %s", raw)
	}
	_, fresh, err := taskQueue.CreateSearchMaintenance(ctx, `{}`)
	if err != nil || !fresh {
		t.Fatalf("terminal task did not release active slot: %v", err)
	}
	// Other task types must retain normal multi-row enqueue behavior.
	for range 2 {
		if err := taskQueue.Create(&taskQueue.Entity{Type: "course-search.upsert"}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMaintenanceLeaseLossStopsBeforeMeiliMutation(t *testing.T) {
	setupCourseSearchTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("lost lease contacted Meili")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	row, _, err := taskQueue.CreateSearchMaintenance(context.Background(), `{"index":"all","action":"rebuild"}`)
	if err != nil {
		t.Fatal(err)
	}
	row.LeaseToken = "stale"
	if err := runIndexMaintenance(context.Background(), meilisearch.New(server.URL), &row); err == nil || !strings.Contains(err.Error(), "lease lost") {
		t.Fatalf("lost lease was not fenced: %v", err)
	}
}

func TestMaintenanceInstanceIsolation(t *testing.T) {
	old := preferences.GetBool("meilisearch.maintenance_enabled", false)
	t.Cleanup(func() { preferences.Set("meilisearch.maintenance_enabled", old) })
	for _, scenario := range []string{"foreign-instance", "maintenance-disabled"} {
		t.Run(scenario, func(t *testing.T) {
			setupCourseSearchTestDB(t)
			preferences.Set("meilisearch.maintenance_enabled", scenario != "maintenance-disabled")
			payload := MaintenancePayload{MaintenanceProgress: MaintenanceProgress{MaintenanceRequest: MaintenanceRequest{Index: "all", Action: "rebuild"}}, Instance: maintenanceInstance()}
			if scenario == "foreign-instance" {
				payload.Instance = "copied-from-another-origin"
			}
			raw, _ := json.Marshal(payload)
			row, _, err := taskQueue.CreateSearchMaintenance(context.Background(), string(raw))
			if err != nil {
				t.Fatal(err)
			}
			row, claimed, err := taskQueue.ClaimTask(row.Id)
			if err != nil || !claimed {
				t.Fatalf("claim %v", err)
			}
			// A nil client proves no external API is reached when skipping snapshots.
			if err := runIndexMaintenance(context.Background(), nil, &row); err != nil {
				t.Fatal(err)
			}
			row, err = taskQueue.GetByID(row.Id)
			if err != nil {
				t.Fatal(err)
			}
			if maintenanceJob(row).Phase != "skipped" {
				t.Fatal("copied/disabled task not skipped")
			}
		})
	}
}

func TestMaintenanceMissingIndexAndTransportFailure(t *testing.T) {
	setupCourseSearchTestDB(t)
	if err := dbconnect.Connect().Create(&course.Entity{Id: 17, PrimaryCode: "MISSING", Name: "Missing course", Status: course.StatusVisible}).Error; err != nil {
		t.Fatal(err)
	}
	for _, status := range []int{404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				code := "internal"
				if status == 404 {
					code = "index_not_found"
				}
				_, _ = fmt.Fprintf(w, `{"code":%q,"message":"unavailable"}`, code)
			}))
			defer server.Close()
			report, err := checkManagedIndex(context.Background(), meilisearch.New(server.URL), CourseIndex, func(string, string, int) error { return nil })
			if status == 404 {
				if err != nil || report.Complete || report.Missing != 1 || !report.Stable {
					t.Fatalf("missing index: %+v %v", report, err)
				}
			} else if err == nil {
				t.Fatal("server error reported as a missing index")
			}
		})
	}
}

func TestMaintenanceSettingsFailureDoesNotWriteDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/indexes/courses/settings":
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"taskUid":1,"status":"enqueued"}`)
		case "/tasks/1":
			_, _ = fmt.Fprint(w, `{"uid":1,"status":"failed","error":{"code":"invalid_settings","message":"rejected"}}`)
		default:
			t.Errorf("failed settings touched documents: %s", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	if err := rebuildManagedIndex(context.Background(), meilisearch.New(server.URL), CourseIndex, func(string, string, int) error { return nil }); err == nil {
		t.Fatal("settings failure ignored")
	}
}

// Explicit opt-in: this URL must point at a disposable, dedicated Meili engine.
// Real v1.11 coverage verifies hidden attributes, task completion and replacements.
func TestMaintenanceAllIndexesIntegration(t *testing.T) {
	url := os.Getenv("YOURTJ_TEST_MAINTENANCE_MEILI_URL")
	if url == "" {
		t.Skip("set YOURTJ_TEST_MAINTENANCE_MEILI_URL to a disposable engine")
	}
	setupCourseSearchTestDB(t)
	conn := dbconnect.Connect()
	models := []any{&users.EntityComplete{}, &category.Entity{}, &topics.Entity{}, &posts.Entity{}, &wikiPages.Entity{}}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatal(err)
		}
	}
	create := func(value any) {
		t.Helper()
		if err := conn.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 205; i++ {
		status := course.StatusVisible
		if i > 100 && i <= 200 {
			status = course.StatusHidden
		}
		create(&course.Entity{Id: uint64(i), PrimaryCode: fmt.Sprintf("C%d", i), Name: fmt.Sprintf("Course %d", i), Status: status})
	}
	create(&users.EntityComplete{Id: 1, Username: "human", Email: "human@example.test"})
	create(&users.EntityComplete{Id: 2, Username: "bot", Email: "bot@example.test", ActorType: users.ActorTypeBot})
	create(&category.Entity{Id: 1, Name: "分类", Slug: "test"})
	create(&topics.Entity{Id: 1, Title: "Public", Status: 1, FirstPostId: 1})
	create(&posts.Entity{Id: 1, TopicId: 1, PostNo: 1, Content: "Public body"})
	create(&topics.Entity{Id: 2, Title: "Private", Status: 0, FirstPostId: 2})
	create(&posts.Entity{Id: 2, TopicId: 2, PostNo: 1, Content: "Private body"})
	create(&wikiPages.Entity{Id: 1, TopicId: 1, Path: "public", Title: "Wiki", ParaAnchors: `[{"index":1,"anchor":"s-1","text":"First paragraph"},{"index":2,"anchor":"s-2","text":"Second paragraph"}]`})
	create(&wikiPages.Entity{Id: 2, TopicId: 2, Path: "private", Title: "Private wiki", ParaAnchors: `[{"index":1,"text":"secret"}]`})
	client := meilisearch.New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	progress := func(string, string, int) error { return ctx.Err() }
	counts := map[string]int{CourseIndex: 105, TopicIndex: 1, UserIndex: 1, CategoryIndex: 1, WikiPageIndex: 2}
	for _, name := range managedIndexes {
		t.Run(name, func(t *testing.T) {
			// Dirty equal-count indexes must not pass simply because counts match.
			report, err := runManagedIndex(ctx, client, name, "rebuild", progress)
			if err != nil {
				t.Fatal(err)
			}
			if !report.Complete || report.Expected != counts[name] {
				t.Fatalf("rebuild did not reconcile: %+v", report)
			}
			batch, _, err := projectionPage(ctx, name, 0)
			if err != nil {
				t.Fatal(err)
			}
			first, err := documentID(batch[0])
			if err != nil {
				t.Fatal(err)
			}
			index := client.Index(name)
			task, err := index.DeleteDocument(first, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := waitForTaskCheckedContext(ctx, client, task.TaskUID, time.Minute); err != nil {
				t.Fatal(err)
			}
			ghost, _ := projectionJSON(map[string]any{"id": "999999-ghost", "private": "must never appear in admin response"})
			if err := writeProjectionBatch(ctx, index, []projectionDocument{ghost}); err != nil {
				t.Fatal(err)
			}
			report, err = checkManagedIndex(ctx, client, name, progress)
			if err != nil {
				t.Fatal(err)
			}
			if report.Complete || report.Expected != report.Indexed || report.Missing != 1 || report.Extra != 1 || report.ObservedVersions["0"] != 1 {
				t.Fatalf("equal-count drift missed: %+v", report)
			}
			report, err = runManagedIndex(ctx, client, name, "rebuild", progress)
			if err != nil {
				t.Fatal(err)
			}
			if !report.Complete {
				t.Fatalf("ghost repair incomplete: %+v", report)
			}
			// Same id/count but old content and missing version must be detected.
			delete(batch[0], "_projectionVersion")
			if err := writeProjectionBatch(ctx, index, batch[:1]); err != nil {
				t.Fatal(err)
			}
			report, err = checkManagedIndex(ctx, client, name, progress)
			if err != nil {
				t.Fatal(err)
			}
			if report.Outdated != 1 || report.Complete {
				t.Fatalf("outdated projection missed: %+v", report)
			}
			report, err = runManagedIndex(ctx, client, name, "rebuild", progress)
			if err != nil || !report.Complete {
				t.Fatalf("version repair: %+v %v", report, err)
			}
			// A concurrent no-content-change write still invalidates the scan's
			// stability claim, even if every digest and both counts match.
			if name == UserIndex {
				changed := false
				report, err = checkManagedIndex(ctx, client, name, func(phase, _ string, _ int) error {
					if phase == "read_index" && !changed {
						changed = true
						docs, _, err := projectionPage(ctx, name, 0)
						if err != nil {
							return err
						}
						return writeProjectionBatch(ctx, index, docs)
					}
					return nil
				})
				if err != nil || report.Stable || report.Complete {
					t.Fatalf("concurrent write reported complete: %+v %v", report, err)
				}
			}
		})
	}
	// Empty source rebuilds remove all ghosts without DeleteAllDocuments.
	if err := conn.Where("1 = 1").Delete(&category.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	report, err := runManagedIndex(ctx, client, CategoryIndex, "rebuild", progress)
	if err != nil || !report.Complete || report.Indexed != 0 {
		t.Fatalf("empty index: %+v %v", report, err)
	}
}
