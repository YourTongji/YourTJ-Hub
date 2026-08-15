package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupNotificationChatContractTest 在共享 harness（setupHTTPContractTest）之上
// 补齐通知 4 条 + 聊天 3 条路由，中间件链与 route4api.go 的生产注册保持一致
// （chat 组继承 forumApi 经 Use 挂载的 JWTAuthCheck 并再叠一层，与生产相同）。
func setupNotificationChatContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&eventNotification.Entity{},
		&imConversations.Entity{},
		&imUserChatConfigs.Entity{},
		&messages.Entity{},
	); err != nil {
		t.Fatalf("migrate notification/chat contract tables: %v", err)
	}

	forumAPI := router.Group("/api/forum")
	forumLoginAPI := forumAPI.Use(middleware.JWTAuthCheck)
	forumLoginAPI.GET("/unread-status", middleware.NoUpdateUserActivity, UpButterReq(api.GetUnreadStatus))
	forumLoginAPI.GET("/notifications", middleware.NoUpdateUserActivity, UpQueryReq(api.NotificationList))
	forumLoginAPI.POST("/notification/mark-read", middleware.CheckWritableAccount, UpButterReq(api.MarkAsRead))
	forumLoginAPI.POST("/notification/mark-all-read", middleware.CheckWritableAccount, UpButterReq(api.MarkAllAsRead))

	chatAPI := forumAPI.Group("/chat", middleware.JWTAuthCheck)
	chatAPI.POST("/send", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitMessageSend), UpButterReq(api.SendMessage))
	chatAPI.POST("/messages", UpButterReq(api.GetMessages))
	chatAPI.POST("/mark-read", middleware.CheckWritableAccount, UpButterReq(api.MarkChatRead))
	return conn, router
}

// createContractNotification 直接写库造一条通知（显式 id/CreatedAt 供确定性 fixture 断言）。
func createContractNotification(t *testing.T, conn *gorm.DB, id, userID uint64, eventType string, isRead bool, payload eventNotification.NotificationPayload, createdAt time.Time) {
	t.Helper()
	entity := eventNotification.Entity{
		Id:        id,
		UserId:    userID,
		TopicID:   payload.TopicId,
		Payload:   payload,
		EventType: eventType,
		IsRead:    isRead,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("create contract notification: %v", err)
	}
}

// createContractConversation 直接写库造一个单聊会话及双向成员配置。
func createContractConversation(t *testing.T, conn *gorm.DB, convID, userID, peerID uint64) {
	t.Helper()
	if err := conn.Create(&imConversations.Entity{Id: convID, Type: 1, LastMsgTime: time.Now()}).Error; err != nil {
		t.Fatalf("create contract conversation: %v", err)
	}
	for _, pair := range [2][2]uint64{{userID, peerID}, {peerID, userID}} {
		if err := conn.Create(&imUserChatConfigs.Entity{UserId: pair[0], PeerId: pair[1], ConvId: convID, UpdatedAt: time.Now()}).Error; err != nil {
			t.Fatalf("create contract chat config: %v", err)
		}
	}
}

// createContractMessage 直接写库造一条聊天消息（显式 id/CreatedAt 供确定性 fixture 断言）。
func createContractMessage(t *testing.T, conn *gorm.DB, id, convID, senderID uint64, content string, isRead int, createdAt time.Time) {
	t.Helper()
	entity := messages.Entity{
		Id:        id,
		ConvId:    convID,
		SenderId:  senderID,
		Content:   content,
		MsgType:   1,
		IsRead:    isRead,
		CreatedAt: createdAt,
	}
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("create contract message: %v", err)
	}
}

func TestUnreadStatusHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractNotification(t, conn, contractTestID(), user.Id, eventNotification.EventTypePostReply, false,
			eventNotification.NotificationPayload{Title: "契约未读通知", ActorId: user.Id}, time.Now())
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/unread-status", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unread status status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "unread-status-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupNotificationChatContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/unread-status", "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "unread-status-unauthenticated.json"))
	})
}

func TestNotificationListHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// 固定 id/时间戳/payload；actor/topic 名称直接存于 payload（hydrate 为
		// best-effort 补强，无对应 users/topics 行时保留存储值），使响应与确定性
		// fixture 精确一致。
		createContractNotification(t, conn, 2048, user.Id, eventNotification.EventTypePostReply, false,
			eventNotification.NotificationPayload{
				Title:      "你的回复收到了新回复",
				Content:    "tongji_user 回复了你的评论",
				ActorId:    1024,
				ActorName:  "tongji_user",
				TopicId:    512,
				TopicTitle: "期中复习资料汇总",
				PostId:     4096,
			},
			time.Date(2026, 8, 15, 10, 20, 30, 0, time.UTC))
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/notifications", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("notifications status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "notifications-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupNotificationChatContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/notifications", "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "notifications-unauthenticated.json"))
	})

	t.Run("unknown filter stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/notifications?filter=bogus", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid filter status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "notifications-invalid-filter.json"))
	})

	t.Run("non-numeric cursor returns strict 400", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/notifications?cursor=abc", "", contractSessionToken(t, user))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("parse failed status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "notifications-parse-failed.json"))
	})
}

func TestNotificationMarkReadHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		notificationID := contractTestID()
		createContractNotification(t, conn, notificationID, user.Id, eventNotification.EventTypePostReply, false,
			eventNotification.NotificationPayload{Title: "契约待读通知", ActorId: user.Id}, time.Now())
		body := fmt.Sprintf(`{"notificationId":%d}`, notificationID)
		recorder := serveJSON(router, "/api/forum/notification/mark-read", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("mark read status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "notification-mark-read-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupNotificationChatContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/notification/mark-read", `{}`, "notification-mark-read-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/notification/mark-read", `{}`, "notification-mark-read-forbidden.json")
	})
}

func TestNotificationMarkAllReadHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		createContractNotification(t, conn, contractTestID(), user.Id, eventNotification.EventTypePostReply, false,
			eventNotification.NotificationPayload{Title: "契约待读通知", ActorId: user.Id}, time.Now())
		recorder := serveJSON(router, "/api/forum/notification/mark-all-read", `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("mark all read status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "notification-mark-all-read-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupNotificationChatContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/notification/mark-all-read", `{}`, "notification-mark-all-read-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/notification/mark-all-read", `{}`, "notification-mark-all-read-forbidden.json")
	})
}

func TestChatSendHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		peer := createHTTPContractUser(t, conn, contractTestID())
		body := fmt.Sprintf(`{"peerId":%d,"content":"契约私信内容","msgType":1}`, peer.Id)
		recorder := serveJSON(router, "/api/forum/chat/send", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("chat send status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		fixture := contractFixture(t, "chat-send-success.json")
		if response.Code != fixture.Code {
			t.Fatalf("chat send code = %d, want fixture code %d", response.Code, fixture.Code)
		}
		// result.convId 为共享测试库自增 id（随用例执行变化），断言为正数即可。
		var result struct {
			ConvId uint64 `json:"convId"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			t.Fatalf("decode chat send result %s: %v", response.Result, err)
		}
		if result.ConvId == 0 {
			t.Fatalf("convId = %d, want positive conversation id", result.ConvId)
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupNotificationChatContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/chat/send", `{}`, "chat-send-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/chat/send", `{}`, "chat-send-forbidden.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/chat/send",
			`{"peerId":1,"content":"Contract rate limit probe."}`,
			"chat-send-rate-limited.json", middleware.RateLimitMessageSend)
	})

	t.Run("missing msgType stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/chat/send", `{"peerId":1,"content":"契约私信内容"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "chat-send-invalid-params.json"))
	})
}

func TestChatMessagesHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		// 固定 viewer(2048)/peer(1024)/conv(7701)/消息 id 与时间戳，使游标分页
		// 响应与确定性 fixture 精确一致（viewer=2048 视角：9002 为 isSelf）。
		viewer := createHTTPContractUser(t, conn, 2048)
		createContractConversation(t, conn, 7701, viewer.Id, 1024)
		createContractMessage(t, conn, 9001, 7701, 1024, "你好，请问资料还能发我一份吗？", 1,
			time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC))
		createContractMessage(t, conn, 9002, 7701, viewer.Id, "可以，稍后传你", 0,
			time.Date(2026, 8, 15, 9, 1, 12, 0, time.UTC))
		recorder := serveJSON(router, "/api/forum/chat/messages", `{"convId":7701}`, contractSessionToken(t, viewer))
		if recorder.Code != http.StatusOK {
			t.Fatalf("chat messages status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "chat-messages-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupNotificationChatContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/chat/messages", `{}`, "chat-messages-unauthenticated.json")
	})

	t.Run("non-member conversation returns business failure", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// user 不是该会话成员：越权读取他人会话被拒绝。
		recorder := serveJSON(router, "/api/forum/chat/messages", `{"convId":987654321}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("non-member status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "chat-messages-failed.json"))
	})
}

func TestChatMarkReadHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		peer := createHTTPContractUser(t, conn, contractTestID())
		convID := contractTestID()
		createContractConversation(t, conn, convID, user.Id, peer.Id)
		body := fmt.Sprintf(`{"convId":%d}`, convID)
		recorder := serveJSON(router, "/api/forum/chat/mark-read", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("chat mark read status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "chat-mark-read-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupNotificationChatContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/chat/mark-read", `{}`, "chat-mark-read-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/chat/mark-read", `{}`, "chat-mark-read-forbidden.json")
	})

	t.Run("non-member conversation returns business failure", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// user 不是该会话成员：越权翻转他人会话已读状态被拒绝（issue #111，CWE-639）。
		recorder := serveJSON(router, "/api/forum/chat/mark-read", `{"convId":987654321}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("non-member status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "chat-mark-read-failed.json"))
	})
}
