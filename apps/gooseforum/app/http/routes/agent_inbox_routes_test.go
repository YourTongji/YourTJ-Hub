package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
)

func createInboxRowForTest(t *testing.T, agentID, topicID, postID uint64, eventType string, status uint8) uint64 {
	t.Helper()
	row := agentInbox.Entity{
		AgentId:        agentID,
		TopicId:        topicID,
		PostId:         postID,
		EventType:      eventType,
		ActorId:        1,
		ContentPreview: "preview text",
		Status:         status,
		DeliveryStatus: agentInbox.DeliveryPending,
	}
	if err := db.Connect().Create(&row).Error; err != nil {
		t.Fatalf("create inbox row: %v", err)
	}
	return row.Id
}

func TestAgentInboxRoutesRequireBearer(t *testing.T) {
	setupAgentForumTestDB(t)
	router := agentForumRouter()

	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{"list missing token", http.MethodGet, "/api/v1/agent/inbox"},
		{"detail missing token", http.MethodGet, "/api/v1/agent/inbox/1"},
		{"read missing token", http.MethodPost, "/api/v1/agent/inbox/1/read"},
		{"read-all missing token", http.MethodPost, "/api/v1/agent/inbox/read-all"},
		{"delete missing token", http.MethodDelete, "/api/v1/agent/inbox/1"},
		{"clear missing token", http.MethodDelete, "/api/v1/agent/inbox"},
		{"list wrong token", http.MethodGet, "/api/v1/agent/inbox"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := ""
			if tc.name == "list wrong token" {
				token = "agt_not-a-real-token"
			}
			rec, envelope := agentRequest(t, router, tc.method, tc.path, "", token)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401: %s", rec.Code, rec.Body.String())
			}
			if envelope.Code != 1 || envelope.MessageCode != "auth.required" {
				t.Fatalf("envelope = %#v, want canonical auth.required", envelope)
			}
		})
	}
}

func TestAgentInboxListAndUnreadFilter(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "inbox-list-agent")

	unreadID := createInboxRowForTest(t, agentID, 1001, 9001, agentInbox.EventTypeTopicPublished, agentInbox.StatusUnread)
	createInboxRowForTest(t, agentID, 1002, 9002, agentInbox.EventTypePostCreated, agentInbox.StatusRead)

	t.Run("all statuses", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/inbox", "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("list failed: %s", rec.Body.String())
		}
		var result struct {
			List     []json.RawMessage `json:"list"`
			Page     int               `json:"page"`
			PageSize int               `json:"pageSize"`
			HasNext  bool              `json:"hasNext"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil {
			t.Fatalf("decode list result: %v", err)
		}
		if len(result.List) != 2 {
			t.Fatalf("list length = %d, want 2", len(result.List))
		}
		if result.Page != 1 || result.PageSize != 10 || result.HasNext {
			t.Fatalf("page meta = %#v", result)
		}
	})

	t.Run("unread filter", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/inbox?status=unread", "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("unread list failed: %s", rec.Body.String())
		}
		var result struct {
			List []struct {
				Id     uint64 `json:"id"`
				Status uint8  `json:"status"`
			} `json:"list"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil {
			t.Fatalf("decode unread result: %v", err)
		}
		if len(result.List) != 1 || result.List[0].Id != unreadID || result.List[0].Status != agentInbox.StatusUnread {
			t.Fatalf("unread list = %#v", result.List)
		}
	})

	t.Run("invalid status value behaves as all", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/inbox?status=weird", "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("list failed: %s", rec.Body.String())
		}
		var result struct {
			List []json.RawMessage `json:"list"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil {
			t.Fatalf("decode result: %v", err)
		}
		if len(result.List) != 2 {
			t.Fatalf("invalid status list length = %d, want all 2", len(result.List))
		}
	})
}

func TestAgentInboxDetailOwnershipAndNotFound(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "inbox-detail-agent")
	_, otherToken := createAgentForumAgent(t, conn, "inbox-detail-other")

	ownedID := createInboxRowForTest(t, agentID, 1003, 9003, agentInbox.EventTypeTopicUpdated, agentInbox.StatusUnread)

	t.Run("owned detail succeeds", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/inbox/"+strconv.FormatUint(ownedID, 10), "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("detail failed: %s", rec.Body.String())
		}
		var item struct {
			Id        uint64 `json:"id"`
			EventType string `json:"eventType"`
			Status    uint8  `json:"status"`
		}
		if err := json.Unmarshal(envelope.Result, &item); err != nil {
			t.Fatalf("decode detail: %v", err)
		}
		if item.Id != ownedID || item.EventType != agentInbox.EventTypeTopicUpdated || item.Status != agentInbox.StatusUnread {
			t.Fatalf("detail = %#v", item)
		}
	})

	t.Run("cross-agent detail returns notFound", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/inbox/"+strconv.FormatUint(ownedID, 10), "", otherToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("cross-agent status = %d, want business 200", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "agent.inbox.notFound" {
			t.Fatalf("envelope = %#v, want agent.inbox.notFound", envelope)
		}
	})

	t.Run("nonexistent id returns notFound", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/inbox/999999", "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("nonexistent status = %d, want business 200", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "agent.inbox.notFound" {
			t.Fatalf("envelope = %#v, want agent.inbox.notFound", envelope)
		}
	})
}

func TestAgentInboxReadAndReadAll(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "inbox-read-agent")

	firstID := createInboxRowForTest(t, agentID, 1004, 9004, agentInbox.EventTypeTopicPublished, agentInbox.StatusUnread)
	secondID := createInboxRowForTest(t, agentID, 1005, 9005, agentInbox.EventTypePostCreated, agentInbox.StatusUnread)

	t.Run("read marks one row read", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/inbox/"+strconv.FormatUint(firstID, 10)+"/read", "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("read failed: %s", rec.Body.String())
		}
		row := agentInbox.GetByID(firstID)
		if row == nil || row.Status != agentInbox.StatusRead || row.ReadAt == nil {
			t.Fatalf("row after read = %#v", row)
		}
	})

	t.Run("read again is idempotent", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/inbox/"+strconv.FormatUint(firstID, 10)+"/read", "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("idempotent read failed: %s", rec.Body.String())
		}
	})

	t.Run("read nonexistent returns notFound", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/inbox/999999/read", "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("read nonexistent status = %d, want business 200", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "agent.inbox.notFound" {
			t.Fatalf("envelope = %#v, want agent.inbox.notFound", envelope)
		}
	})

	t.Run("read-all marks every row read", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/inbox/read-all", "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("read-all failed: %s", rec.Body.String())
		}
		for _, id := range []uint64{firstID, secondID} {
			row := agentInbox.GetByID(id)
			if row == nil || row.Status != agentInbox.StatusRead {
				t.Fatalf("row %d after read-all = %#v", id, row)
			}
		}
	})
}

func TestAgentInboxDeleteAndClear(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "inbox-delete-agent")
	_, otherToken := createAgentForumAgent(t, conn, "inbox-delete-other")

	ownedID := createInboxRowForTest(t, agentID, 1006, 9006, agentInbox.EventTypeTopicPublished, agentInbox.StatusUnread)
	createInboxRowForTest(t, agentID, 1007, 9007, agentInbox.EventTypePostCreated, agentInbox.StatusUnread)

	t.Run("cross-agent delete returns notFound", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodDelete, "/api/v1/agent/inbox/"+strconv.FormatUint(ownedID, 10), "", otherToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("cross-agent delete status = %d, want business 200", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "agent.inbox.notFound" {
			t.Fatalf("envelope = %#v, want agent.inbox.notFound", envelope)
		}
		if agentInbox.GetByID(ownedID) == nil {
			t.Fatal("cross-agent delete must not remove the row")
		}
	})

	t.Run("delete removes owned row", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodDelete, "/api/v1/agent/inbox/"+strconv.FormatUint(ownedID, 10), "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("delete failed: %s", rec.Body.String())
		}
		if agentInbox.GetByID(ownedID) != nil {
			t.Fatal("row still exists after delete")
		}
	})

	t.Run("delete nonexistent returns notFound", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodDelete, "/api/v1/agent/inbox/999999", "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("delete nonexistent status = %d, want business 200", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "agent.inbox.notFound" {
			t.Fatalf("envelope = %#v, want agent.inbox.notFound", envelope)
		}
	})

	t.Run("clear empties inbox", func(t *testing.T) {
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodDelete, "/api/v1/agent/inbox", "", token)
		if rec.Code != http.StatusOK || envelope.Code != 0 {
			t.Fatalf("clear failed: %s", rec.Body.String())
		}
		var count int64
		db.Connect().Model(&agentInbox.Entity{}).Where("agent_id = ?", agentID).Count(&count)
		if count != 0 {
			t.Fatalf("inbox rows after clear = %d, want 0", count)
		}
	})
}

func TestAgentInboxNonNumericPathReturns400(t *testing.T) {
	setupAgentForumTestDB(t)
	_, token := createAgentForumAgent(t, db.Connect(), "inbox-strict-agent")

	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/inbox/abc", "", token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if envelope.Code != 1 || envelope.MessageCode != "common.request.parseFailed" {
		t.Fatalf("envelope = %#v, want parseFailed", envelope)
	}
}
