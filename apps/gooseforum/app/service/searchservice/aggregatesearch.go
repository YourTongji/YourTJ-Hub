package searchservice

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/leancodebox/GooseForum/app/bundles/connect/meiliconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/meilisearch/meilisearch-go"
	"github.com/samber/lo"
)

// Search scope values.
const (
	ScopeAll        = "all"
	ScopeTopics     = "topics"
	ScopeUsers      = "users"
	ScopeCategories = "categories"
	ScopeCourses    = "courses"
)

// MaxAggregateLimit caps per-scope result counts.
const MaxAggregateLimit = 30

// AggregateSearchRequest is an aggregate search across topics, users, categories and courses.
type AggregateSearchRequest struct {
	Query  string
	Scope  string // all / topics / users / categories / courses
	Limit  int
	Offset int
}

// UserSearchResult 用户搜索结果（展示数据由 DB 重构填充）
type UserSearchResult struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Bio       string `json:"bio"`
}

// CategorySearchResult 分类搜索结果（展示数据由 DB 重构填充）
type CategorySearchResult struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
	Desc  string `json:"desc"`
}

// CourseSearchResult 课程搜索结果（展示数据由 PG 重构填充）
type CourseSearchResult struct {
	ID          uint64   `json:"id"`
	PrimaryCode string   `json:"primaryCode"`
	Name        string   `json:"name"`
	Department  string   `json:"department"`
	CreditX10   int      `json:"creditX10"`
	Aliases     []string `json:"aliases"`
	Instructors []string `json:"instructors"`
	Terms       []string `json:"terms"`
	Campus      []string `json:"campus"`
}

// AggregateSearchResponse 分组聚合搜索结果。
type AggregateSearchResponse struct {
	Topics          []SearchResult         `json:"topics"`
	Users           []UserSearchResult     `json:"users"`
	Categories      []CategorySearchResult `json:"categories"`
	Courses         []CourseSearchResult   `json:"courses"`
	Total           int64                  `json:"total"`
	UsersTotal      int64                  `json:"usersTotal"`
	CategoriesTotal int64                  `json:"categoriesTotal"`
	CoursesTotal    int64                  `json:"coursesTotal"`
	FailedScopes    []string               `json:"failedScopes"`
}

// NormalizeScope 归一整合法 scope，非法值回退 all。
func NormalizeScope(raw string) string {
	switch raw {
	case ScopeTopics, ScopeUsers, ScopeCategories, ScopeCourses:
		return raw
	default:
		return ScopeAll
	}
}

// scopeQueries 按 scope 构建 multi-search 的 query 切片。
func scopeQueries(req AggregateSearchRequest) []*meilisearch.SearchRequest {
	queries := make([]*meilisearch.SearchRequest, 0, 4)
	limit := req.Limit
	if limit <= 0 || limit > MaxAggregateLimit {
		limit = MaxAggregateLimit
	}
	if req.Scope == ScopeAll || req.Scope == ScopeTopics {
		queries = append(queries, &meilisearch.SearchRequest{
			IndexUID:             TopicIndex,
			Query:                req.Query,
			Limit:                int64(limit),
			Offset:               int64(req.Offset),
			AttributesToRetrieve: []string{"id", "title"},
		})
	}
	if req.Scope == ScopeAll || req.Scope == ScopeUsers {
		queries = append(queries, &meilisearch.SearchRequest{
			IndexUID:             UserIndex,
			Query:                req.Query,
			Limit:                int64(limit),
			AttributesToRetrieve: []string{"id", "username", "nickname"},
		})
	}
	if req.Scope == ScopeAll || req.Scope == ScopeCategories {
		queries = append(queries, &meilisearch.SearchRequest{
			IndexUID:             CategoryIndex,
			Query:                req.Query,
			Limit:                int64(limit),
			AttributesToRetrieve: []string{"id", "name", "slug"},
		})
	}
	if req.Scope == ScopeAll || req.Scope == ScopeCourses {
		queries = append(queries, &meilisearch.SearchRequest{
			IndexUID:             CourseIndex,
			Query:                req.Query,
			Limit:                int64(limit),
			AttributesToRetrieve: []string{"id", "primaryCode", "name", "department", "creditX10", "aliases", "instructors", "terms", "campus"},
		})
	}
	return queries
}

// searchSingleIndex 对单索引执行一次搜索（multi-search 失败时的回退路径）。
func searchSingleIndex(query *meilisearch.SearchRequest) (*meilisearch.SearchResponse, error) {
	client := meiliconnect.GetClient()
	index := client.Index(query.IndexUID)
	return index.Search(query.Query, &meilisearch.SearchRequest{
		Limit:                query.Limit,
		Offset:               query.Offset,
		Filter:               query.Filter,
		AttributesToRetrieve: query.AttributesToRetrieve,
	})
}

// AggregateSearch 聚合搜索：一次 multi-search 请求，失败时逐索引回退（部分降级）。
func AggregateSearch(req AggregateSearchRequest) (*AggregateSearchResponse, error) {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return &AggregateSearchResponse{
			Topics:     []SearchResult{},
			Users:      []UserSearchResult{},
			Categories: []CategorySearchResult{},
			Courses:    []CourseSearchResult{},
		}, nil
	}
	if len([]rune(req.Query)) > 100 {
		return nil, errors.New("query too long (max 100 characters)")
	}
	if !meiliconnect.IsAvailable() {
		return nil, ErrSearchUnavailable
	}
	req.Scope = NormalizeScope(req.Scope)

	queries := scopeQueries(req)

	resp := &AggregateSearchResponse{
		Topics:     []SearchResult{},
		Users:      []UserSearchResult{},
		Categories: []CategorySearchResult{},
		Courses:    []CourseSearchResult{},
	}

	multiResp, err := meiliconnect.GetClient().MultiSearch(&meilisearch.MultiSearchRequest{
		Federation: nil,
		Queries:    queries,
	})
	if err != nil {
		// multi-search fail-fast：回退逐索引搜索，成功返回、失败记入 failedScopes
		slog.Warn("multi-search failed, falling back to per-index search", "err", err)
		multiResp = nil
	}

	// 按 queries 顺序解析各域结果
	for i, query := range queries {
		var searchResp *meilisearch.SearchResponse
		if multiResp != nil && i < len(multiResp.Results) {
			searchResp = &multiResp.Results[i]
		} else {
			var sErr error
			searchResp, sErr = searchSingleIndex(query)
			if sErr != nil {
				slog.Warn("per-index search failed", "index", query.IndexUID, "err", sErr)
				resp.FailedScopes = append(resp.FailedScopes, query.IndexUID)
				continue
			}
		}
		collectScopeResults(resp, query.IndexUID, searchResp)
	}

	if len(resp.FailedScopes) == len(queries) && len(resp.FailedScopes) > 0 {
		return nil, ErrSearchUnavailable
	}
	return resp, nil
}

// collectScopeResults 把单索引搜索结果解析进对应分区（仅收集 ID，展示数据由 DB 重构）。
func collectScopeResults(resp *AggregateSearchResponse, indexUID string, searchResp *meilisearch.SearchResponse) {
	switch indexUID {
	case TopicIndex:
		results := lo.FilterMap(searchResp.Hits, func(hit meilisearch.Hit, _ int) (SearchResult, bool) {
			item := SearchResult{}
			if err := hit.Decode(&item); err != nil {
				slog.Error("failed to decode topic search hit", "err", err)
				return SearchResult{}, false
			}
			return item, item.ID > 0
		})
		resp.Topics = append(resp.Topics, results...)
		resp.Total = searchResp.EstimatedTotalHits
	case UserIndex:
		type userHit struct {
			ID       uint64 `json:"id"`
			Username string `json:"username"`
		}
		ids := lo.FilterMap(searchResp.Hits, func(hit meilisearch.Hit, _ int) (uint64, bool) {
			item := userHit{}
			if err := hit.Decode(&item); err != nil {
				slog.Error("failed to decode user search hit", "err", err)
				return 0, false
			}
			return item.ID, item.ID > 0
		})
		userMap := users.GetMapByIds(ids)
		resp.Users = lo.FilterMap(ids, func(id uint64, _ int) (UserSearchResult, bool) {
			user, ok := userMap[id]
			if !ok || user == nil {
				return UserSearchResult{}, false
			}
			return UserSearchResult{
				ID:        user.Id,
				Username:  user.Username,
				Nickname:  user.Nickname,
				AvatarURL: user.GetWebAvatarUrl(),
				Bio:       user.Bio,
			}, true
		})
		resp.UsersTotal = searchResp.EstimatedTotalHits
	case CategoryIndex:
		type categoryHit struct {
			ID   uint64 `json:"id"`
			Name string `json:"name"`
		}
		ids := lo.FilterMap(searchResp.Hits, func(hit meilisearch.Hit, _ int) (uint64, bool) {
			item := categoryHit{}
			if err := hit.Decode(&item); err != nil {
				slog.Error("failed to decode category search hit", "err", err)
				return 0, false
			}
			return item.ID, item.ID > 0
		})
		catMap := make(map[uint64]*category.Entity, len(ids))
		for _, cat := range category.All() {
			catMap[cat.Id] = cat
		}
		resp.Categories = lo.FilterMap(ids, func(id uint64, _ int) (CategorySearchResult, bool) {
			cat, ok := catMap[id]
			if !ok || cat == nil {
				return CategorySearchResult{}, false
			}
			return CategorySearchResult{
				ID:    cat.Id,
				Name:  cat.Name,
				Slug:  cat.Slug,
				Icon:  cat.Icon,
				Color: cat.Color,
				Desc:  cat.Desc,
			}, true
		})
		resp.CategoriesTotal = searchResp.EstimatedTotalHits
	case CourseIndex:
		// Meili hit 的 DisplayedAttributes 已包含 aliases/instructors/terms/campus，
		// 直接从 hit 解码展示字段，避免前端这些展示位恒为空。
		type courseHit struct {
			ID          uint64   `json:"id"`
			Aliases     []string `json:"aliases"`
			Instructors []string `json:"instructors"`
			Terms       []string `json:"terms"`
			Campus      []string `json:"campus"`
		}
		hits := lo.FilterMap(searchResp.Hits, func(hit meilisearch.Hit, _ int) (courseHit, bool) {
			item := courseHit{}
			if err := hit.Decode(&item); err != nil {
				slog.Error("failed to decode course search hit", "err", err)
				return courseHit{}, false
			}
			return item, item.ID > 0
		})
		courseMap := course.GetMapByIds(lo.Map(hits, func(h courseHit, _ int) uint64 { return h.ID }))
		resp.Courses = lo.FilterMap(hits, func(h courseHit, _ int) (CourseSearchResult, bool) {
			c, ok := courseMap[h.ID]
			if !ok || c == nil || c.Status != course.StatusVisible {
				return CourseSearchResult{}, false
			}
			return CourseSearchResult{
				ID:          c.Id,
				PrimaryCode: c.PrimaryCode,
				Name:        c.Name,
				Department:  c.Department,
				CreditX10:   c.CreditX10,
				Aliases:     h.Aliases,
				Instructors: h.Instructors,
				Terms:       h.Terms,
				Campus:      h.Campus,
			}, true
		})
		resp.CoursesTotal = searchResp.EstimatedTotalHits
	}
}
