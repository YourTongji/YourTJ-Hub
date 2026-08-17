package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestAgentContractUnauthorizedMatchesCanonicalFixture(t *testing.T) {
	setupAgentForumTestDB(t)
	router := agentForumRouter()
	canonical := contractFixture(t, "auth-required.json")

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		token  string
	}{
		{"me missing token", http.MethodGet, "/api/v1/agent/me", "", ""},
		{"topics missing token", http.MethodGet, "/api/v1/agent/topics", "", ""},
		{"write topic missing token", http.MethodPost, "/api/v1/agent/topics", `{"title":"t","content":"c","categoryId":[1]}`, ""},
		{"posts missing token", http.MethodGet, "/api/v1/agent/topics/1/posts", "", ""},
		{"create post missing token", http.MethodPost, "/api/v1/agent/topics/1/posts", `{"content":"c"}`, ""},
		{"search missing token", http.MethodGet, "/api/v1/agent/search", "", ""},
		{"me wrong token", http.MethodGet, "/api/v1/agent/me", "", "agt_not-a-real-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveAgentRequest(router, tc.method, tc.path, tc.body, tc.token)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401: %s", rec.Code, rec.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), canonical)
		})
	}
}

func TestAgentContractMeSuccess(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "contract-me-agent")

	rec := serveAgentRequest(agentForumRouter(), http.MethodGet, "/api/v1/agent/me", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	response := decodeContractEnvelope(t, rec)
	if response.Code != 0 {
		t.Fatalf("me envelope = %#v, want success", response)
	}
	var me struct {
		AgentId     uint64 `json:"agentId"`
		TokenPrefix string `json:"tokenPrefix"`
	}
	if err := json.Unmarshal(response.Result, &me); err != nil {
		t.Fatalf("decode me result %s: %v", response.Result, err)
	}
	if me.AgentId != agentID || !strings.HasPrefix(me.TokenPrefix, "agt_") {
		t.Fatalf("me = %#v", me)
	}
	if strings.Contains(rec.Body.String(), token) {
		t.Fatal("me response leaks the token")
	}
}

func TestAgentContractTopicWriteAndList(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	categoryID := contractTestID()
	if err := conn.Create(&category.Entity{Id: categoryID, Name: "Contract", Slug: fmt.Sprintf("contract-%d", categoryID)}).Error; err != nil {
		t.Fatalf("create contract category: %v", err)
	}
	agentID, token := createAgentForumAgent(t, conn, "contract-writer-agent")

	t.Run("write topic succeeds with published topic", func(t *testing.T) {
		body := fmt.Sprintf(`{"title":"Contract agent topic","content":"Contract agent topic content is long enough for default posting rules.","categoryId":[%d]}`, categoryID)
		rec := serveAgentRequest(agentForumRouter(), http.MethodPost, "/api/v1/agent/topics", body, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("write topic status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		response := decodeContractEnvelope(t, rec)
		assertFixtureEnvelope(t, response, contractFixture(t, "agent-topic-write-success.json"))
		var topicID uint64
		if err := json.Unmarshal(response.Result, &topicID); err != nil || topicID == 0 {
			t.Fatalf("write topic result = %s, want topic id: %v", response.Result, err)
		}
		topic := topics.Get(topicID)
		if topic.UserId != agentID || topic.Status != 1 {
			t.Fatalf("topic = %#v, want published and owned by agent", topic)
		}
	})

	t.Run("topic list returns the published topic", func(t *testing.T) {
		rec := serveAgentRequest(agentForumRouter(), http.MethodGet, "/api/v1/agent/topics", "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("topic list status = %d, want 200", rec.Code)
		}
		response := decodeContractEnvelope(t, rec)
		if response.Code != 0 {
			t.Fatalf("topic list envelope = %#v", response)
		}
		var list struct {
			List []struct {
				Id            uint64 `json:"id"`
				Status        int8   `json:"status"`
				ProcessStatus int8   `json:"processStatus"`
			} `json:"list"`
		}
		if err := json.Unmarshal(response.Result, &list); err != nil {
			t.Fatalf("decode topic list result %s: %v", response.Result, err)
		}
		if len(list.List) != 1 || list.List[0].Status != 1 || list.List[0].ProcessStatus != 0 {
			t.Fatalf("topic list = %#v, want one published topic", list.List)
		}
	})

	t.Run("malformed write body returns strict 400", func(t *testing.T) {
		_, malformedToken := createAgentForumAgent(t, conn, "contract-malformed-agent")
		rec := serveAgentRequest(agentForumRouter(), http.MethodPost, "/api/v1/agent/topics", "{", malformedToken)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("malformed status = %d, want 400", rec.Code)
		}
		response := decodeContractEnvelope(t, rec)
		if response.MessageCode != "common.request.parseFailed" {
			t.Fatalf("envelope = %#v, want parseFailed", response)
		}
	})
}

func TestAgentContractPostWindowAndCreate(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "contract-reply-agent")
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: 8001, Title: "Contract reply topic", UserId: agentID, Status: 1, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	firstPost := posts.Entity{Id: 8101, TopicId: topic.Id, PostNo: 1, UserId: agentID, Content: "first", CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Update("first_post_id", firstPost.Id).Error; err != nil {
		t.Fatalf("set first post: %v", err)
	}

	t.Run("post window returns the first post", func(t *testing.T) {
		rec := serveAgentRequest(agentForumRouter(), http.MethodGet, "/api/v1/agent/topics/8001/posts", "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("post window status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		response := decodeContractEnvelope(t, rec)
		if response.Code != 0 {
			t.Fatalf("post window envelope = %#v", response)
		}
		var window struct {
			Posts []struct {
				ID                 uint64 `json:"id"`
				TopicID            uint64 `json:"topicId"`
				PostNo             uint64 `json:"postNo"`
				Content            string `json:"content"`
				RenderedContent    string `json:"renderedContent"`
				ProcessStatus      int8   `json:"processStatus"`
				IsHidden           bool   `json:"isHidden"`
				IsAuthorDeleted    bool   `json:"isAuthorDeleted"`
				IsModeratorRemoved bool   `json:"isModeratorRemoved"`
				CanModerate        bool   `json:"canModerate"`
				Author             struct {
					ID        uint64 `json:"id"`
					Username  string `json:"username"`
					AvatarURL string `json:"avatarUrl"`
				} `json:"author"`
				CreatedAt    string `json:"createdAt"`
				IsOwnPost    bool   `json:"isOwnPost"`
				UpdatedAt    string `json:"updatedAt"`
				LikeCount    uint64 `json:"likeCount"`
				IsLiked      bool   `json:"isLiked"`
				IsBookmarked bool   `json:"isBookmarked"`
			} `json:"posts"`
			ReplyTargets []struct {
				ID     uint64 `json:"id"`
				Author struct {
					ID uint64 `json:"id"`
				} `json:"author"`
				Unavailable bool `json:"unavailable"`
			} `json:"replyTargets"`
			HasBefore bool   `json:"hasBefore"`
			HasAfter  bool   `json:"hasAfter"`
			Total     int64  `json:"total"`
			MaxPostNo uint64 `json:"maxPostNo"`
		}
		if err := json.Unmarshal(response.Result, &window); err != nil {
			t.Fatalf("decode post window result %s: %v", response.Result, err)
		}
		if len(window.Posts) != 1 || window.Posts[0].PostNo != 1 || window.MaxPostNo != 1 {
			t.Fatalf("post window = %#v", window)
		}
		if window.Posts[0].ID == 0 || window.Posts[0].TopicID != topic.Id || window.Posts[0].Author.ID == 0 || window.Posts[0].Author.Username == "" {
			t.Fatalf("post window post identity fields missing: %#v", window.Posts[0])
		}
		if window.Posts[0].RenderedContent == "" || window.Posts[0].CreatedAt == "" || window.Posts[0].UpdatedAt == "" {
			t.Fatalf("post window rendered/time fields missing: %#v", window.Posts[0])
		}
		if window.Posts[0].IsAuthorDeleted || window.Posts[0].IsModeratorRemoved || window.Posts[0].IsHidden {
			t.Fatalf("post window first post must not be hidden/removed: %#v", window.Posts[0])
		}
		if window.ReplyTargets == nil || window.HasBefore || window.HasAfter || window.Total != 1 {
			t.Fatalf("post window pagination fields unexpected: %#v", window)
		}
	})

	t.Run("create post succeeds with postNo 2", func(t *testing.T) {
		body := `{"content":"Contract agent reply with enough content for the posting rules.","replyToPostId":8101}`
		rec := serveAgentRequest(agentForumRouter(), http.MethodPost, "/api/v1/agent/topics/8001/posts", body, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("create post status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		response := decodeContractEnvelope(t, rec)
		if response.Code != 0 {
			t.Fatalf("create post envelope = %#v, want success", response)
		}
		var created struct {
			Id              uint64 `json:"id"`
			PostNo          uint64 `json:"postNo"`
			RenderedContent string `json:"renderedContent"`
		}
		if err := json.Unmarshal(response.Result, &created); err != nil {
			t.Fatalf("decode create post result %s: %v", response.Result, err)
		}
		if created.Id == 0 || created.PostNo != 2 || created.RenderedContent == "" {
			t.Fatalf("created = %#v, want id>0 postNo=2 renderedContent", created)
		}
	})

	t.Run("unknown topic keeps business 200 failure envelope", func(t *testing.T) {
		body := `{"content":"Reply to a missing topic with enough content."}`
		rec := serveAgentRequest(agentForumRouter(), http.MethodPost, "/api/v1/agent/topics/999999/posts", body, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200", rec.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "admin-topic-categories-edit-topic-not-found.json"))
	})
}

func TestAgentContractWriteRateLimits(t *testing.T) {
	t.Run("topic write returns 429 with retry metadata", func(t *testing.T) {
		conn := setupAgentForumTestDB(t)
		_, token := createAgentForumAgent(t, conn, "contract-topic-limit-agent")
		router := agentForumRouter()

		first := serveAgentRequest(router, http.MethodPost, "/api/v1/agent/topics", "{", token)
		if first.Code != http.StatusBadRequest {
			t.Fatalf("first topic write status = %d, want 400", first.Code)
		}
		recorder := serveAgentRequest(router, http.MethodPost, "/api/v1/agent/topics", "{", token)
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limited topic write status = %d, want 429", recorder.Code)
		}
		response := decodeContractEnvelope(t, recorder)
		assertFixtureEnvelope(t, response, contractFixture(t, "topic-write-rate-limited.json"))
		assertRetryAfter(t, recorder, response, middleware.RateLimitTopicWrite)
	})

	t.Run("post create returns 429 with retry metadata", func(t *testing.T) {
		conn := setupAgentForumTestDB(t)
		_, token := createAgentForumAgent(t, conn, "contract-post-limit-agent")
		router := agentForumRouter()

		for attempt := 0; attempt < 10; attempt++ {
			recorder := serveAgentRequest(router, http.MethodPost, "/api/v1/agent/topics/1/posts", "{", token)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("attempt %d status = %d, want 400", attempt+1, recorder.Code)
			}
		}
		recorder := serveAgentRequest(router, http.MethodPost, "/api/v1/agent/topics/1/posts", "{", token)
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limited post create status = %d, want 429", recorder.Code)
		}
		response := decodeContractEnvelope(t, recorder)
		assertFixtureEnvelope(t, response, contractFixture(t, "agent-post-create-rate-limited.json"))
		assertRetryAfter(t, recorder, response, middleware.RateLimitPostCreate)
	})
}

func TestAgentContractSearch(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	_, token := createAgentForumAgent(t, conn, "contract-search-agent")

	rec := serveAgentRequest(agentForumRouter(), http.MethodGet, "/api/v1/agent/search?q=campus", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	response := decodeContractEnvelope(t, rec)
	if response.Code != 0 {
		t.Fatalf("search envelope = %#v", response)
	}
	var search struct {
		Query           string `json:"query"`
		Scope           string `json:"scope"`
		Topics          []any  `json:"topics"`
		Users           []any  `json:"users"`
		Categories      []any  `json:"categories"`
		Courses         []any  `json:"courses"`
		Total           int64  `json:"total"`
		UsersTotal      int64  `json:"usersTotal"`
		CategoriesTotal int64  `json:"categoriesTotal"`
		CoursesTotal    int64  `json:"coursesTotal"`
		TotalPages      int    `json:"totalPages"`
		Pagination      struct {
			Page     int    `json:"page"`
			NextPage int    `json:"nextPage"`
			HasNext  bool   `json:"hasNext"`
			NextURL  string `json:"nextUrl"`
		} `json:"pagination"`
		SearchUnavailable bool     `json:"searchUnavailable"`
		FailedScopes      []string `json:"failedScopes"`
	}
	if err := json.Unmarshal(response.Result, &search); err != nil {
		t.Fatalf("decode search result %s: %v", response.Result, err)
	}
	if search.Query != "campus" {
		t.Fatalf("search query = %q, want campus", search.Query)
	}
	for name, value := range map[string][]any{
		"topics":     search.Topics,
		"users":      search.Users,
		"categories": search.Categories,
		"courses":    search.Courses,
	} {
		if value == nil {
			t.Fatalf("search %s must be a present array", name)
		}
	}
	if search.Pagination.Page != 1 {
		t.Fatalf("search pagination = %#v, want page 1", search.Pagination)
	}
}
