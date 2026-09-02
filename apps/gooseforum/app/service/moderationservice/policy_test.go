package moderationservice

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestCheckUsernameAllowedWithConfig(t *testing.T) {
	cfg := pageConfig.SecurityAndRegistration{
		ReservedUsernames: []string{"Admin", "Root"},
		BannedUsernames:   []string{"spammer", "bot"},
	}

	tests := []struct {
		name     string
		username string
		wantCode component.MessageCode
	}{
		{name: "reserved exact", username: "Admin", wantCode: component.MessageAuthUsernameReserved},
		{name: "reserved case-insensitive", username: "admin", wantCode: component.MessageAuthUsernameReserved},
		{name: "reserved uppercase", username: "ROOT", wantCode: component.MessageAuthUsernameReserved},
		{name: "banned exact", username: "spammer", wantCode: component.MessageAuthUsernameBanned},
		{name: "banned case-insensitive", username: "Spammer", wantCode: component.MessageAuthUsernameBanned},
		{name: "reserved leet variant", username: "adm1n", wantCode: component.MessageAuthUsernameReserved},
		{name: "reserved full-width", username: "ａｄｍｉｎ", wantCode: component.MessageAuthUsernameReserved},
		{name: "reserved zero-width", username: "a\u200bdmin", wantCode: component.MessageAuthUsernameReserved},
		{name: "no substring", username: "myadmin", wantCode: ""},
		{name: "no prefix", username: "rootuser_x", wantCode: ""},
		{name: "no hit", username: "normal_user", wantCode: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := CheckUsernameAllowedWithConfig(tt.username, cfg)
			if code != tt.wantCode {
				t.Fatalf("CheckUsernameAllowedWithConfig(%q) code = %q, want %q", tt.username, code, tt.wantCode)
			}
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("CheckUsernameAllowedWithConfig(%q) error = %v, want nil", tt.username, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("CheckUsernameAllowedWithConfig(%q) error = nil, want non-nil", tt.username)
			}
		})
	}
}

func TestCheckContentAllowedWithConfig(t *testing.T) {
	cfg := pageConfig.SecurityAndRegistration{
		SensitiveWords: []string{"赌博", "代考", "spammer"},
	}

	tests := []struct {
		name     string
		content  string
		wantHit  bool
		wantWord string
	}{
		{name: "hit first word", content: "一起来讨论赌博问题", wantHit: true, wantWord: "赌博"},
		{name: "hit second word", content: "代考服务", wantHit: true, wantWord: "代考"},
		{name: "hit case-insensitive", content: "SPAMMER 内容", wantHit: true, wantWord: "spammer"},
		{name: "hit full-width", content: "全角ＳＰＡＭＭＥＲ", wantHit: true, wantWord: "spammer"},
		{name: "hit zero-width evasion", content: "赌\u200b博", wantHit: true, wantWord: "赌博"},
		{name: "no leet fold on content", content: "sp4mmer", wantHit: false, wantWord: ""},
		{name: "no hit", content: "正常内容", wantHit: false, wantWord: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit, word := CheckContentAllowedWithConfig(tt.content, cfg)
			if hit != tt.wantHit {
				t.Fatalf("CheckContentAllowedWithConfig(%q) hit = %v, want %v", tt.content, hit, tt.wantHit)
			}
			if word != tt.wantWord {
				t.Fatalf("CheckContentAllowedWithConfig(%q) word = %q, want %q", tt.content, word, tt.wantWord)
			}
		})
	}
}

func TestCheckContentAllowedWithConfigEmptyList(t *testing.T) {
	cfg := pageConfig.SecurityAndRegistration{}
	hit, word := CheckContentAllowedWithConfig("任意内容", cfg)
	if hit {
		t.Fatalf("hit = true with empty word list, want false")
	}
	if word != "" {
		t.Fatalf("word = %q with empty word list, want empty", word)
	}
}

func setupPolicyTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &moderationLog.Entity{}); err != nil {
		t.Fatalf("migrate policy tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&users.EntityComplete{})
	conn.Unscoped().Where("1 = 1").Delete(&moderationLog.Entity{})
}

func TestFreezeUsersByBannedUsername(t *testing.T) {
	setupPolicyTestDB(t)

	conn := db.Connect()
	user := users.EntityComplete{Username: "BanMe", Email: "banme@example.com"}
	if err := users.Create(&user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 大小写不敏感命中并冻结
	if err := FreezeUsersByBannedUsername("banme", 0); err != nil {
		t.Fatalf("FreezeUsersByBannedUsername() error = %v", err)
	}
	frozen, err := users.GetByUsername("BanMe")
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if frozen.IsFrozen != users.StatusFrozen {
		t.Fatalf("IsFrozen = %d, want %d", frozen.IsFrozen, users.StatusFrozen)
	}

	// 已冻结用户重复调用不报错
	if err := FreezeUsersByBannedUsername("BanMe", 0); err != nil {
		t.Fatalf("FreezeUsersByBannedUsername() second call error = %v", err)
	}

	// 不存在的用户名返回 nil
	if err := FreezeUsersByBannedUsername("nobody", 0); err != nil {
		t.Fatalf("FreezeUsersByBannedUsername() missing user error = %v", err)
	}

	// 审核日志写入一条 userFrozen 记录
	var logs []moderationLog.Entity
	conn.Where("subject_type = ? AND subject_id = ?", moderationLog.SubjectUser, user.Id).Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("moderation logs count = %d, want 1", len(logs))
	}
	if logs[0].Action != moderationLog.ActionUserFrozen {
		t.Fatalf("log action = %q, want %q", logs[0].Action, moderationLog.ActionUserFrozen)
	}
	if logs[0].ActorUserId != 0 {
		t.Fatalf("log actorUserId = %d, want 0 (system)", logs[0].ActorUserId)
	}
}

func TestCheckNicknameAllowedWithConfig(t *testing.T) {
	cfg := pageConfig.SecurityAndRegistration{
		ReservedUsernames: []string{"Admin", "官方"},
		BannedUsernames:   []string{"spammer"},
	}

	tests := []struct {
		name     string
		nickname string
		wantCode component.MessageCode
	}{
		{name: "reserved exact", nickname: "Admin", wantCode: component.MessageAuthNicknameReserved},
		{name: "reserved lowercase", nickname: "admin", wantCode: component.MessageAuthNicknameReserved},
		{name: "reserved chinese", nickname: "官方", wantCode: component.MessageAuthNicknameReserved},
		{name: "banned", nickname: "SPAMMER", wantCode: component.MessageAuthNicknameBanned},
		{name: "leet variant", nickname: "adm1n", wantCode: component.MessageAuthNicknameReserved},
		{name: "full-width", nickname: "ａｄｍｉｎ", wantCode: component.MessageAuthNicknameReserved},
		{name: "no hit", nickname: "普通鹅友", wantCode: ""},
		{name: "no substring", nickname: "小admin迷", wantCode: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := CheckNicknameAllowedWithConfig(tt.nickname, cfg)
			if code != tt.wantCode {
				t.Fatalf("CheckNicknameAllowedWithConfig(%q) code = %q, want %q", tt.nickname, code, tt.wantCode)
			}
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("CheckNicknameAllowedWithConfig(%q) error = %v, want nil", tt.nickname, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("CheckNicknameAllowedWithConfig(%q) error = nil, want non-nil", tt.nickname)
			}
		})
	}
}

// TestFreezeUsersByBannedUsernamesNormalizedVariants 冻结走与策略检查相同的
// 归一化整串全等（NameOptions：大小写/NFKC/零宽/leet）：管理员新增禁用词 admin
// 时，存量 Adm1n/ａｄｍｉｎ/a\u200bdmin 变体账号一并冻结（review comment P1）。
func TestFreezeUsersByBannedUsernamesNormalizedVariants(t *testing.T) {
	setupPolicyTestDB(t)

	conn := db.Connect()
	makeUser := func(username string) *users.EntityComplete {
		t.Helper()
		user := users.EntityComplete{Username: username, Email: username + "@example.com"}
		if err := users.Create(&user); err != nil {
			t.Fatalf("create user %s: %v", username, err)
		}
		return &user
	}
	leetUser := makeUser("Adm1n")
	wideUser := makeUser("ａｄｍｉｎ")
	zwUser := makeUser("a\u200bdmin")
	normalUser := makeUser("normal_user")
	makeUser("myadmin") // 子串不应命中（整串全等语义）

	if err := FreezeUsersByBannedUsernames([]string{"admin"}, 0); err != nil {
		t.Fatalf("FreezeUsersByBannedUsernames() error = %v", err)
	}

	for _, tt := range []struct {
		name string
		user *users.EntityComplete
		want int8
	}{
		{name: "leet variant frozen", user: leetUser, want: users.StatusFrozen},
		{name: "full-width variant frozen", user: wideUser, want: users.StatusFrozen},
		{name: "zero-width variant frozen", user: zwUser, want: users.StatusFrozen},
		{name: "unrelated stays normal", user: normalUser, want: users.StatusNormal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reloaded, err := users.Get(tt.user.Id)
			if err != nil {
				t.Fatalf("reload user %d: %v", tt.user.Id, err)
			}
			if reloaded.IsFrozen != tt.want {
				t.Fatalf("user %q IsFrozen = %d, want %d", reloaded.Username, reloaded.IsFrozen, tt.want)
			}
		})
	}

	// 已冻结跳过幂等 + 日志恰好按命中账号计数（3 个变体账号命中 admin → 3 条日志；
	// myadmin 因整串全等语义不命中）。
	if err := FreezeUsersByBannedUsernames([]string{"admin"}, 0); err != nil {
		t.Fatalf("second FreezeUsersByBannedUsernames() error = %v", err)
	}
	var count int64
	conn.Model(&moderationLog.Entity{}).
		Where("action = ?", moderationLog.ActionUserFrozen).
		Count(&count)
	if count != 3 {
		t.Fatalf("userFrozen log count = %d, want 3", count)
	}
	// myadmin 明确不被冻结（子串不命中，整串全等语义护栏）。
	myadmin, err := users.GetByUsername("myadmin")
	if err != nil {
		t.Fatalf("reload myadmin: %v", err)
	}
	if myadmin.IsFrozen != users.StatusNormal {
		t.Fatalf("myadmin IsFrozen = %d, want %d (no substring match)", myadmin.IsFrozen, users.StatusNormal)
	}
}
