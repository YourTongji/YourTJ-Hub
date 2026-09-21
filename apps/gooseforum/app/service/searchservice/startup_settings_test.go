package searchservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meilisearch/meilisearch-go"
)

func TestStartupCourseSettingsRepairLegacyPagination(t *testing.T) {
	cap := int64(1000)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
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
			cap = settings.Pagination.MaxTotalHits
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"taskUid":7}`)
		case r.URL.Path == "/tasks/7":
			_, _ = fmt.Fprint(w, `{"uid":7,"status":"succeeded"}`)
		case r.URL.Path == "/indexes/courses/settings/pagination":
			_, _ = fmt.Fprintf(w, `{"maxTotalHits":%d}`, cap)
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
	if attempts != 2 || cap != maxCourseCandidates+1 {
		t.Fatalf("attempts=%d cap=%d", attempts, cap)
	}
	ids, err := searchCourseCandidates(context.Background(), index, "math")
	if err != nil || len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("legacy catalog still unavailable: %v %v", ids, err)
	}
}

func TestStartupSettingsFailureIsBoundedAndCancellable(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
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
	if calls != 6 {
		t.Fatalf("unbounded retries: %d", calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ensureManagedIndexConfigured(ctx, index, CourseIndex, 0); err == nil {
		t.Fatal("cancel ignored")
	}
	if calls != 6 {
		t.Fatal("canceled startup made requests")
	}
}
