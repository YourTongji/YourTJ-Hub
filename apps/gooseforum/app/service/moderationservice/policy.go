package moderationservice

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
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
func CheckUsernameAllowedWithConfig(username string, cfg pageConfig.SecurityAndRegistration) (component.MessageCode, error) {
	lowerUsername := strings.ToLower(username)
	for _, reserved := range cfg.ReservedUsernames {
		if strings.ToLower(reserved) == lowerUsername {
			return component.MessageAuthUsernameReserved, component.NewMessageError(component.MessageAuthUsernameReserved, "该用户名已被保留，不可使用", nil)
		}
	}
	for _, banned := range cfg.BannedUsernames {
		if strings.ToLower(banned) == lowerUsername {
			return component.MessageAuthUsernameBanned, component.NewMessageError(component.MessageAuthUsernameBanned, "该用户名已被禁用，不可使用", nil)
		}
	}
	return "", nil
}

// CheckContentAllowed 检查内容是否命中敏感词，返回是否命中及命中的敏感词。
func CheckContentAllowed(content string) (hit bool, word string) {
	return CheckContentAllowedWithConfig(content, hotdataserve.GetSecuritySettingsConfigCache())
}

// CheckContentAllowedWithConfig 使用给定配置检查内容，便于测试。
func CheckContentAllowedWithConfig(content string, cfg pageConfig.SecurityAndRegistration) (hit bool, word string) {
	lowerContent := strings.ToLower(content)
	for _, sensitive := range cfg.SensitiveWords {
		if sensitive != "" && strings.Contains(lowerContent, strings.ToLower(sensitive)) {
			return true, sensitive
		}
	}
	return false, ""
}

// FreezeUsersByBannedUsername 冻结与禁用用户名匹配（大小写不敏感）的存量账号，并写入审核日志。
func FreezeUsersByBannedUsername(username string, actorUserId uint64) error {
	user, err := findUserByUsernameCaseInsensitive(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if user.IsFrozen == users.StatusFrozen {
		return nil
	}
	user.IsFrozen = users.StatusFrozen
	if err := userservice.SaveUser(user); err != nil {
		return err
	}
	logUserFrozen(actorUserId, user.Id, user.Username)
	return nil
}

func findUserByUsernameCaseInsensitive(username string) (*users.EntityComplete, error) {
	var entity users.EntityComplete
	err := dbconnect.Connect().Model(&users.EntityComplete{}).
		Where("lower(username) = ?", strings.ToLower(username)).
		First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
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
