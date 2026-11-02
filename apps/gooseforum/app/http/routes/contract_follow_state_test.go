package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// issue #1320：关注按钮需要区分「已互关」。后端在用户卡片与关注/粉丝列表上
// 暴露 viewer 视角的 isMutualFollowing，follow-user 响应返回动作执行后的状态，
// Web 与 App 只做渲染，互关判定保持在服务端。

type followUserResultPayload struct {
	IsFollowing       bool `json:"isFollowing"`
	IsMutualFollowing bool `json:"isMutualFollowing"`
}

func assertFollowUserResult(t *testing.T, recorder *httptest.ResponseRecorder, following, mutual bool) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("follow-user status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 {
		t.Fatalf("follow-user code = %d, want 0: %s", envelope.Code, recorder.Body.String())
	}
	var result followUserResultPayload
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode follow-user result %q: %v", envelope.Result, err)
	}
	if result.IsFollowing != following || result.IsMutualFollowing != mutual {
		t.Fatalf("follow-user result = %+v, want isFollowing=%v isMutualFollowing=%v", result, following, mutual)
	}
}

type followStateConnection struct {
	ID                uint64 `json:"id"`
	Username          string `json:"username"`
	IsFollowing       bool   `json:"isFollowing"`
	IsMutualFollowing bool   `json:"isMutualFollowing"`
}

type followStateProps struct {
	User struct {
		UserID            uint64 `json:"userId"`
		IsFollowing       bool   `json:"isFollowing"`
		IsMutualFollowing bool   `json:"isMutualFollowing"`
	} `json:"user"`
	Following  []followStateConnection `json:"following"`
	Followers  []followStateConnection `json:"followers"`
	Activities []struct {
		Action         int    `json:"action"`
		SubjectType    string `json:"subjectType"`
		SubjectID      uint64 `json:"subjectId"`
		ContentPreview string `json:"contentPreview"`
	} `json:"activities"`
}

func requestFollowStateProps(t *testing.T, router http.Handler, path, token string) followStateProps {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Goose-Page", "true")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d %.300s", path, rec.Code, rec.Body.String())
	}
	var payload struct {
		Props followStateProps `json:"props"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %s props: %v", path, err)
	}
	return payload.Props
}

func followStatePath(userID uint64, section string) string {
	path := "/u/" + strconv.FormatUint(userID, 10)
	if section != "" {
		path += "/" + section
	}
	return path
}

func TestUserProfileMutualFollowState(t *testing.T) {
	conn, router := setupProfilePageContractTest(t)
	base := contractTestID()
	viewer := createHTTPContractUser(t, conn, base)
	target := createHTTPContractUser(t, conn, base+1)
	peer := createHTTPContractUser(t, conn, base+2)
	// 三人都互相关注：列表里的非本人条目应当同时是「已关注」与「已互关」。
	for _, pair := range [][2]uint64{
		{viewer.Id, target.Id}, {target.Id, viewer.Id},
		{viewer.Id, peer.Id}, {peer.Id, viewer.Id},
		{target.Id, peer.Id}, {peer.Id, target.Id},
	} {
		if err := conn.Create(&userFollow.Entity{UserId: pair[0], FollowUserId: pair[1], Status: 1}).Error; err != nil {
			t.Fatalf("follow %d -> %d: %v", pair[0], pair[1], err)
		}
	}

	// 存量自关注行（#594 carve-out 允许用 action=2 自愈）：本人条目与本人卡片
	// 都不能因此得出「已关注/已互关」。
	if err := conn.Create(&userFollow.Entity{UserId: viewer.Id, FollowUserId: viewer.Id, Status: 1}).Error; err != nil {
		t.Fatalf("seed residual self follow: %v", err)
	}

	props := requestFollowStateProps(t, router, followStatePath(target.Id, ""), contractSessionToken(t, viewer))
	if !props.User.IsFollowing || !props.User.IsMutualFollowing {
		t.Fatalf("profile card follow state = %+v, want both directions true", props.User)
	}

	for _, section := range []string{"following", "followers"} {
		props := requestFollowStateProps(t, router, followStatePath(target.Id, section), contractSessionToken(t, viewer))
		list := props.Following
		if section == "followers" {
			list = props.Followers
		}
		others := 0
		for _, entry := range list {
			if entry.ID == viewer.Id {
				// 自己不能关注自己，本人条目只用于跳转。
				if entry.IsFollowing || entry.IsMutualFollowing {
					t.Fatalf("%s self entry follow state = %+v, want all false", section, entry)
				}
				continue
			}
			others++
			if entry.ID != peer.Id {
				t.Fatalf("%s entry = %+v, want the peer %d", section, entry, peer.Id)
			}
			if !entry.IsFollowing || !entry.IsMutualFollowing {
				t.Fatalf("%s entry mutual state = %+v, want both directions true", section, entry)
			}
		}
		if others != 1 {
			t.Fatalf("%s non-self entries = %d, want 1 (the peer)", section, others)
		}
	}

	// 匿名访问不带 viewer 视角：两个标记都必须为 false（issue #1320）。
	anonymous := requestFollowStateProps(t, router, followStatePath(target.Id, ""), "")
	if anonymous.User.IsFollowing || anonymous.User.IsMutualFollowing {
		t.Fatalf("anonymous viewer follow state = %+v, want all false", anonymous.User)
	}

	// 看自己的主页：关注与互关都不能为 true。
	own := requestFollowStateProps(t, router, followStatePath(viewer.Id, ""), contractSessionToken(t, viewer))
	if own.User.IsFollowing || own.User.IsMutualFollowing {
		t.Fatalf("own profile follow state = %+v, want all false", own.User)
	}

	// 单向关注（viewer 关注了 target，target 未回关）不是互关。
	if err := conn.Model(&userFollow.Entity{}).Where("user_id = ? AND follow_user_id = ?", target.Id, viewer.Id).Update("status", 0).Error; err != nil {
		t.Fatalf("clear reverse follow: %v", err)
	}
	oneWay := requestFollowStateProps(t, router, followStatePath(target.Id, ""), contractSessionToken(t, viewer))
	if !oneWay.User.IsFollowing || oneWay.User.IsMutualFollowing {
		t.Fatalf("one-way follow state = %+v, want isFollowing=true isMutualFollowing=false", oneWay.User)
	}

	// 反向残留：viewer→target 已置 0 而 target→viewer 仍为 1，两个标记都应为 false。
	if err := conn.Model(&userFollow.Entity{}).Where("user_id = ? AND follow_user_id = ?", target.Id, viewer.Id).Update("status", 1).Error; err != nil {
		t.Fatalf("restore reverse follow: %v", err)
	}
	if err := conn.Model(&userFollow.Entity{}).Where("user_id = ? AND follow_user_id = ?", viewer.Id, target.Id).Update("status", 0).Error; err != nil {
		t.Fatalf("clear forward follow: %v", err)
	}
	residual := requestFollowStateProps(t, router, followStatePath(target.Id, ""), contractSessionToken(t, viewer))
	if residual.User.IsFollowing || residual.User.IsMutualFollowing {
		t.Fatalf("residual unfollow state = %+v, want all false", residual.User)
	}
}

// 存量自关注行（status=1）不得让连接列表的本人条目变成「已关注/已互关」。
func TestUserProfileConnectionSelfEntryFollowState(t *testing.T) {
	conn, router := setupProfilePageContractTest(t)
	base := contractTestID()
	viewer := createHTTPContractUser(t, conn, base)
	target := createHTTPContractUser(t, conn, base+1)
	for _, pair := range [][2]uint64{
		{viewer.Id, viewer.Id},
		{viewer.Id, target.Id},
	} {
		if err := conn.Create(&userFollow.Entity{UserId: pair[0], FollowUserId: pair[1], Status: 1}).Error; err != nil {
			t.Fatalf("follow %d -> %d: %v", pair[0], pair[1], err)
		}
	}

	props := requestFollowStateProps(t, router, followStatePath(target.Id, "followers"), contractSessionToken(t, viewer))
	if len(props.Followers) != 1 || props.Followers[0].ID != viewer.Id {
		t.Fatalf("followers = %+v, want only the viewer's own entry", props.Followers)
	}
	if entry := props.Followers[0]; entry.IsFollowing || entry.IsMutualFollowing {
		t.Fatalf("self entry follow state = %+v, want all false", entry)
	}
}

// issue #1292：关注动态的正文取被关注者展示名。存量行里写入的占位文本必须被
// 读取时解析结果覆盖（历史数据不迁移），注册动态仍然保持空预览。
func TestFollowActivityPreviewShowsFollowedName(t *testing.T) {
	conn, router := setupProfilePageContractTest(t)
	base := contractTestID()
	owner := createHTTPContractUser(t, conn, base)
	target := createHTTPContractUser(t, conn, base+1)
	nicknamed := createHTTPContractUser(t, conn, base+2)
	closed := createHTTPContractUser(t, conn, base+3)
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", nicknamed.Id).Update("nickname", "昵称用户").Error; err != nil {
		t.Fatalf("set nickname: %v", err)
	}

	// 存量行：写入时预览是占位文本。
	if err := userActivities.Record(owner.Id, userActivities.ActionFollow, userActivities.SubjectUser, target.Id, "Followed a user"); err != nil {
		t.Fatalf("record legacy follow activity: %v", err)
	}
	if err := userActivities.Record(owner.Id, userActivities.ActionFollow, userActivities.SubjectUser, nicknamed.Id, ""); err != nil {
		t.Fatalf("record follow activity: %v", err)
	}
	if err := userActivities.Record(owner.Id, userActivities.ActionFollow, userActivities.SubjectUser, closed.Id, ""); err != nil {
		t.Fatalf("record closed-user follow activity: %v", err)
	}
	if err := userActivities.Record(owner.Id, userActivities.ActionSignUp, userActivities.SubjectUser, owner.Id, "joined"); err != nil {
		t.Fatalf("record signup activity: %v", err)
	}
	if err := conn.Delete(&users.EntityComplete{}, "id = ?", closed.Id).Error; err != nil {
		t.Fatalf("soft delete followed user: %v", err)
	}
	var closedRow users.EntityComplete
	if err := conn.Unscoped().First(&closedRow, "id = ?", closed.Id).Error; err != nil {
		t.Fatalf("reload soft-deleted followed user: %v", err)
	}
	if !closedRow.DeletedAt.Valid {
		t.Fatal("followed user is not soft deleted")
	}

	props := requestFollowStateProps(t, router, followStatePath(owner.Id, "activity"), contractSessionToken(t, owner))
	previews := make(map[uint64]string, len(props.Activities))
	var signupPreview *string
	for _, activity := range props.Activities {
		switch userActivities.ActionType(activity.Action) {
		case userActivities.ActionFollow:
			previews[activity.SubjectID] = activity.ContentPreview
		case userActivities.ActionSignUp:
			preview := activity.ContentPreview
			signupPreview = &preview
		}
	}

	if got := previews[target.Id]; got != target.Username {
		t.Fatalf("legacy follow preview = %q, want username %q (stored placeholder must be replaced on read)", got, target.Username)
	}
	if got := previews[nicknamed.Id]; got != "昵称用户" {
		t.Fatalf("follow preview = %q, want nickname 昵称用户", got)
	}
	if got := previews[closed.Id]; got != "" {
		t.Fatalf("closed-user follow preview = %q, want empty", got)
	}
	if signupPreview == nil || *signupPreview != "" {
		t.Fatalf("signup preview = %v, want an empty preview", signupPreview)
	}
}
