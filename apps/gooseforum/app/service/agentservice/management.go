package agentservice

import (
	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"gorm.io/gorm"
)

var ErrManagementDenied = errors.New("agent management denied")

func managementUser(tx *gorm.DB, userID uint64) (users.EntityComplete, bool) {
	var user users.EntityComplete
	err := tx.First(&user, userID).Error
	return user, err == nil && user.Id != 0 && user.ActorType == users.ActorTypeHuman && user.IsFrozen == users.StatusNormal && user.IsActivated == users.ActivationSuccess
}

// CanManage preserves bot login isolation: the human creator or station admin manages it.
func CanManage(userID, agentID uint64) bool {
	return CanManageTx(db.Connect(), userID, agentID)
}

// CanManageTx reads live ownership and account state in the caller's transaction.
func CanManageTx(tx *gorm.DB, userID, agentID uint64) bool {
	user, ok := managementUser(tx, userID)
	if !ok {
		return false
	}
	if user.RoleId == 1 || permission.CheckRoleTx(tx, user.RoleId, permission.Admin) {
		return true
	}
	agent := agents.GetByUserIDWithDB(tx, agentID)
	return agent != nil && agent.CreatedBy == userID
}

func CanAccessManagement(userID uint64) bool {
	tx := db.Connect()
	user, ok := managementUser(tx, userID)
	if !ok {
		return false
	}
	if user.RoleId == 1 || permission.CheckRoleTx(tx, user.RoleId, permission.Admin) {
		return true
	}
	var count int64
	return tx.Model(&agents.Entity{}).Where("created_by = ?", userID).Count(&count).Error == nil && count > 0
}
