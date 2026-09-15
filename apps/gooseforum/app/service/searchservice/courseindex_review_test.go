package searchservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/meilisearch/meilisearch-go"
)

func TestConfigureCourseIndexPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/indexes/courses/settings" {
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"taskUid":17,"status":"enqueued"}`)
			return
		}
		cancel()
		_, _ = fmt.Fprint(w, `{"uid":17,"status":"processing"}`)
	}))
	defer server.Close()
	if err := configureCourseIndex(ctx, meilisearch.New(server.URL).Index(CourseIndex)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was lost while waiting for settings: %v", err)
	}
}

func TestConfigureCourseIndexChecksSettingsTask(t *testing.T) {
	for _, status := range []string{"succeeded", "failed", "canceled"} {
		t.Run(status, func(t *testing.T) {
			var polls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/indexes/courses/settings":
					w.WriteHeader(http.StatusAccepted)
					_, _ = fmt.Fprint(w, `{"taskUid":17,"status":"enqueued"}`)
				case "/tasks/17":
					next := status
					if polls.Add(1) == 1 {
						next = "processing"
					}
					_, _ = fmt.Fprintf(w, `{"uid":17,"status":%q,"error":{"message":"settings rejected"}}`, next)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			err := configureCourseIndex(context.Background(), meilisearch.New(server.URL).Index(CourseIndex))
			if status == "succeeded" && err != nil {
				t.Fatal(err)
			}
			if status != "succeeded" && (err == nil || !strings.Contains(err.Error(), "settings rejected")) {
				t.Fatalf("settings task %s must abort configuration, got %v", status, err)
			}
			if polls.Load() < 2 {
				t.Fatal("returned before settings reached a terminal state")
			}
		})
	}
}

func TestCourseProjectionIncludesIdentityTeacherWithoutOfferings(t *testing.T) {
	setupCourseSearchTestDB(t)
	db := dbconnect.Connect()
	teacher := course.InstructorEntity{Id: 77, Name: "张三", NormalizedName: "张三", NamePinyin: "zhangsan", NameInitials: "zs"}
	if err := db.Create(&teacher).Error; err != nil {
		t.Fatal(err)
	}
	entities := []course.Entity{
		{Id: 81, PrimaryCode: "pure-review", Name: "纯评价课程", TeacherId: teacher.Id, Status: course.StatusVisible},
		{Id: 82, PrimaryCode: "without-teacher", Name: "无教师课程", Status: course.StatusVisible},
	}
	if err := db.Create(&entities).Error; err != nil {
		t.Fatal(err)
	}
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprintf("batch=%t", batch), func(t *testing.T) {
			var docs []CourseSearchDocument
			if batch {
				var err error
				docs, err = convertCoursesToSearchDocuments(entities)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				for _, entity := range entities {
					doc, err := convertCourseToSearchDocument(entity)
					if err != nil {
						t.Fatal(err)
					}
					docs = append(docs, doc)
				}
			}
			if docs[0].TeacherName != teacher.Name {
				t.Fatalf("identity teacher missing: %+v", docs[0])
			}
			for _, form := range []string{teacher.NormalizedName, teacher.NamePinyin, teacher.NameInitials} {
				if !slices.Contains(docs[0].InstructorSearch, form) {
					t.Errorf("missing identity search form %q: %v", form, docs[0].InstructorSearch)
				}
			}
			if len(docs[1].InstructorSearch) != 0 {
				t.Fatalf("search forms leaked to unrelated card: %+v", docs[1])
			}
		})
	}
}

func TestCourseIndexScanSurvivesDeletionBetweenBatches(t *testing.T) {
	setupCourseSearchTestDB(t)
	db := dbconnect.Connect()
	entities := make([]course.Entity, 603)
	for i := range entities {
		entities[i] = course.Entity{Id: uint64((i + 1) * 10), PrimaryCode: fmt.Sprintf("cursor-%d", i), Status: course.StatusVisible}
		// The final row of the first page is hidden; the entire third page is hidden.
		if i == 199 || (i >= 400 && i < 600) {
			entities[i].Status = course.StatusHidden
		}
	}
	if err := db.CreateInBatches(entities, 100).Error; err != nil {
		t.Fatal(err)
	}
	seen := make(map[uint64]int)
	deleted := false
	result, err := buildCourseIndexPages(context.Background(), course.ListCoursesAfterID,
		func(rows []course.Entity) ([]CourseSearchDocument, error) {
			docs := make([]CourseSearchDocument, 0, len(rows))
			for _, row := range rows {
				docs = append(docs, CourseSearchDocument{ID: row.Id})
			}
			return docs, nil
		},
		func(docs []CourseSearchDocument) error {
			for _, doc := range docs {
				seen[doc.ID]++
			}
			if !deleted {
				deleted = true
				if err := db.Unscoped().Delete(&entities[0]).Error; err != nil {
					return err
				}
				return db.Delete(&entities[1]).Error
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	for _, entity := range entities[2:] {
		want := 1
		if entity.Status == course.StatusHidden {
			want = 0
		}
		if seen[entity.Id] != want {
			t.Errorf("course %d indexed %d times, want %d", entity.Id, seen[entity.Id], want)
		}
	}
	if result.ProcessedCount != len(entities) {
		t.Errorf("processed=%d, want %d", result.ProcessedCount, len(entities))
	}
}
