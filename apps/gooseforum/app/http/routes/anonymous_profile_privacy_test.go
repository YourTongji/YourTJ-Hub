package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/gin-gonic/gin"
)

func TestAnonymousProfileContentPrivacyHTTP(t *testing.T) {
	conn, _ := setupAccountContractTest(t)
	if err := conn.AutoMigrate(&identity.Persona{}, &identity.Binding{}, &identity.Quota{}, &identity.Batch{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterByGin(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	uid := fmt.Sprintf("%032x", contractTestID())
	firstID := contractTestID()
	topic := topics.Entity{Id: contractTestID(), FirstPostId: firstID, UserId: owner.Id, PersonaUID: uid, Title: "private profile topic", Status: 1}
	for _, row := range []any{
		&identity.Persona{UID: uid, Name: "躲进云里的猫", AvatarSeed: "private-seed"},
		&identity.Binding{OwnerID: owner.Id, PersonaUID: uid}, &topic,
		&posts.Entity{Id: firstID, TopicId: topic.Id, PostNo: 1, UserId: owner.Id, PersonaUID: uid, Content: "private profile first post"},
		&posts.Entity{Id: firstID + 1, TopicId: topic.Id, PostNo: 2, UserId: owner.Id, PersonaUID: uid, Content: "private profile reply"},
	} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatalf("create %#v: %v", row, err)
		}
		t.Cleanup(func() { conn.Delete(row) })
	}
	profile := func(token string, page int) (forum.AnonymousProfileProps, string) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/a/%s?page=%d&tab=replies", uid, page), nil)
		request.Header.Set("X-Goose-Page", "true")
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, request)
		var payload struct {
			Props forum.AnonymousProfileProps `json:"props"`
		}
		if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Props, rec.Body.String()
	}
	path := "/api/forum/anonymous/privacy"
	token := contractSessionToken(t, owner)
	for _, body := range []string{`{"showContent":false}`, `{"showContent":true}`} {
		if rec := serveJSON(router, path, body, ""); rec.Code != http.StatusUnauthorized {
			t.Fatal("unauthenticated privacy change", rec.Code)
		}
		if res := decodeContractEnvelope(t, serveJSON(router, path, body, contractSessionToken(t, other))); res.Code == 0 {
			t.Fatal("another owner changed persona privacy")
		}
	}
	for _, body := range []string{`{}`, `{"showContent":null}`} {
		if res := decodeContractEnvelope(t, serveJSON(router, path, body, token)); res.Code == 0 {
			t.Fatal("missing privacy value accepted")
		}
	}
	for _, show := range []bool{false, true} {
		body := `{"showContent":false}`
		if show {
			body = `{"showContent":true}`
		}
		rec := serveJSON(router, path, body, token)
		if res := decodeContractEnvelope(t, rec); res.Code != 0 || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("privacy update failed", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "anonymous-profile-privacy-success.json"))
		for _, viewer := range []string{"", token, contractSessionToken(t, other)} {
			props, raw := profile(viewer, 1)
			if props.ShowContent != show {
				t.Fatal("incorrect public privacy flag", props.ShowContent)
			}
			if !show && (len(props.Topics) != 0 || len(props.Replies) != 0 || props.TopicCount != 0 || props.ReplyCount != 0 || props.HasNext || strings.Contains(raw, topic.Title) || strings.Contains(raw, "private profile reply")) {
				t.Fatal("hidden profile leaked aggregate content", raw)
			}
			if show && (props.TopicCount != 1 || props.ReplyCount != 1 || len(props.Topics) != 1 || len(props.Replies) != 1) {
				t.Fatal("restore lost history", props)
			}
			if !show {
				later, _ := profile(viewer, 2)
				if later.HasNext || len(later.Replies) > 0 || len(later.Topics) > 0 {
					t.Fatal("hidden later page retained content", later)
				}
			}
		}
		var saved topics.Entity
		if err := conn.First(&saved, topic.Id).Error; err != nil || saved.Status != 1 {
			t.Fatal("privacy changed topic visibility", saved, err)
		}
	}
}
