package userservice

import (
	_ "embed"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/role"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

//go:embed welcomeTopic.md
var welcomeTopic string

func GetWelcomeTopicContent() string {
	return welcomeTopic
}

func FirstUserInit(adminUser *users.EntityComplete) {
	if adminUser.Id != 1 {
		return
	}

	roleEntity := role.Get(1)
	if roleEntity.Id == 0 {
		roleEntity.RoleName = "管理员"
		roleEntity.Effective = 1
		if err := role.SaveOrCreateById(&roleEntity); err != nil {
			slog.Error("create admin role failed", "error", err)
			return
		}
		slog.Info("created missing admin role")
	}

	rp := rolePermissionRs.GetRsByRoleIdAndPermission(roleEntity.Id, permission.Admin.Id())
	if rp.Id == 0 {
		rp.RoleId = roleEntity.Id
		rp.PermissionId = permission.Admin.Id()
		rp.Effective = 1
		rolePermissionRs.SaveOrCreateById(&rp)
		permission.InvalidateRole(roleEntity.Id)
		slog.Info("created missing admin role permission relation")
	}

	adminUser.RoleId = roleEntity.Id
	if err := SaveUser(adminUser); err != nil {
		slog.Error("save first admin user failed", "userId", adminUser.Id, "error", err)
	}
}
