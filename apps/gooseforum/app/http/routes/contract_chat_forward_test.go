package routes

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/chatservice"
)

func TestChatForwardMergedSnapshotAndRetry(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	sender := createHTTPContractUser(t, conn, contractTestID())
	actor := createHTTPContractUser(t, conn, contractTestID())
	target := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", sender.Id).Update("avatar_url", "/static/pic/3.webp").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", actor.Id).Update("avatar_url", "/static/pic/6.webp").Error; err != nil {
		t.Fatal(err)
	}
	convID, err := chatservice.SendMessage(sender.Id, actor.Id, "first line", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chatservice.SendMessage(actor.Id, sender.Id, "second line [:sticker:smile:]", 1); err != nil {
		t.Fatal(err)
	}
	var source []messages.Entity
	conn.Where("conv_id = ?", convID).Order("id").Find(&source)
	token := contractSessionToken(t, actor)
	body := fmt.Sprintf(`{"convId":%d,"peerId":%d,"messageIds":[%d,%d],"mode":"merged","clientForwardId":"batch-1"}`, convID, target.Id, source[1].Id, source[0].Id)
	for range 2 {
		rec := serveJSON(router, "/api/forum/chat/forward", body, token)
		var envelope struct {
			Code   int `json:"code"`
			Result struct {
				ConvID     uint64   `json:"convId"`
				MessageIDs []uint64 `json:"messageIds"`
			} `json:"result"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &envelope) != nil || envelope.Code != 0 || len(envelope.Result.MessageIDs) != 1 {
			t.Fatalf("forward: %d %s", rec.Code, rec.Body)
		}
		page, err := chatservice.GetMessages(target.Id, envelope.Result.ConvID, 0, 0, 30)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(page)
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]any
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		item := response["list"].([]any)[0].(map[string]any)
		forward, ok := item["forwarded"].(map[string]any)
		if !ok {
			t.Fatalf("missing forwarded snapshot: %s", data)
		}
		entries := forward["messages"].([]any)
		if len(entries) != 2 || entries[0].(map[string]any)["content"] != "first line" {
			t.Fatalf("order/content: %s", data)
		}
		if entries[0].(map[string]any)["avatarUrl"] != "/static/pic/3.webp" {
			t.Fatalf("missing immutable public avatar: %s", data)
		}
		if entries[1].(map[string]any)["avatarUrl"] != "/static/pic/6.webp" {
			t.Fatalf("second sender's avatar was replaced: %s", data)
		}
		if err := conn.Model(&users.EntityComplete{}).Where("id = ?", sender.Id).Update("avatar_url", "/static/pic/4.webp").Error; err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	conn.Model(&messages.Entity{}).Where("sender_id = ? AND conv_id != ?", actor.Id, convID).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate forwards: %d", count)
	}
}

func TestChatForwardRejectsForeignSourcesAndBlockedRecipients(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	actor := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	target := createHTTPContractUser(t, conn, contractTestID())
	outsider := createHTTPContractUser(t, conn, contractTestID())
	conv, _ := chatservice.SendMessage(peer.Id, actor.Id, "private", 1)
	foreign, _ := chatservice.SendMessage(peer.Id, outsider.Id, "foreign secret", 1)
	var own, other messages.Entity
	conn.Where("conv_id = ?", conv).First(&own)
	conn.Where("conv_id = ?", foreign).First(&other)
	good := chatservice.ForwardRequest{ConvID: conv, PeerID: target.Id, MessageIDs: []uint64{own.Id}, Mode: "merged", ClientForwardID: "negative"}
	for _, test := range []struct {
		name   string
		mutate func(*chatservice.ForwardRequest)
	}{
		{"foreign conversation", func(r *chatservice.ForwardRequest) { r.ConvID = foreign; r.MessageIDs = []uint64{other.Id} }},
		{"mixed sources", func(r *chatservice.ForwardRequest) { r.MessageIDs = []uint64{own.Id, other.Id} }},
		{"duplicate IDs", func(r *chatservice.ForwardRequest) { r.MessageIDs = []uint64{own.Id, own.Id} }},
		{"missing source", func(r *chatservice.ForwardRequest) { r.MessageIDs = []uint64{other.Id + 100000} }},
		{"self target", func(r *chatservice.ForwardRequest) { r.PeerID = actor.Id }},
		{"bad identity", func(r *chatservice.ForwardRequest) { r.ClientForwardID = "invalid/key" }},
		{"individual burst limit", func(r *chatservice.ForwardRequest) {
			r.Mode = "individual"
			r.MessageIDs = make([]uint64, 11)
			for i := range r.MessageIDs {
				r.MessageIDs[i] = uint64(i + 1)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := good
			test.mutate(&r)
			if _, err := chatservice.ForwardMessages(actor.Id, r); err == nil {
				t.Fatal("forward should fail")
			}
		})
	}
	if err := users.SetBlockedUser(target.Id, actor.Id, true); err != nil {
		t.Fatal(err)
	}
	if _, err := chatservice.ForwardMessages(actor.Id, good); err == nil {
		t.Fatal("blocked forward should fail")
	}
	var count int64
	conn.Model(&messages.Entity{}).Where("sender_id = ?", actor.Id).Count(&count)
	if count != 0 {
		t.Fatalf("rejected request leaked %d messages", count)
	}
	raw, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	if rec := serveJSON(router, "/api/forum/chat/forward", string(raw), ""); rec.Code != 401 {
		t.Fatalf("anonymous status %d", rec.Code)
	}
}

func TestChatForwardIndividualAtomicRetryAndNestedBounds(t *testing.T) {
	conn, _ := setupNotificationChatContractTest(t)
	actor := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	target := createHTTPContractUser(t, conn, contractTestID())
	conv, _ := chatservice.SendMessage(peer.Id, actor.Id, "first", 1)
	if _, err := chatservice.SendMessage(actor.Id, peer.Id, "second", 1); err != nil {
		t.Fatal(err)
	}
	var source []messages.Entity
	conn.Where("conv_id = ?", conv).Order("id").Find(&source)
	req := chatservice.ForwardRequest{ConvID: conv, PeerID: target.Id, MessageIDs: []uint64{source[1].Id, source[0].Id}, Mode: "individual", ClientForwardID: "atomic"}
	// An insert failure in the second message must roll back the first message,
	// the new conversation, memberships, and counters together.
	conn.Exec(`CREATE TRIGGER fail_forward BEFORE INSERT ON messages WHEN NEW.content = 'second' BEGIN SELECT RAISE(ABORT, 'injected failure'); END`)
	if _, err := chatservice.ForwardMessages(actor.Id, req); err == nil {
		t.Fatal("expected insert failure")
	}
	var leaked int64
	conn.Model(&messages.Entity{}).Where("sender_id = ? AND conv_id != ?", actor.Id, conv).Count(&leaked)
	if leaked != 0 {
		t.Fatal("partial delivery escaped rollback")
	}
	conn.Exec(`DROP TRIGGER fail_forward`)
	first, err := chatservice.ForwardMessages(actor.Id, req)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := chatservice.ForwardMessages(actor.Id, req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, retry) {
		t.Fatalf("retry changed identity: %#v %#v", first, retry)
	}
	var sent []messages.Entity
	conn.Where("conv_id = ?", first.ConvID).Order("id").Find(&sent)
	if len(sent) != 2 || sent[0].Content != "first" || sent[1].Content != "second" {
		t.Fatalf("wrong individual copy: %#v", sent)
	}
	var membership imUserChatConfigs.Entity
	conn.Where("user_id = ? AND conv_id = ?", target.Id, first.ConvID).First(&membership)
	if membership.UnreadCount != 2 {
		t.Fatalf("retry inflated unread count: %d", membership.UnreadCount)
	}
	req.Mode = "merged"
	req.ClientForwardID = "nested"
	merged, err := chatservice.ForwardMessages(actor.Id, req)
	if err != nil {
		t.Fatal(err)
	}
	again, err := chatservice.ForwardMessages(target.Id, chatservice.ForwardRequest{ConvID: merged.ConvID, PeerID: peer.Id, MessageIDs: merged.MessageIDs, Mode: "merged", ClientForwardID: "flatten"})
	if err != nil {
		t.Fatal(err)
	}
	var nested messages.Entity
	conn.First(&nested, again.MessageIDs[0])
	bundle := messages.ParseForward(nested.Content)
	if bundle == nil || len(bundle.Messages) != 1 || bundle.Messages[0].MsgType != messages.ForwardType {
		t.Fatalf("nested history card was lost: %s", nested.Content)
	}
	var original messages.Entity
	conn.First(&original, merged.MessageIDs[0])
	if !reflect.DeepEqual(bundle.Messages[0].Forwarded, messages.ParseForward(original.Content)) {
		t.Fatal("nested content or sender metadata changed")
	}
	level3, err := chatservice.ForwardMessages(peer.Id, chatservice.ForwardRequest{ConvID: again.ConvID, PeerID: actor.Id, MessageIDs: again.MessageIDs, Mode: "merged", ClientForwardID: "level3"})
	if err != nil {
		t.Fatal(err)
	}
	level4, err := chatservice.ForwardMessages(actor.Id, chatservice.ForwardRequest{ConvID: level3.ConvID, PeerID: target.Id, MessageIDs: level3.MessageIDs, Mode: "merged", ClientForwardID: "level4"})
	if err != nil {
		t.Fatal(err)
	}
	tooDeep := chatservice.ForwardRequest{ConvID: level4.ConvID, PeerID: peer.Id, MessageIDs: level4.MessageIDs, Mode: "merged", ClientForwardID: "level5"}
	var before, after int64
	conn.Model(&messages.Entity{}).Count(&before)
	if _, err := chatservice.ForwardMessages(target.Id, tooDeep); err == nil {
		t.Fatal("excessive nested depth delivered")
	}
	conn.Model(&messages.Entity{}).Count(&after)
	if before != after {
		t.Fatal("rejected nested forward left partial messages")
	}
	// Individual forwarding copies a depth-limit card without wrapping it.
	tooDeep.Mode = "individual"
	copy, err := chatservice.ForwardMessages(target.Id, tooDeep)
	if err != nil {
		t.Fatal(err)
	}
	var deepSource, deepCopy messages.Entity
	conn.First(&deepSource, level4.MessageIDs[0])
	conn.First(&deepCopy, copy.MessageIDs[0])
	if deepSource.Content != deepCopy.Content {
		t.Fatal("individual forwarding changed the nested snapshot")
	}
	// Changing the source author's profile must not mutate an acknowledged copy.
	conn.Model(&peer).Where("id = ?", peer.Id).Update("nickname", strings.Repeat("x", messages.MaxForwardBytes))
	replay, err := chatservice.ForwardMessages(actor.Id, req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged, replay) {
		t.Fatal("profile change changed acknowledged message")
	}
}

func TestChatForwardIndividualPreservesLargeMessages(t *testing.T) {
	card, err := (&messages.ForwardedBundle{Version: 1, Messages: []messages.ForwardedEntry{{
		SenderName: "Original sender", Content: strings.Repeat("x", 40*1024), MsgType: 1,
	}}}).Encode()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		content []string
		msgType int8
	}{
		{"ordinary message over snapshot limit", []string{strings.Repeat("字", messages.MaxForwardBytes/3+1)}, 1},
		{"valid cards over combined snapshot limit", []string{card, card}, messages.ForwardType},
	} {
		t.Run(test.name, func(t *testing.T) {
			conn, router := setupNotificationChatContractTest(t)
			actor := createHTTPContractUser(t, conn, contractTestID())
			peer := createHTTPContractUser(t, conn, contractTestID())
			target := createHTTPContractUser(t, conn, contractTestID())
			var convID uint64
			for _, content := range test.content {
				convID, err = chatservice.SendMessage(peer.Id, actor.Id, content, test.msgType)
				if err != nil {
					t.Fatal(err)
				}
			}
			var source []messages.Entity
			if err := conn.Where("conv_id = ?", convID).Order("id").Find(&source).Error; err != nil {
				t.Fatal(err)
			}
			ids := make([]uint64, len(source))
			for i, message := range source {
				ids[i] = message.Id
			}
			request := chatservice.ForwardRequest{ConvID: convID, PeerID: target.Id, MessageIDs: ids, Mode: "individual", ClientForwardID: "large-copy"}
			token := contractSessionToken(t, actor)
			var first chatservice.ForwardResult
			for attempt := range 2 {
				body, err := json.Marshal(request)
				if err != nil {
					t.Fatal(err)
				}
				rec := serveJSON(router, "/api/forum/chat/forward", string(body), token)
				var envelope struct {
					Code   int                       `json:"code"`
					Result chatservice.ForwardResult `json:"result"`
				}
				if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &envelope) != nil || envelope.Code != 0 || len(envelope.Result.MessageIDs) != len(source) {
					t.Fatalf("individual forward: %d %s", rec.Code, rec.Body)
				}
				if attempt == 0 {
					first = envelope.Result
				} else if !reflect.DeepEqual(first, envelope.Result) {
					t.Fatal("retry changed delivery identity")
				}
			}
			var sent []messages.Entity
			if err := conn.Where("conv_id = ?", first.ConvID).Order("id").Find(&sent).Error; err != nil {
				t.Fatal(err)
			}
			if len(sent) != len(source) {
				t.Fatalf("expected %d copies, got %d", len(source), len(sent))
			}
			for i, message := range sent {
				if message.Content != source[i].Content || message.MsgType != source[i].MsgType {
					t.Fatal("individual copy changed content or type")
				}
			}
			request.Mode = "merged"
			request.ClientForwardID = "oversized-merge"
			if _, err := chatservice.ForwardMessages(actor.Id, request); err == nil {
				t.Fatal("oversized merged snapshot should fail")
			}
			var count int64
			if err := conn.Model(&messages.Entity{}).Where("conv_id = ?", first.ConvID).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != int64(len(source)) {
				t.Fatal("rejected merge left a message behind")
			}
		})
	}
}

func TestChatForwardRejectsUnsupportedSourceTypes(t *testing.T) {
	conn, _ := setupNotificationChatContractTest(t)
	actor := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	target := createHTTPContractUser(t, conn, contractTestID())
	for _, msgType := range []int8{0, messages.ForwardType, 5} {
		// Simulate a corrupt or unsupported stored message, including an invalid
		// history card. Individual mode must still validate the original card.
		convID, err := chatservice.SendMessage(peer.Id, actor.Id, "invalid snapshot", msgType)
		if err != nil {
			t.Fatal(err)
		}
		var source messages.Entity
		if err := conn.Where("conv_id = ?", convID).Last(&source).Error; err != nil {
			t.Fatal(err)
		}
		// GORM substitutes the model default for zero-valued types on insert.
		if err := conn.Model(&source).Update("msg_type", msgType).Error; err != nil {
			t.Fatal(err)
		}
		for _, mode := range []string{"individual", "merged"} {
			request := chatservice.ForwardRequest{ConvID: convID, PeerID: target.Id, MessageIDs: []uint64{source.Id}, Mode: mode, ClientForwardID: "invalid-copy"}
			if _, err := chatservice.ForwardMessages(actor.Id, request); err == nil {
				t.Fatalf("%s forwarded unsupported type %d", mode, msgType)
			}
		}
	}
	var count int64
	if err := conn.Model(&messages.Entity{}).Where("sender_id = ?", actor.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("unsupported source left a forwarded copy")
	}
}
