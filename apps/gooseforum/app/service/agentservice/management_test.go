package agentservice

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"testing"
)

func TestAgentManagementOwnershipAndAccountState(t *testing.T) {
	setupAgentTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Where("1 = 1").Delete(&rolePermissionRs.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	owner := users.EntityComplete{Username: "owner", IsActivated: users.ActivationSuccess}
	stranger := users.EntityComplete{Username: "stranger", IsActivated: users.ActivationSuccess}
	admin := users.EntityComplete{Username: "admin", RoleId: 1, IsActivated: users.ActivationSuccess}
	bot := users.EntityComplete{Username: "bot", ActorType: users.ActorTypeBot, IsActivated: users.ActivationSuccess}
	for _, u := range []*users.EntityComplete{&owner, &stranger, &admin, &bot} {
		if err := conn.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.Create(&agents.Entity{UserId: bot.Id, CreatedBy: owner.Id}).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		id   uint64
		want bool
	}{{"owner", owner.Id, true}, {"admin", admin.Id, true}, {"stranger", stranger.Id, false}, {"bot", bot.Id, false}, {"missing", 99999, false}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanManage(tc.id, bot.Id); got != tc.want {
				t.Fatalf("CanManage=%v want %v", got, tc.want)
			}
			if got := CanAccessManagement(tc.id); got != tc.want {
				t.Fatalf("CanAccessManagement=%v want %v", got, tc.want)
			}
		})
	}
	if !CanManage(admin.Id, 99999) {
		t.Fatal("admin lookup should preserve not-found service errors")
	}
	if err := conn.Model(&stranger).Update("role_id", 777).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&rolePermissionRs.Entity{RoleId: 777, PermissionId: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if !CanManage(stranger.Id, bot.Id) || !CanAccessManagement(stranger.Id) {
		t.Fatal("custom administrator rejected")
	}
	for _, patch := range []map[string]any{{"is_frozen": users.StatusFrozen}, {"is_frozen": users.StatusNormal, "is_activated": users.ActivationPending}, {"is_activated": users.ActivationSuccess, "actor_type": users.ActorTypeBot}} {
		if err := conn.Model(&owner).Updates(patch).Error; err != nil {
			t.Fatal(err)
		}
		if CanManage(owner.Id, bot.Id) || CanAccessManagement(owner.Id) {
			t.Fatal("ineligible creator allowed")
		}
	}
	if err := conn.Model(&owner).Update("actor_type", users.ActorTypeHuman).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Delete(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if CanManage(owner.Id, bot.Id) {
		t.Fatal("deleted creator allowed")
	}
}
