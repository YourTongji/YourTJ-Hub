package searchservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/meilisearch/meilisearch-go"
)

// The extra hit detects overflow; never silently sort/page a truncated match
// set in PostgreSQL. This bound also limits per-request memory and SQL parameters.
const maxCourseCandidates int64 = 20000

var ErrCourseSearchUnavailable = errors.New("course search unavailable")

// SearchCourseCandidates uses the same courses index as aggregate search.
// PostgreSQL remains responsible for visibility, exact filters, review ordering
// and totals. An unconfigured local install retains SQL search; a configured but
// unavailable engine fails closed instead of starting expensive fallback scans.
func SearchCourseCandidates(ctx context.Context, keyword string) ([]uint64, bool, error) {
	client := meiliconnect.GetClient()
	if client == nil || keyword == "" {
		return nil, false, nil
	}
	ids, err := searchCourseCandidates(ctx, client.Index(CourseIndex), keyword)
	if err != nil {
		return nil, true, fmt.Errorf("%w: %w", ErrCourseSearchUnavailable, err)
	}
	return ids, true, nil
}

func searchCourseCandidates(ctx context.Context, index meilisearch.IndexManager, keyword string) ([]uint64, error) {
	// Meili caps even exhaustive totals at maxTotalHits. Check settings so an
	// older index cannot silently produce incomplete catalog pages during rollout.
	pagination, err := index.GetPaginationWithContext(ctx)
	if err != nil {
		return nil, err
	}
	if pagination == nil || pagination.MaxTotalHits <= maxCourseCandidates {
		return nil, errors.New("course index requires refresh-course settings")
	}
	limit := maxCourseCandidates + 1
	response, err := index.SearchWithContext(ctx, keyword, &meilisearch.SearchRequest{
		Page: 1, HitsPerPage: &limit,
		AttributesToRetrieve: []string{"id"},
		MatchingStrategy:     meilisearch.All,
		Filter:               "status = 0",
	})
	if err != nil {
		return nil, err
	}
	if response.TotalHits > maxCourseCandidates || int64(len(response.Hits)) != response.TotalHits {
		return nil, errors.New("course search match set exceeds safe exhaustive limit")
	}
	ids := make([]uint64, 0, len(response.Hits))
	for _, hit := range response.Hits {
		var item struct {
			ID uint64 `json:"id"`
		}
		if err := hit.DecodeInto(&item); err != nil || item.ID == 0 {
			return nil, errors.New("invalid course search hit")
		}
		ids = append(ids, item.ID)
	}
	return ids, nil
}
