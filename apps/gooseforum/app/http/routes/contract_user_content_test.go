package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupUserContentContractTest 在共享 harness（setupHTTPContractTest）之上补齐
// 用户内容/账号生命周期 8 条路由，中间件链与 route4api.go 的生产注册保持一致
// （forumLoginApi：JWTAuthCheck；写操作叠加 CheckWritableAccount + interact 限流）。
func setupUserContentContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&contentDeleteEvent.Entity{},
		&moderationLog.Entity{},
		&optRecord.Entity{},
		&eventNotification.Entity{},
		&reports.Entity{},
	); err != nil {
		t.Fatalf("migrate user content contract tables: %v", err)
	}

	loginAPI := router.Group("/api/forum").Use(middleware.JWTAuthCheck)
	loginAPI.GET("/user/my-content", middleware.NoUpdateUserActivity, UpQueryReq(api.MyContentList))
	loginAPI.GET("/user/deleted-content", middleware.NoUpdateUserActivity, UpQueryReq(api.DeletedContentList))
	loginAPI.POST("/user/content-batch-delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BatchDeleteContent))
	loginAPI.POST("/user/content-restore", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.RestoreContent))
	loginAPI.POST("/user/content-purge", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.PurgeContent))
	loginAPI.POST("/user/content-privacy-erase", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.PrivacyErase))
	loginAPI.POST("/user/content-event", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.ReportContentEvent))
	loginAPI.POST("/user/account-close", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AccountClose))
	return conn, router
}

// createContractDeletedTopic 直接写库造一个 USER_DELETED + RECOVERABLE 的话题。
// deletedAt 为 nil 时保持 deleted_at 为空（墓碑态），使列表响应与时间无关、
// fixture 永久稳定；恢复/永久删除场景传 time.Now() 以满足恢复窗口校验。
func createContractDeletedTopic(t *testing.T, conn *gorm.DB, topicID, authorID uint64, title string, deletedAt *time.Time) {
	t.Helper()
	topic := topics.Entity{
		Id:               topicID,
		Title:            title,
		UserId:           authorID,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		VisibilityStatus: topics.VisibilityUserDeleted,
		RetentionStatus:  topics.RetentionRecoverable,
		CreatedAt:        contractInteractionTime,
		UpdatedAt:        contractInteractionTime,
	}
	if deletedAt != nil {
		topic.DeletedAt = gorm.DeletedAt{Time: *deletedAt, Valid: true}
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create contract deleted topic: %v", err)
	}
}

func TestMyContentListHTTPContract(t *testing.T) {
	t.Run("topics success", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractPublishedTopic(t, conn, 8810101, 8810102, user.Id)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/my-content?contentType=topic", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("my-content topics status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "my-content-topics-success.json"))
	})

	t.Run("posts success", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractPublishedTopic(t, conn, 8810201, 8810203, user.Id)
		createContractReplyPost(t, conn, 8810202, 8810201, user.Id)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/my-content?contentType=post", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("my-content posts status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "my-content-posts-success.json"))
	})

	t.Run("unsupported contentType stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/my-content?contentType=bogus", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("my-content invalid status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "my-content-invalid-params.json"))
	})

	t.Run("malformed cursorId returns strict 400", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/my-content?contentType=topic&cursorId=abc", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("my-content parse failed status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "my-content-parse-failed.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/my-content?contentType=topic", "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("my-content unauthenticated status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "my-content-unauthenticated.json"))
	})
}

func TestDeletedContentListHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// deleted_at 置空的墓碑态：canRestore=false / canPermanent=true 与时间无关。
		createContractDeletedTopic(t, conn, 8810301, user.Id, "Contract deleted topic", nil)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/deleted-content?contentType=topic", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("deleted-content status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "deleted-content-success.json"))
	})

	t.Run("missing contentType stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/deleted-content", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("deleted-content invalid status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "deleted-content-invalid-params.json"))
	})

	t.Run("malformed cursorId returns strict 400", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/deleted-content?contentType=topic&cursorId=abc", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("deleted-content parse failed status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "deleted-content-parse-failed.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/deleted-content?contentType=topic", "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("deleted-content unauthenticated status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "deleted-content-unauthenticated.json"))
	})
}

func TestRestoreContentHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		now := time.Now()
		createContractDeletedTopic(t, conn, 8810501, user.Id, "Contract restorable topic", &now)
		recorder := serveJSON(router, "/api/forum/user/content-restore", `{"contentType":"topic","contentId":8810501}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("content-restore status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-restore-success.json"))
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/user/content-restore", `{"contentType":"topic","contentId":8899999}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("content-restore not found status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-restore-topic-not-found.json"))
	})

	t.Run("active topic is not recoverable", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractPublishedTopic(t, conn, 8810502, 8810503, user.Id)
		recorder := serveJSON(router, "/api/forum/user/content-restore", `{"contentType":"topic","contentId":8810502}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("content-restore not recoverable status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-restore-not-recoverable.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/user/content-restore", `{}`, "content-restore-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/user/content-restore", `{}`, "content-restore-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/user/content-restore", "{", "content-restore-rate-limited.json", middleware.RateLimitInteract)
	})
}

func TestBatchDeleteContentHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractPublishedTopic(t, conn, 8810401, 8810402, user.Id)
		recorder := serveJSON(router, "/api/forum/user/content-batch-delete", `{"contentType":"topic","contentIds":[8810401]}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("batch-delete status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-batch-delete-success.json"))
	})

	t.Run("burst deletion requires force and password confirmation", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// 预置 20 条删除事件打满 10 分钟窗口：再删 1 条即触发二次确认（params.count=21）。
		for i := 0; i < 20; i++ {
			event := contentDeleteEvent.Entity{
				EventType:   string(contentDeleteEvent.EventDeleted),
				ContentType: "topic",
				ContentID:   8820000 + uint64(i),
				ActorID:     user.Id,
				TopicID:     8820000 + uint64(i),
			}
			if err := conn.Create(&event).Error; err != nil {
				t.Fatalf("seed content delete event: %v", err)
			}
		}
		recorder := serveJSON(router, "/api/forum/user/content-batch-delete", `{"contentType":"topic","contentIds":[8810499]}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("batch-delete confirm status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-batch-delete-confirm-required.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/user/content-batch-delete", `{}`, "content-batch-delete-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/user/content-batch-delete", `{}`, "content-batch-delete-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/user/content-batch-delete", "{", "content-batch-delete-rate-limited.json", middleware.RateLimitInteract)
	})
}

func TestPurgeContentHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		now := time.Now()
		createContractDeletedTopic(t, conn, 8810601, user.Id, "Contract purge topic", &now)
		recorder := serveJSON(router, "/api/forum/user/content-purge", `{"contentType":"topic","contentId":8810601}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("content-purge status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-purge-success.json"))
	})

	t.Run("active topic cannot be purged directly", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractPublishedTopic(t, conn, 8810602, 8810603, user.Id)
		recorder := serveJSON(router, "/api/forum/user/content-purge", `{"contentType":"topic","contentId":8810602}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("content-purge not recoverable status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-purge-not-recoverable.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/user/content-purge", `{}`, "content-purge-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/user/content-purge", `{}`, "content-purge-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/user/content-purge", "{", "content-purge-rate-limited.json", middleware.RateLimitInteract)
	})
}

func TestPrivacyEraseContentHTTPContract(t *testing.T) {
	t.Run("success on active own topic", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractPublishedTopic(t, conn, 8810701, 8810702, user.Id)
		recorder := serveJSON(router, "/api/forum/user/content-privacy-erase", `{"contentType":"topic","contentId":8810701}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("privacy-erase status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-privacy-erase-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/user/content-privacy-erase", `{}`, "content-privacy-erase-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/user/content-privacy-erase", `{}`, "content-privacy-erase-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/user/content-privacy-erase", "{", "content-privacy-erase-rate-limited.json", middleware.RateLimitInteract)
	})
}

func TestReportContentEventHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/user/content-event", `{"eventType":"content_delete_confirmed","contentType":"topic","contentId":8810801}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("content-event status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-event-success.json"))
	})

	t.Run("backend-owned event types are rejected", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/user/content-event", `{"eventType":"content_deleted","contentType":"topic","contentId":8810801}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("content-event invalid status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "content-event-invalid-params.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/user/content-event", `{}`, "content-event-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/user/content-event", `{}`, "content-event-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/user/content-event", "{", "content-event-rate-limited.json", middleware.RateLimitInteract)
	})
}

func TestAccountCloseHTTPContract(t *testing.T) {
	t.Run("success revokes every existing session", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := serveJSON(router, "/api/forum/user/account-close", `{"mode":"anonymize","password":"secret123"}`, token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("account-close status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-close-success.json"))
		// 注销后 token_version 自增：旧会话立即失效。
		followUp := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/user/my-content?contentType=topic", "", token)
		if followUp.Code != http.StatusUnauthorized {
			t.Fatalf("post-close session status = %d, want 401", followUp.Code)
		}
	})

	t.Run("wrong password returns invalid credentials", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/user/account-close", `{"mode":"anonymize","password":"wrongpass1"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("account-close wrong password status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-close-invalid-credentials.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupUserContentContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/user/account-close", `{}`, "account-close-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/user/account-close", `{}`, "account-close-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupUserContentContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/user/account-close", "{", "account-close-rate-limited.json", middleware.RateLimitInteract)
	})
}
