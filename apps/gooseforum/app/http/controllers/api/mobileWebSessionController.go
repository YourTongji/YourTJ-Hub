package api

import (
	"net/http"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/authsessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
)

// MobileWebSession installs the native client's existing session in its private
// WebView. Only an explicit Bearer credential is accepted: browser cookies and
// query strings must never establish a session. The destination is allowlisted, and
// subsequent admin requests retain their normal role and writable-account gates.
func MobileWebSession(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	authorization := strings.Fields(c.GetHeader("Authorization"))
	if len(authorization) != 2 || !strings.EqualFold(authorization[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		return
	}
	userID, _, _, ok := authsessionservice.ValidateToken(authorization[1])
	if !ok {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		return
	}
	target := c.DefaultQuery("target", "admin")
	destination := "/admin"
	roleID, roleOK := userservice.GetUserRoleId(userID)
	allowed := roleOK && permission.CheckAnyRole(roleID)
	switch target {
	case "admin":
	case "moderation":
		destination = "/moderation"
		allowed = moderationservice.CanAccessModeration(userID)
	case "courseManagement", "courseReviews":
		allowed = roleOK && permission.CheckRole(roleID, permission.CourseManager)
		destination = "/moderation/courses"
		if target == "courseReviews" {
			destination = "/moderation/course-reviews"
		}
	default:
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessagePermissionDenied, nil))
		return
	}
	// Preserve the exact native session, including its expiry and revocation ID.
	// Normal authenticated requests perform sliding renewal after the handoff.
	jwtopt.TokenSetting(c, authorization[1])
	c.Writer.Header().Del("New-Token")
	c.Redirect(http.StatusSeeOther, destination)
}
