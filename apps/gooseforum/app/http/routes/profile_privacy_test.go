package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Issue #1124: profile privacy hides profile aggregates only. These tests
// exercise the real routes so JSON page payloads, SSR HTML, legacy URLs and
// the settings API share one server-side boundary.

func setupProfilePrivacyTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupProfilePageContractTest(t)
	if err := conn.AutoMigrate(&users.PrivacySettingsEntity{}, &users.BlockEntity{}); err != nil {
		t.Fatalf("migrate privacy tables: %v", err)
	}
	router.GET("/p/post/:id", middleware.JWTAuth, forum.TopicDetail)
	loginAPI := router.Group("/api").Use(middleware.CSRFProtection, middleware.JWTAuthCheck)
	loginAPI.GET("/user/privacy-settings", UpQueryReq(api.GetPrivacySettings))
	loginAPI.PATCH("/user/privacy-settings", middleware.RateLimit(middleware.RateLimitInteract), api.UpdatePrivacySettings)
	return conn, router
}

func servePrivacySettings(router http.Handler, method, body, token string) *httptest.ResponseRecorder {
	return serveAuthSecurityJSON(router, method, "/api/user/privacy-settings", body, token)
}

type profileVisibilityState struct {
	Visible  bool   `json:"visible"`
	Reason   string `json:"reason"`
	Audience string `json:"audience"`
}

type profilePrivacyProps struct {
	Topics     []json.RawMessage `json:"topics"`
	Activities []struct {
		Action int `json:"action"`
	} `json:"activities"`
	Likes      []json.RawMessage `json:"likes"`
	Following  []json.RawMessage `json:"following"`
	Followers  []json.RawMessage `json:"followers"`
	Pagination struct {
		HasNext bool   `json:"hasNext"`
		NextURL string `json:"nextUrl"`
	} `json:"pagination"`
	Visibility map[string]profileVisibilityState `json:"visibility"`
}

func requestProfileProps(t *testing.T, router http.Handler, path, token string) (profilePrivacyProps, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Goose-Page", "true")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var payload struct {
		Props profilePrivacyProps `json:"props"`
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d %.300s", path, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return payload.Props, rec
}

func setProfilePrivacy(t *testing.T, ownerID uint64, update users.PrivacySettingsUpdate) {
	t.Helper()
	if _, err := users.UpdatePrivacySettings(ownerID, update); err != nil {
		t.Fatalf("update privacy: %v", err)
	}
}

func follow(t *testing.T, conn *gorm.DB, from, to uint64) {
	t.Helper()
	if err := conn.Create(&userFollow.Entity{UserId: from, FollowUserId: to, Status: 1}).Error; err != nil {
		t.Fatalf("follow %d -> %d: %v", from, to, err)
	}
}

func TestPrivacySettingsHTTPContract(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)

	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		rec := servePrivacySettings(router, method, `{"profileTopics":"self"}`, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s without session = %d", method, rec.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "auth-required.json"))
	}

	rec := servePrivacySettings(router, http.MethodGet, "", token)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("default read: %d %q %s", rec.Code, rec.Header().Get("Cache-Control"), rec.Body)
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "privacy-settings-default.json"))

	rec = servePrivacySettings(router, http.MethodPatch, `{"profileTopics":"self"}`, token)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("first update: %d %s", rec.Code, rec.Body)
	}
	rec = servePrivacySettings(router, http.MethodPatch, `{"followersList":"mutual"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("partial update: %d %s", rec.Code, rec.Body)
	}
	updated := contractFixture(t, "privacy-settings-updated.json")
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), updated)

	// Invalid writes are rejected before storage: the stored row stays as is.
	invalid := map[string]string{
		`{}`:                                  "parse-failed.json",
		`[]`:                                  "parse-failed.json",
		`{"profileTopics":null}`:              "parse-failed.json",
		`{"profileTopics":1}`:                 "parse-failed.json",
		`{"showTopics":false}`:                "parse-failed.json",
		`{"profileTopics":"self","userId":1}`: "parse-failed.json",
		`{"profileTopics":"self"} {}`:         "parse-failed.json",
		`{"profileTopics":"nobody"}`:          "invalid-params.json",
		`{"profileLikes":"members"}`:          "invalid-params.json",
		`{"followingList":"members"}`:         "invalid-params.json",
		`{"profileTopics":"everyone","profileLikes":"mutual"}`: "invalid-params.json",
	}
	for body, fixture := range invalid {
		rec := servePrivacySettings(router, http.MethodPatch, body, token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid %s accepted: %d %s", body, rec.Code, rec.Body)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, fixture))
	}
	rec = servePrivacySettings(router, http.MethodPatch, strings.Repeat(" ", 5000)+`{"profileTopics":"self"}`, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized body accepted: %d", rec.Code)
	}
	rec = servePrivacySettings(router, http.MethodGet, "", token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), updated)

	// Another account only ever reads its own settings, whatever it asks for.
	rec = serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/user/privacy-settings?userId=%d", owner.Id), "", contractSessionToken(t, other))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "privacy-settings-default.json"))

	// Cookie sessions keep the CSRF gate on the write.
	cookieRec := serveAuthSecurityJSONWithCookie(router, http.MethodPatch, "/api/user/privacy-settings", `{"profileTopics":"everyone"}`, token)
	if cookieRec.Code == http.StatusOK {
		t.Fatalf("cross-site cookie write accepted: %s", cookieRec.Body)
	}
}

// profileAudienceFixture builds an owner with one item in every section and a
// viewer for each relationship the evaluator distinguishes.
type profileAudienceFixture struct {
	owner, stranger, follower, mutual, blocked *users.EntityComplete
	topicTitle                                 string
	topicID                                    uint64
}

func newProfileAudienceFixture(t *testing.T, conn *gorm.DB) profileAudienceFixture {
	t.Helper()
	base := contractTestID()
	f := profileAudienceFixture{
		owner:    createHTTPContractUser(t, conn, base),
		stranger: createHTTPContractUser(t, conn, base+1),
		follower: createHTTPContractUser(t, conn, base+2),
		mutual:   createHTTPContractUser(t, conn, base+3),
		blocked:  createHTTPContractUser(t, conn, base+4),
		topicID:  base + 10,
	}
	createContractPublishedTopic(t, conn, f.topicID, base+11, f.owner.Id)
	f.topicTitle = "Contract interaction topic"
	likedTopic := base + 20
	createContractPublishedTopic(t, conn, likedTopic, base+21, f.stranger.Id)
	topicUserAction.SetLiked(f.owner.Id, likedTopic, true)
	follow(t, conn, f.follower.Id, f.owner.Id)
	follow(t, conn, f.mutual.Id, f.owner.Id)
	follow(t, conn, f.owner.Id, f.mutual.Id)
	// A follower who is blocked must not keep follower access.
	follow(t, conn, f.blocked.Id, f.owner.Id)
	if err := users.SetBlockedUser(f.owner.Id, f.blocked.Id, true); err != nil {
		t.Fatalf("block: %v", err)
	}
	now := time.Now()
	for i, row := range []userActivities.Entity{
		{UserId: f.owner.Id, Action: int(userActivities.ActionPost), SubjectType: userActivities.SubjectTopic, SubjectId: f.topicID},
		{UserId: f.owner.Id, Action: int(userActivities.ActionLike), SubjectType: userActivities.SubjectTopic, SubjectId: likedTopic},
		{UserId: f.owner.Id, Action: int(userActivities.ActionFollow), SubjectType: userActivities.SubjectUser, SubjectId: f.mutual.Id},
		{UserId: f.owner.Id, Action: int(userActivities.ActionComment), SubjectType: userActivities.SubjectPost, SubjectId: base + 11},
	} {
		row.Id = base + 30 + uint64(i)
		row.CreatedAt = now.Add(time.Duration(i) * time.Second)
		if err := conn.Create(&row).Error; err != nil {
			t.Fatalf("activity: %v", err)
		}
	}
	return f
}

func TestProfilePrivacyAudienceMatrix(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	f := newProfileAudienceFixture(t, conn)
	tokens := map[string]string{
		"owner":    contractSessionToken(t, f.owner),
		"guest":    "",
		"stranger": contractSessionToken(t, f.stranger),
		"follower": contractSessionToken(t, f.follower),
		"mutual":   contractSessionToken(t, f.mutual),
		"blocked":  contractSessionToken(t, f.blocked),
	}
	uid := f.owner.Id
	type route struct {
		path    string
		section string
		items   func(profilePrivacyProps) int
	}
	routes := []route{
		{fmt.Sprintf("/u/%d/activity/topics", uid), "profileTopics", func(p profilePrivacyProps) int { return len(p.Topics) }},
		{fmt.Sprintf("/u/%d", uid), "profileTopics", func(p profilePrivacyProps) int { return len(p.Topics) }},
		{fmt.Sprintf("/u/%d/activity/likes", uid), "profileLikes", func(p profilePrivacyProps) int { return len(p.Likes) }},
		{fmt.Sprintf("/u/%d/activity", uid), "profileActivity", func(p profilePrivacyProps) int { return len(p.Activities) }},
		{fmt.Sprintf("/u/%d/following", uid), "followingList", func(p profilePrivacyProps) int { return len(p.Following) }},
		{fmt.Sprintf("/u/%d/activity/following", uid), "followingList", func(p profilePrivacyProps) int { return len(p.Following) }},
		{fmt.Sprintf("/u/%d/followers", uid), "followersList", func(p profilePrivacyProps) int { return len(p.Followers) }},
		{fmt.Sprintf("/u/%d/activity/followers", uid), "followersList", func(p profilePrivacyProps) int { return len(p.Followers) }},
	}
	allowed := map[string]map[string]bool{
		users.AudienceEveryone:  {"owner": true, "guest": true, "stranger": true, "follower": true, "mutual": true, "blocked": true},
		users.AudienceMembers:   {"owner": true, "stranger": true, "follower": true, "mutual": true},
		users.AudienceFollowers: {"owner": true, "follower": true, "mutual": true},
		users.AudienceMutual:    {"owner": true, "mutual": true},
		users.AudienceSelf:      {"owner": true},
	}
	reasons := map[string]map[string]string{
		users.AudienceMembers:   {"guest": "signin", "blocked": "private"},
		users.AudienceFollowers: {"guest": "followers", "stranger": "followers", "blocked": "private"},
		users.AudienceMutual:    {"guest": "mutual", "stranger": "mutual", "follower": "mutual", "blocked": "private"},
		users.AudienceSelf:      {"guest": "private", "stranger": "private", "follower": "private", "mutual": "private", "blocked": "private"},
	}
	for _, audience := range []string{users.AudienceEveryone, users.AudienceMembers, users.AudienceFollowers, users.AudienceMutual, users.AudienceSelf} {
		for _, r := range routes {
			if !validAudienceFor(r.section, audience) {
				continue
			}
			resetProfilePrivacy(t, conn, uid)
			setProfilePrivacy(t, uid, updateFor(r.section, audience))
			for viewer, token := range tokens {
				props, rec := requestProfileProps(t, router, r.path, token)
				if got := rec.Header().Get("Cache-Control"); got != "private, no-store" {
					t.Fatalf("%s Cache-Control = %q", r.path, got)
				}
				state := props.Visibility[r.section]
				want := allowed[audience][viewer]
				if state.Visible != want || (r.items(props) > 0) != want {
					t.Fatalf("%s audience=%s viewer=%s: visible=%v items=%d, want %v", r.path, audience, viewer, state.Visible, r.items(props), want)
				}
				if !want {
					if state.Reason != reasons[audience][viewer] {
						t.Fatalf("%s audience=%s viewer=%s reason=%q, want %q", r.path, audience, viewer, state.Reason, reasons[audience][viewer])
					}
					if props.Pagination.HasNext || props.Pagination.NextURL != "" {
						t.Fatalf("%s hidden section leaked a cursor: %+v", r.path, props.Pagination)
					}
				}
				// Raw settings are never sent to anyone but the owner.
				if (state.Audience != "") != (viewer == "owner") {
					t.Fatalf("%s viewer=%s audience field = %q", r.path, viewer, state.Audience)
				}
				if viewer == "owner" && state.Audience != audience {
					t.Fatalf("owner audience = %q, want %q", state.Audience, audience)
				}
			}
		}
	}
}

func validAudienceFor(section, audience string) bool {
	for _, allowed := range users.PrivacyAudiences[section] {
		if allowed == audience {
			return true
		}
	}
	return false
}

func updateFor(section, audience string) users.PrivacySettingsUpdate {
	switch section {
	case "profileTopics":
		return users.PrivacySettingsUpdate{ProfileTopics: &audience}
	case "profileActivity":
		return users.PrivacySettingsUpdate{ProfileActivity: &audience}
	case "profileLikes":
		return users.PrivacySettingsUpdate{ProfileLikes: &audience}
	case "followingList":
		return users.PrivacySettingsUpdate{FollowingList: &audience}
	default:
		return users.PrivacySettingsUpdate{FollowersList: &audience}
	}
}

func resetProfilePrivacy(t *testing.T, conn *gorm.DB, userID uint64) {
	t.Helper()
	if err := conn.Where("user_id = ?", userID).Delete(&users.PrivacySettingsEntity{}).Error; err != nil {
		t.Fatalf("reset privacy: %v", err)
	}
}

// Timeline rows obey their own sections: hidden likes, follows or topics must
// not appear in the timeline, take a page slot or produce a cursor.
func TestProfilePrivacyTimelineDoesNotLeakHiddenActions(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	f := newProfileAudienceFixture(t, conn)
	base := contractTestID()
	for i := range 25 {
		row := userActivities.Entity{Id: base + uint64(i), UserId: f.owner.Id, Action: int(userActivities.ActionLike), SubjectType: userActivities.SubjectTopic, SubjectId: f.topicID, CreatedAt: time.Now()}
		if err := conn.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	timeline := fmt.Sprintf("/u/%d/activity", f.owner.Id)
	summary := fmt.Sprintf("/u/%d", f.owner.Id)
	self := users.AudienceSelf
	cases := []struct {
		name   string
		update users.PrivacySettingsUpdate
		hidden []int
	}{
		{"likes", users.PrivacySettingsUpdate{ProfileLikes: &self}, []int{int(userActivities.ActionLike)}},
		{"following", users.PrivacySettingsUpdate{FollowingList: &self}, []int{int(userActivities.ActionFollow)}},
		{"topics", users.PrivacySettingsUpdate{ProfileTopics: &self}, []int{int(userActivities.ActionPost)}},
		{"all", users.PrivacySettingsUpdate{ProfileLikes: &self, FollowingList: &self, ProfileTopics: &self}, []int{2, 3, 4}},
	}
	for _, tc := range cases {
		resetProfilePrivacy(t, conn, f.owner.Id)
		setProfilePrivacy(t, f.owner.Id, tc.update)
		for _, path := range []string{timeline, summary} {
			props, _ := requestProfileProps(t, router, path, contractSessionToken(t, f.stranger))
			if len(props.Activities) == 0 {
				t.Fatalf("%s %s: visible activity rows disappeared", tc.name, path)
			}
			for _, row := range props.Activities {
				for _, hidden := range tc.hidden {
					if row.Action == hidden {
						t.Fatalf("%s %s leaked hidden action %d", tc.name, path, hidden)
					}
				}
			}
			if path == timeline && tc.name == "likes" && props.Pagination.HasNext {
				t.Fatalf("hidden likes produced a timeline cursor: %+v", props.Pagination)
			}
		}
	}
	// The owner still pages through every row.
	resetProfilePrivacy(t, conn, f.owner.Id)
	setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{ProfileLikes: &self})
	props, _ := requestProfileProps(t, router, timeline, contractSessionToken(t, f.owner))
	if len(props.Activities) != 20 || !props.Pagination.HasNext {
		t.Fatalf("owner timeline = %d rows hasNext=%v", len(props.Activities), props.Pagination.HasNext)
	}
	// A hidden activity section hides the whole timeline, including summaries.
	setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{ProfileActivity: &self})
	for _, path := range []string{timeline, summary} {
		props, _ := requestProfileProps(t, router, path, "")
		if len(props.Activities) != 0 || props.Pagination.HasNext {
			t.Fatalf("%s hidden timeline leaked %d rows", path, len(props.Activities))
		}
	}
}

func TestProfilePrivacyKeepsPublicContentAndBookmarkBoundary(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	f := newProfileAudienceFixture(t, conn)
	ssr := func() string {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/u/%d/activity/topics", f.owner.Id), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("SSR profile = %d", rec.Code)
		}
		return rec.Body.String()
	}
	// Positive control: public crawler HTML does render the topic title.
	if !strings.Contains(ssr(), f.topicTitle) {
		t.Fatal("public SSR profile lacks the topic title; the hidden check below would be vacuous")
	}
	self := users.AudienceSelf
	setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{ProfileTopics: &self, ProfileActivity: &self, ProfileLikes: &self, FollowingList: &self, FollowersList: &self})

	// Model A: the original topic keeps ordinary forum visibility.
	page := getTopicPage(router, f.topicID, "")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), f.topicTitle) {
		t.Fatalf("public topic hidden by profile privacy: %d %.200s", page.Code, page.Body.String())
	}

	// Crawler HTML follows the same boundary as the JSON payload.
	if strings.Contains(ssr(), f.topicTitle) {
		t.Fatal("SSR profile leaked a hidden topic")
	}

	// Bookmarks stay owner-only regardless of the new settings.
	for _, token := range []string{"", contractSessionToken(t, f.stranger), contractSessionToken(t, f.mutual)} {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/u/%d/bookmarks", f.owner.Id), nil)
		req.Header.Set("X-Goose-Page", "true")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("bookmarks visible to non-owner: %d", rec.Code)
		}
	}
	if _, rec := requestProfileProps(t, router, fmt.Sprintf("/u/%d/bookmarks", f.owner.Id), contractSessionToken(t, f.owner)); rec.Code != http.StatusOK {
		t.Fatalf("owner bookmarks = %d", rec.Code)
	}
}

// Settings take effect on the next request: there is no profile cache that
// can replay an older audience, and restoring "everyone" restores history.
func TestProfilePrivacyChangesApplyImmediately(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	f := newProfileAudienceFixture(t, conn)
	token := contractSessionToken(t, f.owner)
	path := fmt.Sprintf("/u/%d/activity/topics", f.owner.Id)
	if props, _ := requestProfileProps(t, router, path, ""); len(props.Topics) == 0 {
		t.Fatal("public topics missing before the change")
	}
	if rec := servePrivacySettings(router, http.MethodPatch, `{"profileTopics":"self"}`, token); rec.Code != http.StatusOK {
		t.Fatalf("restrict: %d %s", rec.Code, rec.Body)
	}
	if props, _ := requestProfileProps(t, router, path, ""); len(props.Topics) != 0 || props.Visibility["profileTopics"].Visible {
		t.Fatal("restricted topics still served")
	}
	if rec := servePrivacySettings(router, http.MethodPatch, `{"profileTopics":"everyone"}`, token); rec.Code != http.StatusOK {
		t.Fatalf("restore: %d %s", rec.Code, rec.Body)
	}
	if props, _ := requestProfileProps(t, router, path, ""); len(props.Topics) == 0 {
		t.Fatal("restored topics not served")
	}
	// Unfollowing revokes follower access on the next request.
	setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{FollowersList: ptr(users.AudienceFollowers)})
	followerToken := contractSessionToken(t, f.follower)
	followers := fmt.Sprintf("/u/%d/followers", f.owner.Id)
	if props, _ := requestProfileProps(t, router, followers, followerToken); !props.Visibility["followersList"].Visible {
		t.Fatal("follower denied")
	}
	if err := conn.Model(&userFollow.Entity{}).Where("user_id = ? AND follow_user_id = ?", f.follower.Id, f.owner.Id).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	if props, _ := requestProfileProps(t, router, followers, followerToken); props.Visibility["followersList"].Visible || len(props.Followers) != 0 {
		t.Fatal("unfollowed viewer kept follower access")
	}
}

// AC-12: staff permissions grant no extra reach through public profile
// routes; moderation keeps its own tools.
func TestProfilePrivacyGivesStaffNoPublicBypass(t *testing.T) {
	conn, router := setupProfilePrivacyTest(t)
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate role permissions: %v", err)
	}
	f := newProfileAudienceFixture(t, conn)
	setProfilePrivacy(t, f.owner.Id, users.PrivacySettingsUpdate{
		ProfileTopics: ptr(users.AudienceSelf), ProfileLikes: ptr(users.AudienceSelf), FollowersList: ptr(users.AudienceSelf),
	})
	for _, perm := range []permission.Enum{permission.Admin, permission.UserManager, permission.TopicsManager} {
		staff := createHTTPContractUser(t, conn, contractTestID())
		grantContractPermission(t, conn, staff.Id, perm)
		token := contractSessionToken(t, staff)
		for _, section := range []struct{ path, key string }{
			{fmt.Sprintf("/u/%d/activity/topics", f.owner.Id), "profileTopics"},
			{fmt.Sprintf("/u/%d/activity/likes", f.owner.Id), "profileLikes"},
			{fmt.Sprintf("/u/%d/followers", f.owner.Id), "followersList"},
		} {
			props, _ := requestProfileProps(t, router, section.path, token)
			got := props.Visibility[section.key]
			if got.Visible || got.Reason != "private" || got.Audience != "" {
				t.Fatalf("permission %d on %s: %+v", perm, section.path, got)
			}
			if len(props.Topics)+len(props.Likes)+len(props.Followers) != 0 {
				t.Fatalf("permission %d on %s leaked rows", perm, section.path)
			}
		}
	}
}

func ptr[T any](value T) *T { return &value }
