package routes

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-status-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/topics/status", `{}`, "topic-status-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/topics/status", `{}`, "topic-status-forbidden.json")
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-status-topic-not-found.json"))
	})

	t.Run("invalid topicStatus stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/topics/status", `{"topicId":1,"topicStatus":2}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-status-invalid-params.json"))
	})
}

func TestDeleteTopicByUserHTTPContract(t *testing.T) {
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-delete-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/topics/delete", `{}`, "topic-delete-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/topics/delete", `{}`, "topic-delete-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/topics/delete",
			`{"topicId":987654321}`,
			"topic-delete-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/topics/delete", `{"topicId":987654321}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-delete-topic-not-found.json"))
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, fixturePrefix+"-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, fixturePrefix+"-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, path,
			`{"topicId":987654321,"action":1}`,
			fixturePrefix+"-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"topicId":987654321,"action":1}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-topic-not-found.json"))
	})

	t.Run("invalid action stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"topicId":1,"action":3}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-invalid-params.json"))
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

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/follow-user", `{}`, "follow-user-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/follow-user", `{}`, "follow-user-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/follow-user",
			`{"id":987654321,"action":1}`,
			"follow-user-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown user returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/follow-user", `{"id":987654321,"action":1}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown user status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "follow-user-user-not-found.json"))
	})

	t.Run("invalid action stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/follow-user", `{"id":1,"action":3}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "follow-user-invalid-params.json"))
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "report-create-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/report", `{}`, "report-create-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/report", `{}`, "report-create-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/report",
			`{"targetType":"topic","targetId":987654321,"reason":"spam"}`,
			"report-create-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("invalid target returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		reporter := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/report", `{"targetType":"topic","targetId":987654321,"reason":"spam"}`, contractSessionToken(t, reporter))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid target status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "report-create-target-invalid.json"))
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, first), contractFixture(t, "report-create-success.json"))
		duplicate := serveJSON(router, "/api/forum/report", body, token)
		if duplicate.Code != http.StatusOK {
			t.Fatalf("duplicate report status = %d, want 200: %s", duplicate.Code, duplicate.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, duplicate), contractFixture(t, "report-create-duplicate.json"))
	})
}
