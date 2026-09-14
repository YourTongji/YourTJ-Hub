package searchservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

func TestCourseIndexWaitUsesTimeoutAsBudgetNotPollInterval(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := "enqueued"
		if calls.Add(1) > 1 {
			status = "succeeded"
		}
		_, _ = fmt.Fprintf(w, `{"uid":1,"status":%q}`, status)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	if err := waitForTaskCheckedContext(ctx, meilisearch.New(server.URL), 1, 2*time.Second); err != nil {
		t.Fatalf("already finished indexing waited for a timeout-sized polling interval: %v", err)
	}
}

func TestCourseCandidatesRejectIncompleteOrFailedSearch(t *testing.T) {
	for _, tt := range []struct {
		name  string
		cap   int64
		total int64
		hits  string
		fail  bool
	}{
		{"old pagination cap", 1000, 1, `[{"id":1}]`, false},
		{"truncated results", 20001, 2, `[{"id":1}]`, false},
		{"oversized match set", 20001, 20001, `[]`, false},
		{"invalid identity", 20001, 1, `[{"id":0}]`, false},
		{"outage", 20001, 0, `[]`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					_, _ = fmt.Fprintf(w, `{"maxTotalHits":%d}`, tt.cap)
					return
				}
				if tt.fail {
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = fmt.Fprint(w, `{}`)
					return
				}
				_, _ = fmt.Fprintf(w, `{"hits":%s,"totalHits":%d}`, tt.hits, tt.total)
			}))
			defer server.Close()
			_, err := searchCourseCandidates(context.Background(), meilisearch.New(server.URL).Index("courses"), "数学")
			if err == nil {
				t.Fatal("unsafe search response was accepted")
			}
		})
	}
}

func TestCourseCandidatesUseExhaustiveIDsAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = fmt.Fprint(w, `{"maxTotalHits":20001}`)
			return
		}
		var request meilisearch.SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Page != 1 || request.HitsPerPage == nil || *request.HitsPerPage != 20001 || request.MatchingStrategy != meilisearch.All || len(request.AttributesToRetrieve) != 1 || request.AttributesToRetrieve[0] != "id" {
			t.Errorf("non-exhaustive search request: %+v", request)
		}
		_, _ = fmt.Fprint(w, `{"hits":[{"id":9},{"id":3}],"totalHits":2}`)
	}))
	defer server.Close()
	index := meilisearch.New(server.URL).Index("courses")
	ids, err := searchCourseCandidates(context.Background(), index, "数学")
	if err != nil || len(ids) != 2 || ids[0] != 9 || ids[1] != 3 {
		t.Fatalf("candidates=%v error=%v", ids, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := searchCourseCandidates(ctx, index, "数学"); err == nil {
		t.Fatal("canceled request completed search")
	}
}

// Runs against the same Meilisearch version as deployment. No production data
// or credentials are used; each test owns and deletes an isolated index.
func TestCourseCatalogMeiliIntegration(t *testing.T) {
	endpoint := os.Getenv("YOURTJ_TEST_MEILI_URL")
	if endpoint == "" {
		t.Skip("YOURTJ_TEST_MEILI_URL not set")
	}
	client := meilisearch.New(endpoint)
	uid := fmt.Sprintf("course_catalog_test_%d", time.Now().UnixNano())
	index := client.Index(uid)
	t.Cleanup(func() { _, _ = client.DeleteIndex(uid) })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := configureCourseIndex(index); err != nil {
		t.Fatal(err)
	}
	docs := make([]CourseSearchDocument, 1005)
	for i := range docs {
		docs[i] = CourseSearchDocument{ID: uint64(i + 1), Name: "高等数学", NormalizedName: "高等数学", NamePinyin: "gaodengshuxue", NameInitials: "gdsx", PrimaryCode: "100001", Status: 0}
	}
	docs[0].Aliases = []string{"高数"}
	docs[0].ClassCodes = []string{"10000101"}
	docs[0].InstructorSearch = []string{"张三", "zhangsan", "zs"}
	key := "id"
	task, err := index.AddDocumentsWithContext(ctx, docs, &meilisearch.DocumentOptions{PrimaryKey: &key})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := client.WaitForTaskWithContext(ctx, task.TaskUID, 20*time.Millisecond)
	if err != nil || completed.Status != meilisearch.TaskStatusSucceeded {
		t.Fatalf("index write: %+v %v", completed, err)
	}
	for _, tt := range []struct {
		keyword string
		want    int
	}{
		{"高等数学", 1005}, {"gdsx", 1005}, {"10000101", 1}, {"高数", 1}, {"zhangsan", 1}, {"zs", 1}, {"notfoundxyz", 0},
	} {
		ids, err := searchCourseCandidates(ctx, index, tt.keyword)
		if err != nil || len(ids) != tt.want {
			t.Errorf("%q: got %d IDs, want %d; err=%v", tt.keyword, len(ids), tt.want, err)
		}
	}
}
