package forum

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

func TestPrivateMessageEvidenceRequiresAdministrator(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	const modID, adminID, roleID = uint64(982301), uint64(982302), uint64(982303)
	createLLMSModerator(t, conn, modID)
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: adminID, Username: "private_report_admin", RoleId: roleID, IsActivated: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&rolePermissionRs.Entity{RoleId: roleID, PermissionId: permission.Admin.Id(), Effective: 1}).Error; err != nil {
		t.Fatal(err)
	}
	permission.InvalidateRole(roleID)
	report := reports.Entity{TargetType: reports.TargetChatMessage, TargetId: 789, ReporterId: modID, Reason: reports.ReasonAbuse, Status: reports.StatusOpen, EvidenceSnapshot: reports.EvidenceSnapshotData{Excerpt: "private evidence", AuthorID: 44}}
	if err := conn.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Delete(&report)
		conn.Unscoped().Where("id = ?", adminID).Delete(&users.EntityComplete{})
		conn.Unscoped().Where("role_id = ?", roleID).Delete(&rolePermissionRs.Entity{})
		permission.InvalidateRole(roleID)
	})
	for _, id := range []uint64{0, modID} {
		if canModerateReportTarget(id, reports.TargetChatMessage, report.TargetId) {
			t.Fatalf("non-admin %d can handle private report", id)
		}
		if _, ok := buildModerationReportItem(id, 0, report, moderationReportBatchMaps{}); ok {
			t.Fatal("private snapshot leaked")
		}
	}
	items, _, _ := moderationReportPage(modID, reports.StatusOpen, 0, 0, 50)
	for _, item := range items {
		if item.TargetType == reports.TargetChatMessage {
			t.Fatal("global moderator saw private report")
		}
	}
	item, ok := buildModerationReportItem(adminID, 0, report, reportBatchMaps([]reports.Entity{report}))
	if !ok || item.Excerpt != "private evidence" || item.TargetURL != "/u/44" {
		t.Fatalf("admin evidence unavailable: %+v %v", item, ok)
	}
	response := UpdateModerationReportStatus(component.BetterRequest[ModerationReportStatusReq]{UserId: adminID, Params: ModerationReportStatusReq{Id: report.Id, Action: "resolve"}})
	if response.Data.Code != 0 {
		t.Fatalf("resolve: %+v", response)
	}
	got := reports.Get(report.Id)
	if got.Status != reports.StatusResolved || got.HandlerId != adminID || got.HandledAt == nil {
		t.Fatalf("missing handling audit: %+v", got)
	}
}
