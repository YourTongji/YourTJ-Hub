package searchservice

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/meilisearch/meilisearch-go"
)

// fakeGhostIndex 是 ghostIndex 的测试替身：模拟分页拉取与批量删除。
type fakeGhostIndex struct {
	docs      []meilisearch.Hit
	total     int64
	deleted   []string
	getCalls  int
	getErr    error
	deleteErr error
}

var _ ghostIndex = (*fakeGhostIndex)(nil)

func (f *fakeGhostIndex) GetDocuments(param *meilisearch.DocumentsQuery, resp *meilisearch.DocumentsResult) error {
	f.getCalls++
	if f.getErr != nil {
		return f.getErr
	}
	limit := param.Limit
	if limit <= 0 {
		limit = int64(len(f.docs))
	}
	offset := param.Offset
	end := offset + limit
	if end > int64(len(f.docs)) {
		end = int64(len(f.docs))
	}
	if offset > int64(len(f.docs)) {
		offset = int64(len(f.docs))
	}
	resp.Results = f.docs[offset:end]
	resp.Limit = limit
	resp.Offset = offset
	resp.Total = f.total
	return nil
}

func (f *fakeGhostIndex) DeleteDocuments(identifiers []string, opts *meilisearch.DocumentOptions) (*meilisearch.TaskInfo, error) {
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	f.deleted = append(f.deleted, identifiers...)
	return &meilisearch.TaskInfo{TaskUID: 1}, nil
}

func hit(id string) meilisearch.Hit {
	raw, _ := json.Marshal(id)
	return meilisearch.Hit{"id": json.RawMessage(raw)}
}

func numericHit(id uint64) meilisearch.Hit {
	raw, _ := json.Marshal(id)
	return meilisearch.Hit{"id": json.RawMessage(raw)}
}

func TestCleanupGhostDocumentsRemovesOnlyGhosts(t *testing.T) {
	fake := &fakeGhostIndex{
		docs:  []meilisearch.Hit{numericHit(1), numericHit(2), numericHit(3), numericHit(4)},
		total: 4,
	}
	want := map[string]struct{}{"2": {}, "4": {}}

	removed, err := cleanupGhostDocuments(fake, want)
	if err != nil {
		t.Fatalf("cleanupGhostDocuments error: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if len(fake.deleted) != 2 || fake.deleted[0] != "1" || fake.deleted[1] != "3" {
		t.Fatalf("deleted = %v, want [1 3]", fake.deleted)
	}
}

func TestCleanupGhostDocumentsNoGhosts(t *testing.T) {
	fake := &fakeGhostIndex{
		docs:  []meilisearch.Hit{numericHit(1), numericHit(2)},
		total: 2,
	}
	want := map[string]struct{}{"1": {}, "2": {}}

	removed, err := cleanupGhostDocuments(fake, want)
	if err != nil {
		t.Fatalf("cleanupGhostDocuments error: %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0", removed)
	}
	if len(fake.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", fake.deleted)
	}
}

func TestCleanupGhostDocumentsMissingIndex(t *testing.T) {
	fake := &fakeGhostIndex{getErr: &meilisearch.Error{StatusCode: http.StatusNotFound}}

	removed, err := cleanupGhostDocuments(fake, map[string]struct{}{"1": {}})
	if err != nil {
		t.Fatalf("missing index should not error, got %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0", removed)
	}
}

func TestCleanupGhostDocumentsGetErrorFails(t *testing.T) {
	fake := &fakeGhostIndex{getErr: &meilisearch.Error{StatusCode: http.StatusInternalServerError}}

	if _, err := cleanupGhostDocuments(fake, map[string]struct{}{"1": {}}); err == nil {
		t.Fatal("non-404 fetch error should fail the rebuild")
	}
}

func TestCleanupGhostDocumentsDeleteErrorFails(t *testing.T) {
	fake := &fakeGhostIndex{
		docs:      []meilisearch.Hit{numericHit(1), numericHit(2)},
		total:     2,
		deleteErr: &meilisearch.Error{StatusCode: http.StatusInternalServerError},
	}

	if _, err := cleanupGhostDocuments(fake, map[string]struct{}{"1": {}}); err == nil {
		t.Fatal("delete error should fail the rebuild")
	}
}

func TestFetchIndexDocumentIDsPaginates(t *testing.T) {
	var docs []meilisearch.Hit
	for i := 0; i < 2500; i++ {
		docs = append(docs, numericHit(uint64(i+1)))
	}
	fake := &fakeGhostIndex{docs: docs, total: 2500}

	ids, err := fetchIndexDocumentIDs(fake)
	if err != nil {
		t.Fatalf("fetchIndexDocumentIDs error: %v", err)
	}
	if len(ids) != 2500 {
		t.Fatalf("ids = %d, want 2500", len(ids))
	}
	if fake.getCalls != 3 {
		t.Fatalf("GetDocuments calls = %d, want 3 (pagination)", fake.getCalls)
	}
}

func TestFetchIndexDocumentIDsEmpty(t *testing.T) {
	fake := &fakeGhostIndex{docs: nil, total: 0}
	ids, err := fetchIndexDocumentIDs(fake)
	if err != nil {
		t.Fatalf("fetchIndexDocumentIDs error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids = %v, want empty", ids)
	}
}

func TestDiffGhostIDs(t *testing.T) {
	indexed := []string{"1", "2", "3"}
	want := map[string]struct{}{"1": {}, "3": {}, "5": {}}
	ghosts := diffGhostIDs(indexed, want)
	if len(ghosts) != 1 || ghosts[0] != "2" {
		t.Fatalf("ghosts = %v, want [2]", ghosts)
	}
}

func TestHitDocumentID(t *testing.T) {
	numeric, err := hitDocumentID(numericHit(123))
	if err != nil || numeric != "123" {
		t.Fatalf("numeric id = %q, %v; want 123", numeric, err)
	}
	str, err := hitDocumentID(hit("abc"))
	if err != nil || str != "abc" {
		t.Fatalf("string id = %q, %v; want abc", str, err)
	}
	if _, err := hitDocumentID(meilisearch.Hit{"title": json.RawMessage(`"x"`)}); err == nil {
		t.Fatal("missing id should error")
	}
}
