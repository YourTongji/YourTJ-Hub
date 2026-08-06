package datamigration

import (
	"errors"
	"testing"

	"github.com/leancodebox/GooseForum/app/service/searchservice"
)

func TestMigrateAggregateSearchIndexesSkipsWhenUnavailable(t *testing.T) {
	result := migrateAggregateSearchIndexes(
		func() bool { return false },
		func() (*searchservice.IndexBuildResult, error) {
			t.Fatal("buildUserIndex should not be called when unavailable")
			return nil, nil
		},
		func() (*searchservice.IndexBuildResult, error) {
			t.Fatal("buildCategoryIndex should not be called when unavailable")
			return nil, nil
		},
	)
	if !result.Skipped {
		t.Fatalf("Skipped = false, want true when Meilisearch is unavailable")
	}
	if result.Failed != 0 {
		t.Fatalf("Failed = %d, want 0 when skipped", result.Failed)
	}
	if result.UsersRebuilt || result.CategoriesRebuilt {
		t.Fatalf("no index should be rebuilt when skipped: %#v", result)
	}
}

func TestMigrateAggregateSearchIndexesBuildsBothIndexes(t *testing.T) {
	result := migrateAggregateSearchIndexes(
		func() bool { return true },
		func() (*searchservice.IndexBuildResult, error) {
			return &searchservice.IndexBuildResult{ProcessedCount: 2, FailedCount: 0, IndexName: searchservice.UserIndex}, nil
		},
		func() (*searchservice.IndexBuildResult, error) {
			return &searchservice.IndexBuildResult{ProcessedCount: 1, FailedCount: 0, IndexName: searchservice.CategoryIndex}, nil
		},
	)
	if !result.UsersRebuilt || result.UsersProcessed != 2 || result.UsersFailed != 0 {
		t.Fatalf("unexpected user index result: %#v", result)
	}
	if !result.CategoriesRebuilt || result.CategoriesProcessed != 1 || result.CategoriesFailed != 0 {
		t.Fatalf("unexpected category index result: %#v", result)
	}
	if result.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", result.Failed)
	}
}

func TestMigrateAggregateSearchIndexesFailsOnUserBuildFailures(t *testing.T) {
	result := migrateAggregateSearchIndexes(
		func() bool { return true },
		func() (*searchservice.IndexBuildResult, error) {
			return &searchservice.IndexBuildResult{ProcessedCount: 2, FailedCount: 1, IndexName: searchservice.UserIndex}, nil
		},
		func() (*searchservice.IndexBuildResult, error) {
			t.Fatal("buildCategoryIndex should not be called when user build has failures")
			return nil, nil
		},
	)
	if result.Failed != 1 || result.LastFailed != "user index build had failures" {
		t.Fatalf("unexpected failure result: %#v", result)
	}
	if !result.UsersRebuilt || result.UsersFailed != 1 {
		t.Fatalf("user index state wrong: %#v", result)
	}
	if result.CategoriesRebuilt {
		t.Fatalf("category index should not be rebuilt when user build failed: %#v", result)
	}
}

func TestMigrateAggregateSearchIndexesFailsOnCategoryBuildFailures(t *testing.T) {
	result := migrateAggregateSearchIndexes(
		func() bool { return true },
		func() (*searchservice.IndexBuildResult, error) {
			return &searchservice.IndexBuildResult{ProcessedCount: 2, FailedCount: 0, IndexName: searchservice.UserIndex}, nil
		},
		func() (*searchservice.IndexBuildResult, error) {
			return &searchservice.IndexBuildResult{ProcessedCount: 1, FailedCount: 1, IndexName: searchservice.CategoryIndex}, nil
		},
	)
	if result.Failed != 1 || result.LastFailed != "category index build had failures" {
		t.Fatalf("unexpected failure result: %#v", result)
	}
	if !result.UsersRebuilt || !result.CategoriesRebuilt || result.CategoriesFailed != 1 {
		t.Fatalf("index state wrong: %#v", result)
	}
}

func TestMigrateAggregateSearchIndexesReportsUserBuildError(t *testing.T) {
	result := migrateAggregateSearchIndexes(
		func() bool { return true },
		func() (*searchservice.IndexBuildResult, error) {
			return nil, errors.New("boom")
		},
		func() (*searchservice.IndexBuildResult, error) {
			t.Fatal("buildCategoryIndex should not be called when user build errors")
			return nil, nil
		},
	)
	if result.Failed != 1 || result.LastFailed != "boom" {
		t.Fatalf("unexpected failure result: %#v", result)
	}
}
