package datamigration

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestBackfillMissingUserPointsPreservesExistingBalances(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}, &userPoints.Entity{}, &pointsRecord.Entity{}); err != nil {
		t.Fatalf("migrate fixtures: %v", err)
	}
	for _, user := range []users.EntityComplete{
		{Id: 1, Username: "existing-points"},
		{Id: 2, Username: "missing-points"},
		{Id: 3, Username: "missing-with-rewards"},
		{Id: 4, Username: "missing-with-ledger-init"},
		{Id: 5, Username: "bot-persona", ActorType: users.ActorTypeBot},
	} {
		if err := conn.Create(&user).Error; err != nil {
			t.Fatalf("create user %d: %v", user.Id, err)
		}
	}
	if err := conn.Create(&userPoints.Entity{UserId: 1, CurrentPoints: 345}).Error; err != nil {
		t.Fatalf("create existing balance: %v", err)
	}
	if err := conn.Create(&[]pointsRecord.Entity{
		{UserId: 3, Action: "topic_published", PointsChange: 10},
		{UserId: 3, Action: "post_created", PointsChange: 2},
		{UserId: 4, Action: "init", PointsChange: 100},
		{UserId: 4, Action: "post_created", PointsChange: 2},
	}).Error; err != nil {
		t.Fatalf("create ledger history: %v", err)
	}

	result := BackfillMissingUserPointsWithDB(conn)
	if result.Failed != 0 || result.Backfilled != 3 {
		t.Fatalf("backfill result = %+v, want three successes (bot excluded)", result)
	}
	var balances []userPoints.Entity
	if err := conn.Order("user_id").Find(&balances).Error; err != nil {
		t.Fatalf("load balances: %v", err)
	}
	if len(balances) != 4 ||
		balances[0].CurrentPoints != 345 ||
		balances[1].CurrentPoints != 100 ||
		balances[2].CurrentPoints != 112 ||
		balances[3].CurrentPoints != 102 {
		t.Fatalf("balances = %+v, want 345, 100, 112, 102 (bot 5 excluded)", balances)
	}
	var botBalance userPoints.Entity
	err = conn.Where("user_id = ?", 5).Take(&botBalance).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("bot user 5 should have no user_points row, got %v / %+v", err, botBalance)
	}
	var initRecords []pointsRecord.Entity
	if err := conn.Where("action = ?", "init").Order("user_id").Find(&initRecords).Error; err != nil {
		t.Fatalf("load init records: %v", err)
	}
	if len(initRecords) != 3 ||
		initRecords[0].UserId != 2 ||
		initRecords[1].UserId != 3 ||
		initRecords[2].UserId != 4 {
		t.Fatalf("init records = %+v, want backfilled users 2 and 3 plus existing user 4", initRecords)
	}

	second := BackfillMissingUserPointsWithDB(conn)
	if second.Failed != 0 || second.Backfilled != 0 {
		t.Fatalf("second backfill result = %+v, want no-op", second)
	}
}
