package searchservice

import "errors"

// ErrSearchUnavailable 表示搜索服务不可用
var ErrSearchUnavailable = errors.New("search service unavailable")

// SearchResult is one search hit (topic scope).
type SearchResult struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
}
