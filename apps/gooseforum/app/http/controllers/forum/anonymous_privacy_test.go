package forum

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
)

func TestPersonaPublicProjectionAndLifecycle(t *testing.T) {
	conn := setupRevisionTestDB(t)
	if err := conn.AutoMigrate(&identity.Persona{}, &userFollow.Entity{}); err != nil {
		t.Fatal(err)
	}
	owner := users.EntityComplete{Id: 991731, Username: "owner-sensitive-name", AvatarUrl: "/owner-sensitive-avatar.png"}
	persona := identity.Persona{UID: strings.Repeat("a", 32), Name: "中华人民共和国道路交通安全法实施条例", AvatarSeed: strings.Repeat("b", 32)}
	topic := topics.Entity{Id: 991740, UserId: owner.Id, PersonaUID: persona.UID, Status: 1, FirstPostId: 991741, Title: "Public topic"}
	first := posts.Entity{Id: 991741, TopicId: topic.Id, PostNo: 1, UserId: owner.Id, PersonaUID: persona.UID, IsAnonymous: true, Content: "Public body", LastEditorId: owner.Id}
	reply := posts.Entity{Id: 991742, TopicId: topic.Id, PostNo: 2, UserId: owner.Id, PersonaUID: persona.UID, IsAnonymous: true, Content: "Public reply", ReplyToPostId: first.Id, LastEditorId: owner.Id}
	revision := postRevisions.Entity{PostId: first.Id, Version: 1, EditorId: owner.Id, Content: first.Content, RenderedHTML: "<p>Public body</p>"}
	for _, row := range []any{&owner, &persona, &topic, &first, &reply, &revision} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(row) })
	}
	check := func(value any) {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{owner.Username, owner.AvatarUrl, fmt.Sprintf(`"id":%d`, owner.Id), `"ownerId"`, `"avatarSeed"`} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("private field %s escaped: %s", forbidden, raw)
			}
		}
	}
	for _, viewer := range []uint64{0, owner.Id, owner.Id + 1} {
		payload, targets := buildPostPayloads([]*posts.Entity{&first, &reply}, map[uint64]*users.EntityComplete{owner.Id: &owner}, viewer, false, &first)
		check(payload)
		check(targets)
		if payload[0].Author.PublicUID != persona.UID || payload[0].Author.ID != 0 || payload[1].ReplyToUserID != 0 || payload[1].ReplyToUsername != persona.Name || payload[0].LastEditor.PublicUID != persona.UID {
			t.Fatal(payload)
		}
		check(PostRevisions(component.BetterRequest[PostRevisionsReq]{UserId: viewer, Params: PostRevisionsReq{PostID: first.Id}}).Data)
	}
	participants, err := posts.PersonaParticipants(conn, []uint64{topic.Id})
	if err != nil || len(participants) != 1 {
		t.Fatal(participants, err)
	}
	rows, nt, nr, err := posts.PublicPersonaPosts(conn, persona.UID, 1)
	if err != nil || len(rows) != 1 || nt != 1 || nr != 1 {
		t.Fatal(rows, nt, nr, err)
	}
	for _, column := range []string{"visibility_status", "retention_status"} {
		value := posts.VisibilityModeratorRemoved
		if column == "retention_status" {
			value = posts.RetentionPurged
		}
		conn.Model(&first).Update(column, value)
		rows, nt, nr, err = posts.PublicPersonaPosts(conn, persona.UID, 1)
		if err != nil || len(rows) != 0 || nt != 0 || nr != 0 {
			t.Fatal("hidden parent exposed", rows, nt, nr, err)
		}
		reset := posts.VisibilityActive
		if column == "retention_status" {
			reset = posts.RetentionNormal
		}
		conn.Model(&first).Update(column, reset)
	}
	// A missing persona cannot fall back to its private author.
	conn.Delete(&persona)
	payload, _ := buildPostPayloads([]*posts.Entity{&first}, map[uint64]*users.EntityComplete{owner.Id: &owner}, 0, false, &first)
	check(payload)
	if payload[0].Author.PublicUID != persona.UID || payload[0].Author.ID != 0 {
		t.Fatal(payload)
	}
}

func TestFollowingFeedExcludesPrivatePersonaOwnership(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &userFollow.Entity{}); err != nil {
		t.Fatal(err)
	}
	edge := userFollow.Entity{UserId: 991801, FollowUserId: 991802, Status: 1}
	topic := topics.Entity{Id: 991803, UserId: edge.FollowUserId, PersonaUID: strings.Repeat("c", 32), Status: 1, FirstPostId: 991804, CreatedAt: time.Now()}
	post := posts.Entity{Id: 991804, TopicId: topic.Id, UserId: edge.FollowUserId, PostNo: 1, PersonaUID: topic.PersonaUID, IsAnonymous: true}
	for _, row := range []any{&edge, &topic, &post} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(row) })
	}
	rows, _, err := topics.FollowingPage(context.Background(), edge.UserId, nil, 20)
	if err != nil || len(rows) != 0 {
		t.Fatal("follow exposed private ownership", rows, err)
	}
}

func TestTopicListProjectsPersonaWithoutTrustingTransformedAuthor(t *testing.T) {
	conn := setupRevisionTestDB(t)
	if err := conn.AutoMigrate(&identity.Persona{}); err != nil {
		t.Fatal(err)
	}
	persona := identity.Persona{UID: strings.Repeat("d", 32), Name: "完整花名", AvatarSeed: strings.Repeat("e", 32)}
	if err := conn.Create(&persona).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&persona) })
	for _, uid := range []string{persona.UID, strings.Repeat("f", 32)} {
		topic := &vo.TopicsSimpleVo{Id: 991850, PersonaUID: uid, AuthorId: 991851, Username: "private-owner", Nickname: "private-nickname", AvatarUrl: "/private-avatar.png",
			Posters: []vo.PosterVo{{PersonaUID: uid, Id: 991851, Username: "private-owner", AvatarUrl: "/private-avatar.png"}, {Id: 991852, Username: "public-replier"}}}
		payload := buildTopicPayloads([]*vo.TopicsSimpleVo{topic})
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"private-owner", "private-nickname", "/private-avatar.png", `"id":991851`} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("private author escaped list boundary: %s", raw)
			}
		}
		if len(payload) != 1 || payload[0].Author.PublicUID != uid || payload[0].Author.ProfileURL != "/a/"+uid || payload[0].Author.ID != 0 {
			t.Fatalf("missing independent persona projection: %s", raw)
		}
		if uid == persona.UID && payload[0].Author.Username != persona.Name {
			t.Fatalf("list did not resolve current public name: %s", raw)
		}
		if len(payload[0].Participants) != 1 || payload[0].Participants[0].ID != 991852 {
			t.Fatalf("public member participant lost: %s", raw)
		}
	}
}

func TestAdminWildcardDoesNotAdvertiseAnonymousReveal(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	const id = 9894401
	if err := conn.Create(&users.EntityComplete{Id: id, Username: "wildcard-admin", RoleId: id}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&rolePermissionRs.Entity{RoleId: id, PermissionId: permission.Admin.Id(), Effective: 1}).Error; err != nil {
		t.Fatal(err)
	}
	permissions := buildAdminPermissions(id)
	if slices.Contains(permissions, permission.RevealAnonymousIdentity.Id()) {
		t.Fatal("Admin wildcard advertised restricted reveal", permissions)
	}
}

func TestPersonaStructuredMetadataUsesIndependentProfile(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "http://forum.local/p/post/1", nil)
	uid := strings.Repeat("a", 32)
	topic := TopicDetailPayload{URL: "/p/post/1", Author: TopicAuthorPayload{Kind: "persona", PublicUID: uid, ProfileURL: "/a/" + uid, Username: "匿名花名"}}
	meta := buildTopicMeta(c, topic)
	structured := meta.JSONLD.(vo.ArticleJSONLD)
	if !strings.HasSuffix(structured.Author.URL, "/a/"+uid) {
		t.Fatal("structured data points to a main profile", structured.Author)
	}
}

func TestAnonymousDeletedEvidenceRedactsSelfDeletionActor(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	owner, moderator, _, topicID, _ := seedReportSnapshotTopic(t, conn, 9895000)
	topic := topics.UnscopedGet(topicID)
	uid := strings.Repeat("d", 32)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Updates(map[string]any{"persona_uid": uid, "deleted_by": owner, "visibility_status": topics.VisibilityUserDeleted}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&posts.Entity{}).Where("id = ?", topic.FirstPostId).Updates(map[string]any{"persona_uid": uid, "is_anonymous": true, "deleted_by": owner, "visibility_status": posts.VisibilityUserDeleted}).Error; err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		kind string
		id   uint64
	}{{reports.TargetTopic, topicID}, {reports.TargetPost, topic.FirstPostId}} {
		res := ViewDeletedContent(component.BetterRequest[ViewDeletedContentReq]{UserId: moderator, Params: ViewDeletedContentReq{ContentType: target.kind, ContentID: target.id, Reason: "核对匿名取证"}})
		if res.Data.Code != component.SUCCESS {
			t.Fatal(res)
		}
		view := res.Data.Result.(ModerationDeletedContentView)
		if view.AuthorID != 0 || view.DeletedBy != 0 || view.AuthorName != "" || view.DeletedByWho != "" {
			t.Fatalf("%s self-delete exposed owner: %+v", target.kind, view)
		}
	}
}
