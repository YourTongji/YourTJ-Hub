package reports

import (
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
)

func TestClearExpiredEvidenceSnapshots(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&Entity{}, &topics.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// clean fixtures
	conn.Where("id IN ?", []uint64{880001, 880002, 880003, 880004}).Delete(&Entity{})
	conn.Where("id IN ?", []uint64{880101, 880102, 880103}).Delete(&topics.Entity{})

	oldHandled := time.Now().Add(-200 * 24 * time.Hour)
	snap := EvidenceSnapshotData{TargetType: TargetTopic, TargetID: 1, Title: "kept-for-ttl-test", CreatedAt: time.Now()}

	// topic without hold
	if err := conn.Create(&topics.Entity{Id: 880101, Title: "normal", RetentionStatus: topics.RetentionNormal, Status: 1}).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	// topic with legal hold
	if err := conn.Create(&topics.Entity{Id: 880102, Title: "hold", RetentionStatus: topics.RetentionLegalHold, Status: 1}).Error; err != nil {
		t.Fatalf("create hold topic: %v", err)
	}
	// topic with evidence hold
	if err := conn.Create(&topics.Entity{Id: 880103, Title: "evidence-hold", RetentionStatus: topics.RetentionEvidenceHold, Status: 1}).Error; err != nil {
		t.Fatalf("create evidence-hold topic: %v", err)
	}

	closed := Entity{
		Id: 880001, TargetType: TargetTopic, TargetId: 880101, TopicId: 880101,
		ReporterId: 1, Reason: ReasonSpam, Status: StatusResolved, Resolution: ResolutionBanned,
		HandlerId: 1, HandledAt: &oldHandled, EvidenceSnapshot: snap,
	}
	openRep := Entity{
		Id: 880002, TargetType: TargetTopic, TargetId: 880101, TopicId: 880101,
		ReporterId: 2, Reason: ReasonSpam, Status: StatusOpen, EvidenceSnapshot: snap,
	}
	held := Entity{
		Id: 880003, TargetType: TargetTopic, TargetId: 880102, TopicId: 880102,
		ReporterId: 3, Reason: ReasonIllegal, Status: StatusResolved, Resolution: ResolutionBanned,
		HandlerId: 1, HandledAt: &oldHandled, EvidenceSnapshot: snap,
	}
	evidenceHeld := Entity{
		Id: 880004, TargetType: TargetTopic, TargetId: 880103, TopicId: 880103,
		ReporterId: 4, Reason: ReasonSpam, Status: StatusRejected, Resolution: ResolutionIgnored,
		HandlerId: 1, HandledAt: &oldHandled, EvidenceSnapshot: snap,
	}
	for _, e := range []Entity{closed, openRep, held, evidenceHeld} {
		if err := conn.Create(&e).Error; err != nil {
			t.Fatalf("create report %d: %v", e.Id, err)
		}
	}

	cleared, err := ClearExpiredEvidenceSnapshots(time.Now().Add(-180*24*time.Hour), 50)
	if err != nil {
		t.Fatalf("ClearExpiredEvidenceSnapshots: %v", err)
	}
	if cleared < 1 {
		t.Fatalf("cleared=%d, want >=1", cleared)
	}

	gotClosed := Get(880001)
	if gotClosed.EvidenceSnapshot.Title != "" {
		t.Fatalf("closed snapshot title=%q, want empty after TTL", gotClosed.EvidenceSnapshot.Title)
	}
	gotOpen := Get(880002)
	if gotOpen.EvidenceSnapshot.Title != "kept-for-ttl-test" {
		t.Fatalf("open snapshot cleared unexpectedly: %q", gotOpen.EvidenceSnapshot.Title)
	}
	gotHeld := Get(880003)
	if gotHeld.EvidenceSnapshot.Title != "kept-for-ttl-test" {
		t.Fatalf("LEGAL_HOLD snapshot cleared unexpectedly: %q", gotHeld.EvidenceSnapshot.Title)
	}
	gotEvidenceHeld := Get(880004)
	if gotEvidenceHeld.EvidenceSnapshot.Title != "kept-for-ttl-test" {
		t.Fatalf("EVIDENCE_HOLD snapshot cleared unexpectedly: %q", gotEvidenceHeld.EvidenceSnapshot.Title)
	}
}
