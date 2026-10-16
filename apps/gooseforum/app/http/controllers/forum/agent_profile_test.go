package forum

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
)

func TestAgentProfileManagementEntryUsesLiveHumanPermissions(t *testing.T) {
	conn := setupRevisionTestDB(t)
	if err := conn.AutoMigrate(&agents.Entity{}); err != nil {
		t.Fatal(err)
	}
	owner := users.EntityComplete{Id: 979801, Username: "profile-agent-owner", IsActivated: users.ActivationSuccess}
	admin := users.EntityComplete{Id: 979802, Username: "profile-agent-admin", RoleId: 1, IsActivated: users.ActivationSuccess}
	stranger := users.EntityComplete{Id: 979803, Username: "profile-agent-stranger", IsActivated: users.ActivationSuccess}
	bot := users.EntityComplete{Id: 979804, Username: "profile-agent-bot", ActorType: users.ActorTypeBot, IsActivated: users.ActivationSuccess}
	agent := agents.Entity{UserId: bot.Id, CreatedBy: owner.Id}
	for _, row := range []any{&owner, &admin, &stranger, &bot, &agent} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(row) })
	}
	check := func(viewer uint64, target users.EntityComplete, want bool) {
		t.Helper()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/u/979804", nil)
		c.Set("userId", viewer)
		props := buildUserProfileProps(c, target, userProfileSectionBadges, userProfileActivityTimeline)
		if props.CanManageAgent != want {
			t.Fatalf("viewer=%d target=%d canManageAgent=%v want %v", viewer, target.Id, props.CanManageAgent, want)
		}
	}
	for _, viewer := range []uint64{owner.Id, admin.Id} {
		check(viewer, bot, true)
	}
	for _, viewer := range []uint64{0, stranger.Id, bot.Id} {
		check(viewer, bot, false)
	}
	check(admin.Id, owner, false)
	if err := conn.Model(&owner).Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatal(err)
	}
	check(owner.Id, bot, false)
}
