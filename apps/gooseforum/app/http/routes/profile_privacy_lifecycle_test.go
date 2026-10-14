package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
)

// AC-16 (issue #1124): a stricter audience keeps holding while content and
// accounts change. Deleted content never comes back through the profile, and
// widening the audience again only restores what is still public.
func TestProfilePrivacyHoldsThroughContentDeletion(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	f := newProfileAudienceFixture(t, conn)
	path := fmt.Sprintf("/u/%d/activity/topics", f.owner.Id)
	followerToken := contractSessionToken(t, f.follower)
	setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{ProfileTopics: ptr(users.AudienceFollowers)})

	// Positive control: the follower sees the topic before it is removed.
	if props, _ := requestProfileProps(t, router, path, followerToken); len(props.Topics) != 1 {
		t.Fatalf("follower topics = %d, want 1", len(props.Topics))
	}
	if props, _ := requestProfileProps(t, router, path, contractSessionToken(t, f.stranger)); len(props.Topics) != 0 || props.Visibility["profileTopics"].Visible {
		t.Fatal("stranger saw a followers-only topic list")
	}

	for _, status := range []string{topics.VisibilityUserDeleted, topics.VisibilityModeratorRemoved} {
		if err := conn.Model(&topics.Entity{}).Where("id = ?", f.topicID).Update("visibility_status", status).Error; err != nil {
			t.Fatal(err)
		}
		for _, viewer := range []string{followerToken, contractSessionToken(t, f.owner)} {
			props, rec := requestProfileProps(t, router, path, viewer)
			if len(props.Topics) != 0 || strings.Contains(rec.Body.String(), f.topicTitle) {
				t.Fatalf("%s topic still listed on the profile", status)
			}
		}
		// Widening the audience does not resurrect removed content.
		setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{ProfileTopics: ptr(users.AudienceEveryone)})
		if props, rec := requestProfileProps(t, router, path, ""); len(props.Topics) != 0 || strings.Contains(rec.Body.String(), f.topicTitle) {
			t.Fatalf("%s topic came back after widening", status)
		}
		setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{ProfileTopics: ptr(users.AudienceFollowers)})
	}
}

// Closing the account removes the profile for everyone and drops its stored
// audiences together with the other private relationship state.
func TestProfilePrivacyClosedAccountServesNothing(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	if err := conn.AutoMigrate(&users.PrivateNoteEntity{}); err != nil {
		t.Fatal(err)
	}
	f := newProfileAudienceFixture(t, conn)
	self := users.AudienceSelf
	setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{ProfileTopics: &self, ProfileActivity: &self, ProfileLikes: &self, FollowingList: &self, FollowersList: &self})
	if err := users.CloseAccount(f.owner.Id); err != nil {
		t.Fatalf("close account: %v", err)
	}
	var rows int64
	if err := conn.Model(&users.PrivacySettingsEntity{}).Where("user_id = ?", f.owner.Id).Count(&rows).Error; err != nil || rows != 0 {
		t.Fatalf("privacy row survived closure: rows=%d err=%v", rows, err)
	}
	for _, section := range []string{"", "/activity", "/activity/topics", "/activity/likes", "/following", "/followers"} {
		for _, token := range []string{"", contractSessionToken(t, f.follower), contractSessionToken(t, f.mutual)} {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/u/%d%s", f.owner.Id, section), nil)
			req.Header.Set("X-Goose-Page", "true")
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), f.topicTitle) {
				t.Fatalf("closed profile %s = %d %.200s", section, rec.Code, rec.Body.String())
			}
		}
	}
	// Model A: retained public topics keep their own visibility.
	if page := getTopicPage(router, f.topicID, ""); page.Code != http.StatusOK {
		t.Fatalf("retained topic = %d", page.Code)
	}
}

// Real-account audiences must never reach an anonymous profile: a matching
// hidden state there would link the persona back to its owner.
func TestProfilePrivacyNeverReachesAnonymousProfiles(t *testing.T) {
	conn, _ := setupAccountContractTest(t)
	if err := conn.AutoMigrate(&identity.Persona{}, &identity.Binding{}, &identity.Quota{}, &identity.Batch{}, &topics.Entity{}, &posts.Entity{}, &users.PrivacySettingsEntity{}); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterByGin(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	uid := fmt.Sprintf("%032x", contractTestID())
	firstID := contractTestID()
	topic := topics.Entity{Id: contractTestID(), FirstPostId: firstID, UserId: owner.Id, PersonaUID: uid, Title: "persona topic under strict owner", Status: 1}
	for _, row := range []any{
		&identity.Persona{UID: uid, Name: "persona", AvatarSeed: "persona-seed"},
		&identity.Binding{OwnerID: owner.Id, PersonaUID: uid}, &topic,
		&posts.Entity{Id: firstID, TopicId: topic.Id, PostNo: 1, UserId: owner.Id, PersonaUID: uid, Content: "persona first post"},
	} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatalf("create %#v: %v", row, err)
		}
		t.Cleanup(func() { conn.Delete(row) })
	}
	anonymous := func(token string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/a/"+uid, nil)
		req.Header.Set("X-Goose-Page", "true")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("anonymous profile = %d", rec.Code)
		}
		var payload struct {
			Props map[string]json.RawMessage `json:"props"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if _, ok := payload.Props["visibility"]; ok {
			t.Fatal("anonymous profile exposes real-account visibility")
		}
		return string(payload.Props["topics"])
	}
	before := anonymous("")
	self := users.AudienceSelf
	setProfilePrivacy(t, owner.Id, users.PrivacySettingsUpdate{ProfileTopics: &self, ProfileActivity: &self, ProfileLikes: &self, FollowingList: &self, FollowersList: &self})
	after := anonymous("")
	if !strings.Contains(after, topic.Title) || before != after {
		t.Fatalf("real-account privacy changed the anonymous profile: before=%s after=%s", before, after)
	}
	if !strings.Contains(anonymous(contractSessionToken(t, owner)), topic.Title) {
		t.Fatal("owner lost the persona topic")
	}
}
