package searchservice

import (
	"encoding/json"
	"testing"

	"github.com/meilisearch/meilisearch-go"
)

func TestNormalizeScope(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "all"},
		{"all", "all"},
		{"topics", "topics"},
		{"users", "users"},
		{"categories", "categories"},
		{"invalid", "all"},
		{"Topics", "all"},
	}
	for _, tc := range cases {
		if got := NormalizeScope(tc.input); got != tc.want {
			t.Fatalf("NormalizeScope(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestScopeQueries(t *testing.T) {
	queries := scopeQueries(AggregateSearchRequest{Query: "hello", Scope: ScopeAll, Limit: 10})
	if len(queries) != 3 {
		t.Fatalf("scope all should produce 3 queries, got %d", len(queries))
	}
	if queries[0].IndexUID != TopicIndex || queries[0].Limit != 10 || queries[0].Offset != 0 {
		t.Fatalf("topics query wrong: %+v", queries[0])
	}
	if queries[1].IndexUID != UserIndex {
		t.Fatalf("users query wrong: %+v", queries[1])
	}
	if queries[2].IndexUID != CategoryIndex {
		t.Fatalf("categories query wrong: %+v", queries[2])
	}

	usersOnly := scopeQueries(AggregateSearchRequest{Query: "hello", Scope: ScopeUsers, Limit: 0})
	if len(usersOnly) != 1 || usersOnly[0].IndexUID != UserIndex {
		t.Fatalf("users scope should produce only users query: %+v", usersOnly)
	}
	if usersOnly[0].Limit != MaxAggregateLimit {
		t.Fatalf("limit should be capped to MaxAggregateLimit, got %d", usersOnly[0].Limit)
	}
}

func TestAggregateSearchEmptyQuery(t *testing.T) {
	resp, err := AggregateSearch(AggregateSearchRequest{Query: "   ", Scope: ScopeAll})
	if err != nil {
		t.Fatalf("empty query should not error, got %v", err)
	}
	if resp == nil || resp.Topics == nil || resp.Users == nil || resp.Categories == nil {
		t.Fatalf("empty query should return empty slices, got %+v", resp)
	}
}

func TestCollectScopeResultsTopics(t *testing.T) {
	resp := &AggregateSearchResponse{Topics: []SearchResult{}, Users: []UserSearchResult{}, Categories: []CategorySearchResult{}}
	searchResp := &meilisearch.SearchResponse{
		Hits: []meilisearch.Hit{
			{"id": json.RawMessage(`1`), "title": json.RawMessage(`"hello"`)},
			{"id": json.RawMessage(`2`), "title": json.RawMessage(`"world"`)},
		},
		EstimatedTotalHits: 2,
	}
	collectScopeResults(resp, TopicIndex, searchResp)
	if len(resp.Topics) != 2 {
		t.Fatalf("topics results = %d, want 2", len(resp.Topics))
	}
	if resp.Topics[0].ID != 1 || resp.Topics[1].ID != 2 {
		t.Fatalf("topics ids wrong: %+v", resp.Topics)
	}
	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2", resp.Total)
	}
}

func TestCollectScopeResultsSkipsInvalidIDs(t *testing.T) {
	resp := &AggregateSearchResponse{Topics: []SearchResult{}, Users: []UserSearchResult{}, Categories: []CategorySearchResult{}}
	searchResp := &meilisearch.SearchResponse{
		Hits: []meilisearch.Hit{
			{"id": json.RawMessage(`0`), "title": json.RawMessage(`"bad"`)},
			{"id": json.RawMessage(`3`), "title": json.RawMessage(`"good"`)},
		},
		EstimatedTotalHits: 1,
	}
	collectScopeResults(resp, TopicIndex, searchResp)
	if len(resp.Topics) != 1 || resp.Topics[0].ID != 3 {
		t.Fatalf("invalid id should be skipped: %+v", resp.Topics)
	}
}
