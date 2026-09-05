package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// TestEditUsernameRejectsReservedWithVariants 用户名改名对保留名单的归一化全等：
// 大小写/leet 变体命中返回 auth.username.reserved。用例保留词 ≥6 位，
// 保证先通过 username 格式校验、名单检查可达（<6 位词会被格式校验先拦截）。
func TestEditUsernameRejectsReservedWithVariants(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	persistSecurityPolicyConfig(t, conn, func() pageConfig.SecurityAndRegistration {
		cfg := emptySecurityConfig()
		cfg.ReservedUsernames = []string{"administrator", "moderator"}
		return cfg
	}())

	for _, tt := range []struct {
		name     string
		username string
		wantCode string
	}{
		{name: "reserved exact", username: "administrator", wantCode: string(component.MessageAuthUsernameReserved)},
		{name: "reserved case", username: "Administrator", wantCode: string(component.MessageAuthUsernameReserved)},
		{name: "reserved leet", username: "adm1nistrat0r", wantCode: string(component.MessageAuthUsernameReserved)},
		{name: "reserved second", username: "MODERATOR", wantCode: string(component.MessageAuthUsernameReserved)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"username":"` + tt.username + `"}`
			rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/set-user-name", body, token)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 envelope: %s", rec.Code, rec.Body.String())
			}
			var resp map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got := resp["messageCode"]; got != tt.wantCode {
				t.Fatalf("messageCode = %v, want %s (body %s)", got, tt.wantCode, rec.Body.String())
			}
		})
	}
}

// TestEditUserInfoRejectsReservedNicknameAndSensitiveProfile 昵称名单检查 +
// 资料字段敏感词拦截。
func TestEditUserInfoRejectsReservedNicknameAndSensitiveProfile(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	persistSecurityPolicyConfig(t, conn, func() pageConfig.SecurityAndRegistration {
		cfg := emptySecurityConfig()
		cfg.ReservedUsernames = []string{"官方", "admin"}
		cfg.SensitiveWords = []string{"代开发票", "赌博"}
		return cfg
	}())

	t.Run("reserved nickname blocked", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/set-user-info",
			`{"nickname":"官方","bio":"x"}`, token)
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if got := resp["messageCode"]; got != string(component.MessageAuthNicknameReserved) {
			t.Fatalf("nickname reserved messageCode = %v, want %s (body %s)",
				got, component.MessageAuthNicknameReserved, rec.Body.String())
		}
	})

	t.Run("reserved nickname leet blocked", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/set-user-info",
			`{"nickname":"adm1n","bio":"x"}`, token)
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if got := resp["messageCode"]; got != string(component.MessageAuthNicknameReserved) {
			t.Fatalf("nickname leet messageCode = %v, want %s", got, component.MessageAuthNicknameReserved)
		}
	})

	t.Run("sensitive bio blocked", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/set-user-info",
			`{"nickname":"正常鹅","bio":"需要代开发票请联系"}`, token)
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if got := resp["messageCode"]; got != string(component.MessageContentSensitiveBlocked) {
			t.Fatalf("bio sensitive messageCode = %v, want %s (body %s)",
				got, component.MessageContentSensitiveBlocked, rec.Body.String())
		}
	})

	t.Run("sensitive signature blocked", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/set-user-info",
			`{"nickname":"正常鹅","signature":"一起来赌博"}`, token)
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if got := resp["messageCode"]; got != string(component.MessageContentSensitiveBlocked) {
			t.Fatalf("signature sensitive messageCode = %v, want %s (body %s)",
				got, component.MessageContentSensitiveBlocked, rec.Body.String())
		}
	})

	t.Run("normal profile passes", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/set-user-info",
			`{"nickname":"契约昵称","bio":"正常简介","signature":"正常签名"}`, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("normal profile status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if got := resp["messageCode"]; got != string(component.MessageUserUpdateSuccess) {
			t.Fatalf("normal profile messageCode = %v, want %s (body %s)",
				got, component.MessageUserUpdateSuccess, rec.Body.String())
		}
	})
}

// TestCourseReviewCreateRejectsSensitiveWord 课评命中敏感词 → 拦截返回
// course.review.sensitiveBlocked，不落库（无评价创建）。
func TestCourseReviewCreateRejectsSensitiveWord(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 903)
	alice := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, alice)
	persistSecurityPolicyConfig(t, conn, func() pageConfig.SecurityAndRegistration {
		cfg := emptySecurityConfig()
		cfg.SensitiveWords = []string{"代开发票", "赌博"}
		return cfg
	}())

	// 命中 → 拦截
	rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
		`{"offeringId":903,"rating":5,"content":"这课有代开发票广告"}`, token)
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if got := resp["messageCode"]; got != string(component.MessageCourseReviewSensitiveBanned) {
		t.Fatalf("sensitive review messageCode = %v, want %s (body %s)",
			got, component.MessageCourseReviewSensitiveBanned, rec.Body.String())
	}
	// 未落库
	var count int64
	conn.Model(&course.ReviewEntity{}).Where("offering_id = ?", 903).Count(&count)
	if count != 0 {
		t.Fatalf("sensitive review was persisted: count = %d, want 0", count)
	}

	// 正常评价通过
	rec = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
		`{"offeringId":903,"rating":4,"content":"好课"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("clean review status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

// TestCourseReviewUpdateRejectsSensitiveWord 课评编辑命中敏感词 → 拦截。
func TestCourseReviewUpdateRejectsSensitiveWord(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 904)
	alice := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, alice)
	persistSecurityPolicyConfig(t, conn, func() pageConfig.SecurityAndRegistration {
		cfg := emptySecurityConfig()
		cfg.SensitiveWords = []string{"代开发票"}
		return cfg
	}())

	// 先创建一条干净评价
	rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
		`{"offeringId":904,"rating":4,"content":"好课"}`, token)
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	result, _ := created["result"].(map[string]any)
	reviewID := uint64(result["id"].(float64))

	// 编辑为敏感内容 → 拦截
	rec = serveAuthSecurityJSON(router, http.MethodPatch,
		"/api/forum/course-reviews/"+strconv.FormatUint(reviewID, 10),
		`{"content":"需要代开发票请联系"}`, token)
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if got := resp["messageCode"]; got != string(component.MessageCourseReviewSensitiveBanned) {
		t.Fatalf("update sensitive messageCode = %v, want %s (body %s)",
			got, component.MessageCourseReviewSensitiveBanned, rec.Body.String())
	}
}

// TestCourseReviewUpdateSensitiveNotOwnedNoAuditLog 非作者 PATCH 他人课评为敏感内容：
// 归属预检先于敏感词检查与审核日志写入执行，返回 review.notOwned，且不产生挂在
// 他人课评上的 course_review 审核日志（codex review P2：防止伪造审核记录）。
func TestCourseReviewUpdateSensitiveNotOwnedNoAuditLog(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 905)
	alice := createHTTPContractUser(t, conn, contractTestID())
	aliceToken := contractSessionToken(t, alice)
	persistSecurityPolicyConfig(t, conn, func() pageConfig.SecurityAndRegistration {
		cfg := emptySecurityConfig()
		cfg.SensitiveWords = []string{"代开发票"}
		return cfg
	}())

	// alice 先创建一条干净评价
	rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
		`{"offeringId":905,"rating":4,"content":"好课"}`, aliceToken)
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	result, _ := created["result"].(map[string]any)
	reviewID := uint64(result["id"].(float64))

	// bob 对 alice 的课评 PATCH 敏感内容 → 归属预检先行，返回 review.notOwned
	bob := createHTTPContractUser(t, conn, contractTestID())
	bobToken := contractSessionToken(t, bob)
	rec = serveAuthSecurityJSON(router, http.MethodPatch,
		"/api/forum/course-reviews/"+strconv.FormatUint(reviewID, 10),
		`{"content":"需要代开发票请联系"}`, bobToken)
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if got := resp["messageCode"]; got != string(component.MessageReviewNotOwned) {
		t.Fatalf("non-owner sensitive PATCH messageCode = %v, want %s (body %s)",
			got, component.MessageReviewNotOwned, rec.Body.String())
	}

	// 不产生挂在他人课评上的 course_review 审核日志
	var logCount int64
	conn.Model(&moderationLog.Entity{}).
		Where("subject_type = ? AND subject_id = ?", moderationLog.SubjectCourseReview, reviewID).
		Count(&logCount)
	if logCount != 0 {
		t.Fatalf("course_review moderation log count = %d, want 0 (no forged audit entry)", logCount)
	}

	// 正文未被篡改
	var review course.ReviewEntity
	if err := conn.First(&review, reviewID).Error; err != nil {
		t.Fatalf("load review: %v", err)
	}
	if review.Content != "好课" {
		t.Fatalf("review content = %q, want 好课 (must stay untouched)", review.Content)
	}
}
