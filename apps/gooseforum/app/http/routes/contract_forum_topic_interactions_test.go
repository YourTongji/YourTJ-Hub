package routes

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
)

func TestUpdateTopicStatusHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		body := fmt.Sprintf(`{"topicId":%d,"topicStatus":0}`, topicID)
		recorder := serveJSON(router, "/api/forum/topics/status", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("topic status status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "result-true.json"))
	})

	t.Run("another user cannot delete an author's draft", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		otherUser := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, author.Id)
		if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 0).Error; err != nil {
			t.Fatalf("set topic as draft: %v", err)
		}

		body := fmt.Sprintf(`{"topicId":%d}`, topicID)
		response := serveJSON(router, "/api/forum/topics/delete", body, contractSessionToken(t, otherUser))
		if response.Code != http.StatusOK {
			t.Fatalf("delete draft status = %d, want 200: %s", response.Code, response.Body.String())
		}
		if got := decodeContractEnvelope(t, response).MessageCode; got != "topic.ownerMismatch" {
			t.Fatalf("delete draft messageCode = %q, want topic.ownerMismatch", got)
		}

		drafts, err := topics.GetDraftsByUserId(author.Id, 100)
		if err != nil {
			t.Fatalf("list author's drafts: %v", err)
		}
		if len(drafts) != 1 || drafts[0].Id != topicID {
			t.Fatalf("author's drafts after unauthorized delete = %v, want draft %d", drafts, topicID)
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/topics/status", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/topics/status", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/topics/status",
			`{"topicId":987654321,"topicStatus":0}`,
			"topic-status-rate-limited.json", middleware.RateLimitTopicStatus)
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/topics/status", `{"topicId":987654321,"topicStatus":0}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-topic-categories-edit-topic-not-found.json"))
	})

	t.Run("invalid topicStatus stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/topics/status", `{"topicId":1,"topicStatus":2}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

func TestDeleteTopicByUserHTTPContract(t *testing.T) {
	t.Run("deleting a draft prevents a later editor autosave from restoring it", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 0).Error; err != nil {
			t.Fatalf("set topic as draft: %v", err)
		}

		deleteResponse := serveJSON(router, "/api/forum/topics/delete", fmt.Sprintf(`{"topicId":%d}`, topicID), contractSessionToken(t, user))
		if deleteResponse.Code != http.StatusOK {
			t.Fatalf("delete draft status = %d, want 200: %s", deleteResponse.Code, deleteResponse.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, deleteResponse), contractFixture(t, "result-true.json"))

		// A different tab may still submit a debounced autosave using the deleted ID.
		// The normal scoped topic lookup must reject that update instead of creating a new draft.
		writeBody := fmt.Sprintf(`{"topicId":%d,"title":"Resaved draft","content":"A sufficiently long draft body after deletion.","categoryId":[1],"topicStatus":0,"contentType":3}`, topicID)
		writeResponse := serveJSON(router, "/api/forum/topics/write", writeBody, contractSessionToken(t, user))
		if writeResponse.Code != http.StatusOK {
			t.Fatalf("autosave after deletion status = %d, want 200: %s", writeResponse.Code, writeResponse.Body.String())
		}
		if got := decodeContractEnvelope(t, writeResponse).MessageCode; got != "topic.ownerMismatch" {
			t.Fatalf("autosave after deletion messageCode = %q, want topic.ownerMismatch", got)
		}

		drafts, err := topics.GetDraftsByUserId(user.Id, 100)
		if err != nil {
			t.Fatalf("list drafts after autosave: %v", err)
		}
		for _, draft := range drafts {
			if draft.Id == topicID {
				t.Fatal("deleted draft reappeared after editor autosave")
			}
		}
		if stored := topics.UnscopedGet(topicID); !stored.DeletedAt.Valid {
			t.Fatal("deleted draft lost its soft-delete marker")
		}
	})

	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		body := fmt.Sprintf(`{"topicId":%d}`, topicID)
		recorder := serveJSON(router, "/api/forum/topics/delete", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("delete topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "result-true.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/topics/delete", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/topics/delete", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/topics/delete",
			`{"topicId":987654321}`,
			"account-close-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/topics/delete", `{"topicId":987654321}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-topic-categories-edit-topic-not-found.json"))
	})
}

// contractTopicActionScenarios 覆盖 topics/like、topics/bookmark、topics/watch 三个
// 结构完全一致的互动端点（fixture 前缀不同，行为对称）。
func contractTopicActionScenarios(t *testing.T, path string, fixturePrefix string) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		body := fmt.Sprintf(`{"topicId":%d,"action":1}`, topicID)
		recorder := serveJSON(router, path, body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("success status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "result-true.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, path,
			`{"topicId":987654321,"action":1}`,
			"account-close-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"topicId":987654321,"action":1}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-topic-categories-edit-topic-not-found.json"))
	})

	t.Run("invalid action stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"topicId":1,"action":3}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

func TestLikeTopicHTTPContract(t *testing.T) {
	contractTopicActionScenarios(t, "/api/forum/topics/like", "topic-like")
}

func TestBookmarkTopicHTTPContract(t *testing.T) {
	contractTopicActionScenarios(t, "/api/forum/topics/bookmark", "topic-bookmark")
}

func TestWatchTopicHTTPContract(t *testing.T) {
	contractTopicActionScenarios(t, "/api/forum/topics/watch", "topic-watch")
}

func TestFollowUserHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		userBase := contractTestID()
		follower := createHTTPContractUser(t, conn, userBase)
		target := createHTTPContractUser(t, conn, userBase+1)
		body := fmt.Sprintf(`{"id":%d,"action":1}`, target.Id)
		recorder := serveJSON(router, "/api/forum/follow-user", body, contractSessionToken(t, follower))
		if recorder.Code != http.StatusOK {
			t.Fatalf("follow user status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "follow-user-success.json"))
	})

	t.Run("mutual follow reports both directions across transitions", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		userBase := contractTestID()
		follower := createHTTPContractUser(t, conn, userBase)
		target := createHTTPContractUser(t, conn, userBase+1)
		followBody := func(id uint64, action int) string {
			return fmt.Sprintf(`{"id":%d,"action":%d}`, id, action)
		}

		// 单向关注：follower → target，只有一方向。
		recorder := serveJSON(router, "/api/forum/follow-user", followBody(target.Id, 1), contractSessionToken(t, follower))
		assertFollowUserResult(t, recorder, true, false)

		// target 回关后进入互关。
		recorder = serveJSON(router, "/api/forum/follow-user", followBody(follower.Id, 1), contractSessionToken(t, target))
		assertFollowUserResult(t, recorder, true, true)

		// 幂等重复：状态与响应形状不变。
		recorder = serveJSON(router, "/api/forum/follow-user", followBody(follower.Id, 1), contractSessionToken(t, target))
		assertFollowUserResult(t, recorder, true, true)

		// 单向取消后不再是互关。
		recorder = serveJSON(router, "/api/forum/follow-user", followBody(follower.Id, 2), contractSessionToken(t, target))
		assertFollowUserResult(t, recorder, false, false)
	})

	t.Run("self follow returns a dedicated business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		body := fmt.Sprintf(`{"id":%d,"action":1}`, user.Id)
		recorder := serveJSON(router, "/api/forum/follow-user", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("self follow status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "follow-user-self-follow.json"))
		// 守卫必须先于副作用生效：不得出现自关注行，也不得累加任何一侧计数。
		if followRow := userFollow.GetByUserId(user.Id, user.Id); followRow.Id != 0 {
			t.Fatalf("self follow created a user_follow row: %+v", followRow)
		}
		stats := userStatistics.Get(user.Id)
		if stats.FollowingCount != 0 || stats.FollowerCount != 0 {
			t.Fatalf("self follow moved counters: following=%d follower=%d", stats.FollowingCount, stats.FollowerCount)
		}
	})

	t.Run("self unfollow stays an idempotent success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		body := fmt.Sprintf(`{"id":%d,"action":2}`, user.Id)
		recorder := serveJSON(router, "/api/forum/follow-user", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("self unfollow status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "follow-user-unfollowed.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/follow-user", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/follow-user", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/follow-user",
			`{"id":987654321,"action":1}`,
			"account-close-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown user returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/follow-user", `{"id":987654321,"action":1}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown user status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-save-user-badges-user-not-found.json"))
	})

	t.Run("invalid action stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/follow-user", `{"id":1,"action":3}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

func TestCreateReportHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		userBase := contractTestID()
		reporter := createHTTPContractUser(t, conn, userBase)
		author := createHTTPContractUser(t, conn, userBase+1)
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, author.Id)
		body := fmt.Sprintf(`{"targetType":"topic","targetId":%d,"reason":"spam","note":"contract report note"}`, topicID)
		recorder := serveJSON(router, "/api/forum/report", body, contractSessionToken(t, reporter))
		if recorder.Code != http.StatusOK {
			t.Fatalf("create report status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "result-true.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/report", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/report", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/report",
			`{"targetType":"topic","targetId":987654321,"reason":"spam"}`,
			"account-close-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("invalid target returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		reporter := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/report", `{"targetType":"topic","targetId":987654321,"reason":"spam"}`, contractSessionToken(t, reporter))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid target status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "course-review-report-target-invalid.json"))
	})

	t.Run("duplicate report returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		userBase := contractTestID()
		reporter := createHTTPContractUser(t, conn, userBase)
		author := createHTTPContractUser(t, conn, userBase+1)
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, author.Id)
		token := contractSessionToken(t, reporter)
		body := fmt.Sprintf(`{"targetType":"topic","targetId":%d,"reason":"spam"}`, topicID)
		first := serveJSON(router, "/api/forum/report", body, token)
		if first.Code != http.StatusOK {
			t.Fatalf("first report status = %d, want 200: %s", first.Code, first.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, first), contractFixture(t, "result-true.json"))
		duplicate := serveJSON(router, "/api/forum/report", body, token)
		if duplicate.Code != http.StatusOK {
			t.Fatalf("duplicate report status = %d, want 200: %s", duplicate.Code, duplicate.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, duplicate), contractFixture(t, "course-review-report-duplicate.json"))
	})
}
