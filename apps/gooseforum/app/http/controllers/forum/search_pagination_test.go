package forum

import (
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"testing"
)

func TestSearchPagePagination(t *testing.T) {
	previous := searchPageAggregate
	t.Cleanup(func() { searchPageAggregate = previous })
	for _, tc := range []struct {
		scope       string
		total       int64
		page, pages int
		next        bool
	}{
		{"topics", 27, 1, 3, true}, {"topics", 27, 2, 3, true}, {"topics", 27, 3, 3, false},
		{"all", 27, 1, 3, true}, {"topics", 0, 1, 0, false}, {"topics", 10, 1, 1, false},
	} {
		t.Run(fmt.Sprintf("%s/%d/page%d", tc.scope, tc.total, tc.page), func(t *testing.T) {
			searchPageAggregate = func(req searchservice.AggregateSearchRequest) (*searchservice.AggregateSearchResponse, error) {
				if req.Offset != (tc.page-1)*10 || req.Scope != tc.scope {
					t.Fatalf("request = %+v", req)
				}
				return &searchservice.AggregateSearchResponse{Total: tc.total}, nil
			}
			got := buildSearchPageProps("课程", tc.scope, tc.page)
			if got.TotalPages != tc.pages || got.Pagination.Page != tc.page || got.Pagination.HasNext != tc.next {
				t.Fatalf("pagination = %+v, pages=%d", got.Pagination, got.TotalPages)
			}
			if tc.next && got.Pagination.NextURL != buildSearchURL("课程", tc.scope, tc.page+1) {
				t.Fatalf("next URL lost query/scope: %s", got.Pagination.NextURL)
			}
		})
	}
}
