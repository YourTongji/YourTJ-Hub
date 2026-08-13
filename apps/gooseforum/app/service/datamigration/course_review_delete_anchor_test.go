package datamigration

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// v17：存量 deleted 行锚点回填。deleted_at 为空的 status=deleted 行应被
// 写入 now() 锚点，避免 COALESCE 回退 updated_at 造成窗口塌缩（review 发现）。
func TestBackfillCourseReviewDeleteAnchorsFillsLegacyRows(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&course.ReviewEntity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", []uint64{9_800_010_001, 9_800_010_002, 9_800_010_003}).Delete(&course.ReviewEntity{})
	})

	author := uint64(9_800_010_099)
	rating := 4
	// 存量已删行：deleted_at 为空（旧删除路径不写锚点）
	legacy := &course.ReviewEntity{
		Id:           9_800_010_001,
		OfferingId:   1,
		AuthorUserId: &author,
		Rating:       &rating,
		Content:      "存量删除正文",
		Status:       course.ReviewStatusDeleted,
	}
	if err := conn.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy deleted row: %v", err)
	}
	// 可见行不应被回填
	visible := &course.ReviewEntity{
		Id:           9_800_010_002,
		OfferingId:   2,
		AuthorUserId: &author,
		Rating:       &rating,
		Content:      "可见正文",
		Status:       course.ReviewStatusVisible,
	}
	if err := conn.Create(visible).Error; err != nil {
		t.Fatalf("create visible row: %v", err)
	}
	// 已有锚点的 deleted 行不应被改写（幂等）
	anchored := &course.ReviewEntity{
		Id:           9_800_010_003,
		OfferingId:   3,
		AuthorUserId: &author,
		Rating:       &rating,
		Content:      "已有锚点",
		Status:       course.ReviewStatusDeleted,
	}
	if err := conn.Create(anchored).Error; err != nil {
		t.Fatalf("create anchored row: %v", err)
	}
	oldAnchor := time.Now().Add(-60 * 24 * time.Hour)
	if err := conn.Model(anchored).Where("id = ?", anchored.Id).Update("deleted_at", oldAnchor).Error; err != nil {
		t.Fatalf("set anchor: %v", err)
	}

	result := BackfillCourseReviewDeleteAnchorsWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("backfill failed: %+v", result)
	}
	if result.Backfilled != 1 {
		t.Fatalf("backfilled = %d, want 1（只有锚点为空的 deleted 行被回填）", result.Backfilled)
	}

	var reloaded course.ReviewEntity
	if err := conn.First(&reloaded, legacy.Id).Error; err != nil {
		t.Fatalf("reload legacy row: %v", err)
	}
	if reloaded.DeletedAt == nil {
		t.Fatal("legacy row deleted_at still NULL after backfill")
	}
	if time.Since(*reloaded.DeletedAt) > time.Minute {
		t.Fatalf("backfilled anchor not now: %v", *reloaded.DeletedAt)
	}
	var reloadVisible, reloadAnchored course.ReviewEntity
	if err := conn.First(&reloadVisible, visible.Id).Error; err != nil {
		t.Fatalf("reload visible row: %v", err)
	}
	if reloadVisible.DeletedAt != nil {
		t.Fatalf("visible row got anchor: %v", *reloadVisible.DeletedAt)
	}
	if err := conn.First(&reloadAnchored, anchored.Id).Error; err != nil {
		t.Fatalf("reload anchored row: %v", err)
	}
	if reloadAnchored.DeletedAt == nil || time.Since(*reloadAnchored.DeletedAt) < 50*24*time.Hour {
		t.Fatalf("anchored row was rewritten, want original anchor preserved: %v", reloadAnchored.DeletedAt)
	}

	// 幂等：再次执行不再改写
	again := BackfillCourseReviewDeleteAnchorsWithDB(conn)
	if again.Failed > 0 || again.Backfilled != 0 {
		t.Fatalf("second backfill = %+v, want 0 backfilled", again)
	}
}

// 表不存在时静默跳过（全新库：AutoMigrate 建表前无 course_review）。
func TestBackfillCourseReviewDeleteAnchorsSkipsMissingTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:anchor-backfill-missing-table?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if db.Migrator().HasTable(&course.ReviewEntity{}) {
		t.Fatal("precondition: fresh db should not have course_review")
	}
	result := BackfillCourseReviewDeleteAnchorsWithDB(db)
	if result.Failed > 0 || result.Backfilled != 0 {
		t.Fatalf("missing table backfill = %+v, want clean skip", result)
	}
}
