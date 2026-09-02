package oauthservice

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// TestCreateUserFromOAuthBacksOffReservedAndBanned 社交登录对保留/禁用用户名
// 做退避建号（<name>_<n>）而非拒绝登录（最小惊讶）；退避候选同样过名单。
func TestCreateUserFromOAuthBacksOffReservedAndBanned(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		ReservedUsernames: []string{"administrator"},
		BannedUsernames:   []string{"spammer"},
	})

	// 保留名 → 退避为 administrator_1
	user, err := createUserFromOAuth(OAuthUserInfo{
		ID: "uid-reserved", Login: "administrator", Provider: ProviderGitHub,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth(reserved) error = %v", err)
	}
	if user.Username != "administrator_1" {
		t.Fatalf("reserved username = %q, want administrator_1", user.Username)
	}

	// 禁用名 → 退避为 spammer_1
	user2, err := createUserFromOAuth(OAuthUserInfo{
		ID: "uid-banned", Login: "spammer", Provider: ProviderGitHub,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth(banned) error = %v", err)
	}
	if user2.Username != "spammer_1" {
		t.Fatalf("banned username = %q, want spammer_1", user2.Username)
	}

	// 普通名 → 原样保留
	user3, err := createUserFromOAuth(OAuthUserInfo{
		ID: "uid-normal", Login: "normal-user", Provider: ProviderGitHub,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth(normal) error = %v", err)
	}
	if user3.Username != "normal-user" {
		t.Fatalf("normal username = %q, want normal-user", user3.Username)
	}

	// 三个账号都真实落库
	conn := db.Connect()
	var count int64
	conn.Model(&users.EntityComplete{}).Count(&count)
	if count != 3 {
		t.Fatalf("user count = %d, want 3", count)
	}
}
