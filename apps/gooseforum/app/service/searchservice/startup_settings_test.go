package searchservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/meilisearch/meilisearch-go"
)

func TestStartupCourseSettingsRepairLegacyPagination(t *testing.T) {
	enableStartupMaintenance(t)
	paginationCap := int64(1000)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/indexes/courses":
			_, _ = fmt.Fprint(w, `{"uid":"courses","primaryKey":"id"}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/indexes/courses/settings":
			attempts++
			if attempts == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, `{"message":"starting"}`)
				return
			}
			var settings meilisearch.Settings
			if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
				t.Error(err)
				return
			}
			if settings.Pagination == nil {
				t.Error("pagination not managed")
				return
			}
			paginationCap = settings.Pagination.MaxTotalHits
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"taskUid":7}`)
		case r.URL.Path == "/tasks/7":
			_, _ = fmt.Fprint(w, `{"uid":7,"status":"succeeded"}`)
		case r.URL.Path == "/indexes/courses/settings/pagination":
			_, _ = fmt.Fprintf(w, `{"maxTotalHits":%d}`, paginationCap)
		case r.URL.Path == "/indexes/courses/search":
			_, _ = fmt.Fprint(w, `{"hits":[{"id":42}],"totalHits":1}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	index := meilisearch.New(server.URL).Index(CourseIndex)
	if err := ensureManagedIndexConfigured(context.Background(), index, CourseIndex, 0); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || paginationCap != maxCourseCandidates+1 {
		t.Fatalf("attempts=%d paginationCap=%d", attempts, paginationCap)
	}
	ids, err := searchCourseCandidates(context.Background(), index, "math")
	if err != nil || len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("legacy catalog still unavailable: %v %v", ids, err)
	}
}

func TestStartupSettingsFailureIsBoundedAndCancellable(t *testing.T) {
	enableStartupMaintenance(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/indexes/courses" {
			_, _ = fmt.Fprint(w, `{"uid":"courses","primaryKey":"id"}`)
			return
		}
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"taskUid":1}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"uid":1,"status":"failed","error":{"message":"rejected"}}`)
	}))
	defer server.Close()
	index := meilisearch.New(server.URL).Index(CourseIndex)
	if err := ensureManagedIndexConfigured(context.Background(), index, CourseIndex, 0); err == nil {
		t.Fatal("failed settings task accepted")
	}
	if calls != 9 {
		t.Fatalf("unbounded retries: %d", calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ensureManagedIndexConfigured(ctx, index, CourseIndex, 0); err == nil {
		t.Fatal("cancel ignored")
	}
	if calls != 9 {
		t.Fatal("canceled startup made requests")
	}
}

func TestStartupCourseSettingsRespectOwnerAndMissingIndex(t *testing.T) {
	old := preferences.GetBool("meilisearch.maintenance_enabled", false)
	t.Cleanup(func() { preferences.Set("meilisearch.maintenance_enabled", old) })
	for _, owner := range []bool{false, true} {
		preferences.Set("meilisearch.maintenance_enabled", owner)
		patches, reads := 0, 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodPatch {
				patches++
				w.WriteHeader(http.StatusAccepted)
				_, _ = fmt.Fprint(w, `{"taskUid":1}`)
				return
			}
			if r.URL.Path == "/tasks/1" {
				_, _ = fmt.Fprint(w, `{"uid":1,"status":"succeeded"}`)
				return
			}
			reads++
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"code":"index_not_found","message":"missing","type":"invalid_request"}`)
		}))
		err := ensureManagedIndexConfigured(context.Background(), meilisearch.New(server.URL).Index(CourseIndex), CourseIndex, 0)
		server.Close()
		if patches != 0 {
			t.Fatalf("owner=%t created missing index through %d settings writes", owner, patches)
		}
		if !owner && (reads != 0 || err != nil) {
			t.Fatal("non-owner contacted shared index")
		}
		if owner && err == nil {
			t.Fatal("missing projection marked ready")
		}
	}
}

// The topic repair must reach the settings PATCH without the course-style
// existence probe: any probe here would 404-fail a repair that is expected to
// succeed once the engine answers.
func TestStartupTopicSettingsSkipExistenceProbe(t *testing.T) {
	fetches, patches := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch && r.URL.Path == "/indexes/topics/settings" {
			patches++
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"taskUid":3}`)
			return
		}
		if r.URL.Path == "/tasks/3" {
			_, _ = fmt.Fprint(w, `{"uid":3,"status":"succeeded"}`)
			return
		}
		fetches++
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"code":"index_not_found","message":"missing","type":"invalid_request"}`)
	}))
	defer server.Close()
	if err := ensureManagedIndexConfigured(context.Background(), meilisearch.New(server.URL).Index(TopicIndex), TopicIndex, 0); err != nil {
		t.Fatal(err)
	}
	if fetches != 0 {
		t.Fatalf("topic repair probed index existence %d times", fetches)
	}
	if patches != 1 {
		t.Fatalf("patches=%d, want exactly 1", patches)
	}
}

func enableStartupMaintenance(t *testing.T) {
	t.Helper()
	old := preferences.GetBool("meilisearch.maintenance_enabled", false)
	preferences.Set("meilisearch.maintenance_enabled", true)
	t.Cleanup(func() { preferences.Set("meilisearch.maintenance_enabled", old) })
}

func TestStartupQueuedSettingsRespectWholeOperationDeadline(t *testing.T) {
	enableStartupMaintenance(t)
	patches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/indexes/courses":
			_, _ = fmt.Fprint(w, `{"uid":"courses","primaryKey":"id"}`)
		case "/indexes/courses/settings":
			patches++
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"taskUid":1}`)
		default:
			_, _ = fmt.Fprint(w, `{"uid":1,"status":"enqueued"}`)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := ensureManagedIndexConfigured(ctx, meilisearch.New(server.URL).Index(CourseIndex), CourseIndex, time.Second)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second || patches != 1 {
		t.Fatalf("queued repair escaped budget: %v, patches=%d", err, patches)
	}
}
