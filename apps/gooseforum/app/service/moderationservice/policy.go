package moderationservice

import (
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/wordmatch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

// CheckUsernameAllowed 检查用户名是否命中保留或禁用名单，未命中时返回空错误码与 nil。
func CheckUsernameAllowed(username string) (component.MessageCode, error) {
	return CheckUsernameAllowedWithConfig(username, hotdataserve.GetSecuritySettingsConfigCache())
}

// CheckUsernameAllowedWithConfig 使用给定配置检查用户名，便于测试。
// 匹配为整串归一化后全等（wordmatch.NameOptions：大小写/NFKC/零宽折叠 +
// ASCII leetspeak 折叠），不子串匹配，避免误伤 myadmin 之类合法名。
func CheckUsernameAllowedWithConfig(username string, cfg pageConfig.SecurityAndRegistration) (component.MessageCode, error) {
	reserved := wordmatch.Compile(cfg.ReservedUsernames, wordmatch.NameOptions)
	if _, ok := reserved.Equal(username); ok {
		return component.MessageAuthUsernameReserved, component.NewMessageError(component.MessageAuthUsernameReserved, "该用户名已被保留，不可使用", nil)
	}
	banned := wordmatch.Compile(cfg.BannedUsernames, wordmatch.NameOptions)
	if _, ok := banned.Equal(username); ok {
		return component.MessageAuthUsernameBanned, component.NewMessageError(component.MessageAuthUsernameBanned, "该用户名已被禁用，不可使用", nil)
	}
	return "", nil
}

// CheckNicknameAllowed 检查昵称是否命中保留或禁用名单（与用户名同一份名单、
// 同一归一化全等规则，防止 "官方/客服/管理员" 之类冒充性昵称），未命中返回空错误码与 nil。
func CheckNicknameAllowed(nickname string) (component.MessageCode, error) {
	return CheckNicknameAllowedWithConfig(nickname, hotdataserve.GetSecuritySettingsConfigCache())
}

// CheckNicknameAllowedWithConfig 使用给定配置检查昵称，便于测试。
func CheckNicknameAllowedWithConfig(nickname string, cfg pageConfig.SecurityAndRegistration) (component.MessageCode, error) {
	reserved := wordmatch.Compile(cfg.ReservedUsernames, wordmatch.NameOptions)
	if _, ok := reserved.Equal(nickname); ok {
		return component.MessageAuthNicknameReserved, component.NewMessageError(component.MessageAuthNicknameReserved, "该昵称已被保留，不可使用", nil)
	}
	banned := wordmatch.Compile(cfg.BannedUsernames, wordmatch.NameOptions)
	if _, ok := banned.Equal(nickname); ok {
		return component.MessageAuthNicknameBanned, component.NewMessageError(component.MessageAuthNicknameBanned, "该昵称已被禁用，不可使用", nil)
	}
	return "", nil
}

// CheckContentAllowed 检查内容是否命中敏感词，返回是否命中及命中的敏感词。
func CheckContentAllowed(content string) (hit bool, word string) {
	return CheckContentAllowedWithConfig(content, hotdataserve.GetSecuritySettingsConfigCache())
}

// CheckContentAllowedWithConfig 使用给定配置检查内容，便于测试。
// 词表经 wordmatch.ContentOptions（大小写/NFKC/零宽折叠，不含 leetspeak）
// 归一化后单遍 AC 扫描；命中返回配置中的原词。
func CheckContentAllowedWithConfig(content string, cfg pageConfig.SecurityAndRegistration) (hit bool, word string) {
	matcher := wordmatch.Compile(cfg.SensitiveWords, wordmatch.ContentOptions)
	word, hit = matcher.Find(content)
	return hit, word
}

// FreezeUsersByBannedUsername 冻结与单个禁用词（归一化整串全等，与策略检查同规则）
// 匹配的存量账号。保留该单数入口以兼容既有调用/测试，内部复用批量实现。
func FreezeUsersByBannedUsername(username string, actorUserId uint64) error {
	if strings.TrimSpace(username) == "" {
		return nil
	}
	return FreezeUsersByBannedUsernames([]string{username}, actorUserId)
}

// FreezeUsersByBannedUsernames 冻结与任一禁用词（wordmatch.NameOptions 归一化整串
// 全等：大小写/NFKC 全半角/零宽/ASCII leetspeak 折叠）匹配的存量账号，并写入审核
// 日志。与策略层 CheckUsernameAllowed 使用同一归一化语义：管理员新增禁用词 admin
// 时，存量 Adm1n/ａｄｍｉｎ/a\u200bdmin 等归一化变体账号同样会被冻结，避免出现
// 「策略判定禁用但存量变体账号仍活跃」的口径分裂。冻结幂等（已冻结跳过，不重复
// 写日志）；无匹配账号返回 nil。整表扫描仅在管理端保存禁用名单时触发（低频管理
// 动作），单次调用编译全部新词后只扫描一遍。
func FreezeUsersByBannedUsernames(bannedUsernames []string, actorUserId uint64) error {
	matcher := wordmatch.Compile(bannedUsernames, wordmatch.NameOptions)
	if matcher.Len() == 0 {
		return nil
	}
	var list []users.EntityComplete
	if err := dbconnect.Connect().Model(&users.EntityComplete{}).Find(&list).Error; err != nil {
		return err
	}
	for i := range list {
		user := &list[i]
		if user.IsFrozen == users.StatusFrozen {
			continue
		}
		if _, ok := matcher.Equal(user.Username); !ok {
			continue
		}
		user.IsFrozen = users.StatusFrozen
		if err := userservice.SaveUser(user); err != nil {
			return err
		}
		logUserFrozen(actorUserId, user.Id, user.Username)
	}
	return nil
}

// logUserFrozen 记录用户被冻结的审核日志。
func logUserFrozen(actorUserId uint64, userId uint64, username string) {
	create(moderationLog.Entity{
		ActorUserId: actorUserId,
		Action:      moderationLog.ActionUserFrozen,
		SubjectType: moderationLog.SubjectUser,
		SubjectId:   userId,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.user.frozen",
			Params: map[string]any{
				"userId":   userId,
				"username": username,
			},
		},
	})
}
