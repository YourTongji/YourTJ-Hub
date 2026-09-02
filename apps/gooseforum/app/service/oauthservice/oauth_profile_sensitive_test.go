package oauthservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// TestCreateUserFromOAuthClearsSensitiveBioAndBlog OAuth 建号时第三方资料自由文本
// （GitHub bio/blog）与 EditUserInfo 同规则过内容敏感词检查：命中即清空对应字段并写
// 审核日志（subject=user_profile），社交登录建号不被第三方简介阻断（codex review P2）。
func TestCreateUserFromOAuthClearsSensitiveBioAndBlog(t *testing.T) {
	conn := setupOAuthTestDB(t)
	if err := conn.AutoMigrate(&moderationLog.Entity{}); err != nil {
		t.Fatalf("migrate moderation_logs: %v", err)
	}
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		SensitiveWords: []string{"代开发票"},
	})

	user, err := createUserFromOAuth(OAuthUserInfo{
		ID: "uid-bio", Login: "bio-user", Provider: ProviderGitHub,
		Bio: "需要代开发票请联系", Blog: "https://example.com",
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth(sensitive bio) error = %v", err)
	}
	if user.Bio != "" {
		t.Fatalf("sensitive bio not cleared: %q", user.Bio)
	}
	if user.Website != "https://example.com" {
		t.Fatalf("clean blog should be kept, got %q", user.Website)
	}

	// 命中 blog 时同样清空并写日志
	user2, err := createUserFromOAuth(OAuthUserInfo{
		ID: "uid-blog", Login: "blog-user", Provider: ProviderGitHub,
		Bio: "正常简介", Blog: "https://example.com/代开发票",
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth(sensitive blog) error = %v", err)
	}
	if user2.Website != "" {
		t.Fatalf("sensitive blog not cleared: %q", user2.Website)
	}
	if user2.Bio != "正常简介" {
		t.Fatalf("clean bio should be kept, got %q", user2.Bio)
	}

	// 审核日志：两个命中账号各写一条 user_profile 敏感拦截记录
	var logs []moderationLog.Entity
	conn.Where("subject_type = ? AND action = ?",
		moderationLog.SubjectUserProfile, moderationLog.ActionSensitiveBlocked).
		Find(&logs)
	if len(logs) != 2 {
		t.Fatalf("user_profile sensitive log count = %d, want 2", len(logs))
	}
	for _, log := range logs {
		if log.SubjectId == 0 {
			t.Fatalf("user_profile log subjectId = 0, want the created user id")
		}
	}

	// 正常资料原样保留（无日志）
	before := int64(len(logs))
	user3, err := createUserFromOAuth(OAuthUserInfo{
		ID: "uid-clean", Login: "clean-user", Provider: ProviderGitHub,
		Bio: "普通简介", Blog: "https://example.com",
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth(clean profile) error = %v", err)
	}
	if user3.Bio != "普通简介" || user3.Website != "https://example.com" {
		t.Fatalf("clean profile mutated: bio=%q website=%q", user3.Bio, user3.Website)
	}
	var after int64
	conn.Model(&moderationLog.Entity{}).
		Where("subject_type = ? AND action = ?",
			moderationLog.SubjectUserProfile, moderationLog.ActionSensitiveBlocked).
		Count(&after)
	if after != before {
		t.Fatalf("clean profile produced a moderation log: %d -> %d", before, after)
	}

	// 账号全部真实落库（bio/blog 变体账号名避免重名退避干扰用户名断言）
	var total int64
	conn.Model(&users.EntityComplete{}).Count(&total)
	if total != 3 {
		t.Fatalf("user count = %d, want 3", total)
	}
}
