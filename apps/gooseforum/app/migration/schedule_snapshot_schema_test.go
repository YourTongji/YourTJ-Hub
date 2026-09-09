package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestSchemaModelsRegistersScheduleSnapshot 验证 SchemaModels 已注册
// pk.ScheduleSnapshotEntity（生产迁移入口依赖该注册，注册丢失则表永远不会创建）。
func TestSchemaModelsRegistersScheduleSnapshot(t *testing.T) {
	for _, model := range SchemaModels() {
		if _, ok := model.(*pk.ScheduleSnapshotEntity); ok {
			return
		}
	}
	t.Fatal("SchemaModels() does not include *pk.ScheduleSnapshotEntity")
}

// TestScheduleSnapshotSchemaCreatedOnSQLite 验证全新库上 pk_schedule_snapshot
// 表可创建，结构完整（user_id 唯一、四数据列非空）且可正常写入/查询。
func TestScheduleSnapshotSchemaCreatedOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-schedule-snapshot-create?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&pk.ScheduleSnapshotEntity{}); err != nil {
		t.Fatalf("AutoMigrate pk_schedule_snapshot: %v", err)
	}
	if !conn.Migrator().HasTable("pk_schedule_snapshot") {
		t.Fatal("pk_schedule_snapshot table missing after AutoMigrate")
	}
	for _, column := range []string{"id", "user_id", "plans", "active_plan_id", "major_selected", "week_view", "created_at", "updated_at"} {
		if !conn.Migrator().HasColumn(&pk.ScheduleSnapshotEntity{}, column) {
			t.Errorf("pk_schedule_snapshot column %q missing after AutoMigrate", column)
		}
	}

	// 写入/查询：JSON 列经 serializer 编码，user_id 唯一约束生效。
	if err := conn.Create(&pk.ScheduleSnapshotEntity{UserId: 1, ActivePlanId: "plan_a"}).Error; err != nil {
		t.Fatalf("insert first snapshot: %v", err)
	}
	dup := pk.ScheduleSnapshotEntity{UserId: 1, ActivePlanId: "plan_a"}
	if err := conn.Create(&dup).Error; err == nil {
		t.Fatal("duplicate user_id insert succeeded, want unique constraint error")
	}
	var count int64
	if err := conn.Model(&pk.ScheduleSnapshotEntity{}).Where("user_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if count != 1 {
		t.Fatalf("snapshot count = %d, want 1", count)
	}
}

// TestScheduleSnapshotSchemaUpgradeFromLegacySubset 模拟存量实例升级：旧库没有
// pk_schedule_snapshot 表，AutoMigrate 必须自动补齐新表且不破坏旧表与存量数据。
func TestScheduleSnapshotSchemaUpgradeFromLegacySubset(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-schedule-snapshot-upgrade?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if conn.Migrator().HasTable("pk_schedule_snapshot") {
		t.Fatal("precondition failed: fresh schema should not have pk_schedule_snapshot")
	}
	if err := conn.AutoMigrate(&pk.ScheduleSnapshotEntity{}); err != nil {
		t.Fatalf("upgrade AutoMigrate pk_schedule_snapshot: %v", err)
	}
	if !conn.Migrator().HasTable("pk_schedule_snapshot") {
		t.Fatal("pk_schedule_snapshot table missing after upgrade")
	}
	// 新表可用（含 JSON 列读写）。
	if err := conn.Create(&pk.ScheduleSnapshotEntity{UserId: 7, ActivePlanId: "plan_b"}).Error; err != nil {
		t.Fatalf("insert snapshot after upgrade: %v", err)
	}
	var got pk.ScheduleSnapshotEntity
	if err := conn.Where("user_id = ?", 7).First(&got).Error; err != nil || got.ActivePlanId != "plan_b" {
		t.Fatalf("read snapshot after upgrade: %+v err=%v", got, err)
	}
}
