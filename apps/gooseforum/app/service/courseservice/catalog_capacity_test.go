package courseservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

func TestCatalogCandidatesKeepDatabaseVisibilityFiltersAndOrdering(t *testing.T) {
	db := setupCatalogTest(t)
	first := createCatalogCourse(t, db, "one", "Math")
	second := createCatalogCourse(t, db, "two", "Math")
	other := createCatalogCourse(t, db, "other", "CS")
	hidden := createCatalogCourse(t, db, "hidden", "Math")
	deleted := createCatalogCourse(t, db, "deleted", "Math")
	db.Model(&course.Entity{}).Where("id = ?", hidden).Update("status", course.StatusHidden)
	db.Delete(&course.Entity{Id: deleted})
	setCatalogStats(t, db, first, 2, 8, 2)
	setCatalogStats(t, db, second, 1, 5, 1)
	previous := searchCatalogCandidates
	t.Cleanup(func() { searchCatalogCandidates = previous })
	searchCatalogCandidates = func(ctx context.Context, keyword string) ([]uint64, bool, error) {
		return []uint64{other, first, deleted, 999999, hidden, second}, true, nil
	}
	page, err := ListCatalogContext(context.Background(), CatalogQuery{Keyword: "does-not-match-sql", Department: []string{"Math"}, HasReview: true, SortBy: "rating", Size: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || !page.HasNext || len(page.List) != 1 || page.List[0].Id != second {
		t.Fatalf("database validation/order lost: %+v", page)
	}
	searchCatalogCandidates = func(context.Context, string) ([]uint64, bool, error) { return []uint64{}, true, nil }
	empty, err := ListCatalog(CatalogQuery{Keyword: "missing"})
	if err != nil || empty.Total != 0 || len(empty.List) != 0 {
		t.Fatalf("empty matches became an unfiltered catalog: %+v %v", empty, err)
	}
}

func TestCatalogRejectsExcessWorkWithoutStartingSearch(t *testing.T) {
	previous := searchCatalogCandidates
	t.Cleanup(func() { searchCatalogCandidates = previous })
	searchCatalogCandidates = func(context.Context, string) ([]uint64, bool, error) {
		t.Fatal("overloaded request started search")
		return nil, false, nil
	}
	for range cap(catalogSlots) {
		catalogSlots <- struct{}{}
	}
	defer func() {
		for range cap(catalogSlots) {
			<-catalogSlots
		}
	}()
	_, err := ListCatalog(CatalogQuery{})
	if !errors.Is(err, ErrCatalogBusy) {
		t.Fatalf("expected immediate overload rejection, got %v", err)
	}
}

func TestCatalogCancellationReachesConnectionWait(t *testing.T) {
	db := setupCatalogTest(t)
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	previousMax := pool.Stats().MaxOpenConnections
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.SetMaxOpenConns(previousMax) })
	conn, err := pool.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = ListCatalogContext(ctx, CatalogQuery{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("connection wait ignored request cancellation: %v", err)
	}
}
