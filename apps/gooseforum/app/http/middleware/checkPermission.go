package middleware

import (
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
)

func CheckPermission(permissionType permission.Enum) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleId, ok := resolveRoleId(c)
		if !ok {
			return
		}

		if !permission.CheckRole(roleId, permissionType) {
			c.JSON(http.StatusForbidden, component.FailDataCode(
				component.MessagePermissionDenied,
				component.MessageParams{"permission": permissionType.LocalizedName(component.RequestLang(c))}))
			c.Abort()
			return
		}
		c.Next()
	}
}

func CheckAnyPermissionOrNotFound(c *gin.Context) {
	roleId, ok := resolveRoleId(c)
	if !ok {
		return
	}
	if !permission.CheckAnyRole(roleId) {
		forum.RenderNotFoundPage(c, component.MessagePageNotFound)
		c.Abort()
		return
	}
	c.Next()
}

// CheckWritableAccount 校验可写账号：登录态、账号存在、未冻结；邮箱验证开启时
// 待激活账号的写请求同样被拦截（issue #404：注册即发会话的 pending 账号不得
// 持有可用写权限）。激活恢复路径（resend-activation-email / set-user-email）
// 请改用 CheckWritableAccountAllowPendingActivation。
func CheckWritableAccount(c *gin.Context) {
	checkWritableAccount(c, false)
}

// CheckWritableAccountAllowPendingActivation 是 CheckWritableAccount 的恢复路径
// 变体：保留登录/冻结拦截，但不因待激活状态拒绝——仅用于激活恢复所需的重发
// 激活邮件与修改邮箱（改邮箱会把账号重新置为 pending，账号借此继续走恢复流程；
// 其余写操作仍被 CheckWritableAccount 拦截）。
func CheckWritableAccountAllowPendingActivation(c *gin.Context) {
	checkWritableAccount(c, true)
}

func checkWritableAccount(c *gin.Context, allowPendingActivation bool) {
	userId := c.GetUint64("userId")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}

	user, ok := userservice.GetUserInfo(userId)
	if !ok {
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessagePermissionResolveFailed, nil))
		c.Abort()
		return
	}
	if user.IsFrozen == users.StatusFrozen {
		c.JSON(http.StatusForbidden, component.FailDataCode(
			component.MessagePermissionUserFrozen,
			component.MessageParams{
				"action":     "写入",
				"actionCode": string(component.PermissionActionWrite),
			},
		))
		c.Abort()
		return
	}
	if !allowPendingActivation {
		securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
		if securityConfig.EnableEmailVerification && user.IsActivated == users.ActivationPending {
			c.JSON(http.StatusForbidden, component.FailDataCode(
				component.MessagePermissionEmailRequired,
				component.MessageParams{
					"action":     "写入",
					"actionCode": string(component.PermissionActionWrite),
				},
			))
			c.Abort()
			return
		}
	}
	c.Next()
}

func resolveRoleId(c *gin.Context) (uint64, bool) {
	userId := c.GetUint64("userId")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return 0, false
	}

	if val, exists := c.Get("roleId"); exists {
		if roleId, ok := val.(uint64); ok && roleId != 0 {
			return roleId, true
		}
	}

	roleId, ok := userservice.GetUserRoleId(userId)
	if !ok {
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessagePermissionResolveFailed, nil))
		c.Abort()
		return 0, false
	}
	return roleId, true
}
