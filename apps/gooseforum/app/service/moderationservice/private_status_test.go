package moderationservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

func TestPrivateReportsNotifyOnlyAdminsAndInvalidate(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &rolePermissionRs.Entity{}, &reports.Entity{}, &category.Entity{}); err != nil {
		t.Fatal(err)
	}
	const admin = uint64(980501)
	const moderator = uint64(980502)
	const role = uint64(980503)
	for _, user := range []users.EntityComplete{{Id: admin, Username: "private-report-admin", RoleId: role}, {Id: moderator, Username: "private-report-moderator"}} {
		if err := conn.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	grant := rolePermissionRs.Entity{RoleId: role, PermissionId: uint64(permission.Admin), Effective: 1}
	if err := conn.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	snapshotValue.Store(Snapshot{Grants: []Grant{{UserID: moderator, ScopeType: moderators.ScopeGlobal}}, ExpiresAt: time.Now().Add(time.Hour)})
	statusCache.Clear()
	t.Cleanup(func() {
		conn.Where("reporter_id = ?", admin).Delete(&reports.Entity{})
		conn.Unscoped().Delete(&users.EntityComplete{}, []uint64{admin, moderator})
		conn.Unscoped().Delete(&rolePermissionRs.Entity{}, grant.Id)
		userservice.InvalidateUserInfoCache(admin)
		userservice.InvalidateUserInfoCache(moderator)
		permission.InvalidateRole(role)
		statusCache.Clear()
		Invalidate()
	})
	if HasOpenReports(admin) {
		t.Fatal("unexpected initial report")
	}
	report := reports.Entity{TargetType: reports.TargetChatMessage, TargetId: 100, ReporterId: admin, Reason: reports.ReasonAbuse, Status: reports.StatusOpen}
	if err := conn.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	InvalidatePrivateReports()
	if !HasOpenReports(admin) {
		t.Fatal("private report did not notify administrator")
	}
	if HasOpenReports(moderator) {
		t.Fatal("private report status leaked to global moderator")
	}
	if err := reports.UpdateStatus(report.Id, reports.StatusResolved, "", admin); err != nil {
		t.Fatal(err)
	}
	InvalidatePrivateReports()
	if HasOpenReports(admin) {
		t.Fatal("resolved private report still notified administrator")
	}
}
