package courseservice

import (
	"context"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// backdateReviewDelete 把 deleted_at/updated_at 回拨到隔离窗口之外，模拟
// "删除已超 30 天"。裸 Table().Update：无 schema 无 autoUpdateTime，
// 两列均真实写入（判别性要求，spec N1 教训）。
func backdateReviewDelete(t *testing.T, reviewID uint64, ago time.Duration) {
	t.Helper()
	conn := dbconnect.Connect()
	ts := time.Now().Add(-ago)
	if err := conn.Table((&course.ReviewEntity{}).TableName()).
		Where("id = ?", reviewID).
		Updates(map[string]any{"deleted_at": ts, "updated_at": ts}).Error; err != nil {
		t.Fatalf("backdate review delete: %v", err)
	}
}

// loadReviewRow 用 Table 查询读取行（课评领域不启用 GORM 软删）。
func loadReviewRow(t *testing.T, reviewID uint64) course.ReviewEntity {
	t.Helper()
	conn := dbconnect.Connect()
	var ent course.ReviewEntity
	if err := conn.Table((&course.ReviewEntity{}).TableName()).
		Where("id = ?", reviewID).First(&ent).Error; err != nil {
		t.Fatalf("load review %d: %v", reviewID, err)
	}
	return ent
}

// TestCleanupExpiredReviewClearsContentAndAuthor 验收 1：删除超 30 天 →
// content 为空、作者关联断开（NULL，释放占位）、行保留（可审计）、幂等。
func TestCleanupExpiredReviewClearsContentAndAuthor(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "将被清理的正文", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	backdateReviewDelete(t, payload.Id, ReviewCleanupRetentionDays*24*time.Hour+24*time.Hour)

	cleaned, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("cleaned = %d, want 1", cleaned)
	}
	ent := loadReviewRow(t, payload.Id)
	if ent.Status != course.ReviewStatusDeleted {
		t.Fatalf("status after cleanup = %d, want deleted（行保留可审计）", ent.Status)
	}
	if ent.Content != "" {
		t.Fatalf("content after cleanup = %q, want empty", ent.Content)
	}
	if ent.AuthorID() != 0 {
		t.Fatalf("author after cleanup = %d, want 0（作者关联断开）", ent.AuthorID())
	}
	if ent.AuthorUserId != nil {
		t.Fatalf("author_user_id after cleanup = %v, want NULL（唯一索引占位释放）", *ent.AuthorUserId)
	}
	// 统计投影不受清理影响（删除时已扣减，清理不改 status）。
	courseStats, err := course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 0 || courseStats.RatingSum != 0 || courseStats.ReviewCount != 0 {
		t.Fatalf("stats corrupted by cleanup: %+v", courseStats)
	}
	// 幂等：再次清理不再选中该行（author_user_id IS NULL 谓词）。
	again, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("second cleanup: %v", err)
	}
	if again != 0 {
		t.Fatalf("second cleanup cleaned = %d, want 0（幂等）", again)
	}
}

// TestCleanupSkipsRecentDelete 验收 2：删除未到隔离窗口的行不被清理。
func TestCleanupSkipsRecentDelete(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "刚删除的正文", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	// 不回拨：deleted_at 为当前时间（窗口内）。

	cleaned, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 0 {
		t.Fatalf("cleaned = %d, want 0（窗口内不动）", cleaned)
	}
	ent := loadReviewRow(t, payload.Id)
	if ent.Content != "刚删除的正文" || ent.AuthorID() != 1001 {
		t.Fatalf("window-internal review was modified: %+v", ent)
	}
}

// TestCleanupSkipsVisibleAndLegacy 可见评价与 legacy 导入评价永不进入清理。
func TestCleanupSkipsVisibleAndLegacy(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "可见正文", IsAnonymous: false}); err != nil {
		t.Fatalf("create review: %v", err)
	}
	conn := dbconnect.Connect()
	legacy := course.ReviewEntity{
		OfferingId:   offeringId,
		AuthorUserId: uint64Ptr(0),
		Content:      "历史评价",
		IsAnonymous:  true,
		Status:       course.ReviewStatusVisible,
		Source:       course.ReviewSourceLegacyImport,
	}
	if err := conn.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy review: %v", err)
	}
	cleaned, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 0 {
		t.Fatalf("cleaned = %d, want 0（可见/legacy 不动）", cleaned)
	}
}

// TestCleanupFallbackToUpdatedAt 存量 deleted 行（本功能上线前删除，
// deleted_at 为空）以 updated_at 近似窗口起点，超期后同样被清理（COALESCE）。
func TestCleanupFallbackToUpdatedAt(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 3, Content: "存量删除正文", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	conn := dbconnect.Connect()
	// 抹掉 deleted_at，只留超期的 updated_at（模拟上线前的存量 deleted 行）。
	old := time.Now().Add(-(time.Duration(ReviewCleanupRetentionDays)*24*time.Hour + 24*time.Hour))
	if err := conn.Table((&course.ReviewEntity{}).TableName()).
		Where("id = ?", payload.Id).
		Updates(map[string]any{"deleted_at": nil, "updated_at": old}).Error; err != nil {
		t.Fatalf("clear deleted_at: %v", err)
	}
	cleaned, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("cleaned = %d, want 1（COALESCE 回退 updated_at）", cleaned)
	}
	ent := loadReviewRow(t, payload.Id)
	if ent.Content != "" || ent.AuthorID() != 0 {
		t.Fatalf("legacy deleted row not cleaned: %+v", ent)
	}
}

// TestCleanupAllowsRecreateAfterCleanup 验收 1 的"占位释放"：清理后同一用户
// 可对同一 offering 重新写评（新建行，不与唯一索引冲突），stats 正确累加。
func TestCleanupAllowsRecreateAfterCleanup(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	first, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "第一次", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create first review: %v", err)
	}
	if err := DeleteReview(1001, first.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	backdateReviewDelete(t, first.Id, ReviewCleanupRetentionDays*24*time.Hour+24*time.Hour)
	if _, err := CleanupExpiredReviewsBatch(10); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	second, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "清理后重写", IsAnonymous: true})
	if err != nil {
		t.Fatalf("recreate after cleanup must succeed（占位已释放）: %v", err)
	}
	if second.Id == first.Id {
		t.Fatalf("recreated review reused cleaned row id %d, want a new row", second.Id)
	}
	courseStats, err := course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 1 || courseStats.RatingSum != 5 || courseStats.ReviewCount != 1 {
		t.Fatalf("stats after recreate = %+v, want 1/5/1", courseStats)
	}
	// 旧的已清理行仍保留（审计），且不与新行冲突。
	old := loadReviewRow(t, first.Id)
	if old.Status != course.ReviewStatusDeleted || old.Content != "" || old.AuthorID() != 0 {
		t.Fatalf("cleaned row altered by recreate: %+v", old)
	}
}

// TestDeleteReviewIdempotentAfterCleanup 清理后（作者已断开）作者再次删除仍幂等成功。
func TestDeleteReviewIdempotentAfterCleanup(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "正文", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	backdateReviewDelete(t, payload.Id, ReviewCleanupRetentionDays*24*time.Hour+24*time.Hour)
	if _, err := CleanupExpiredReviewsBatch(10); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete after cleanup must be idempotent: %v", err)
	}
}

// TestCleanupSameOfferingMultipleCleanedRows 吸收 #202 核心语义：同一 offering
// 多条 deleted 行清理后 author 均置 NULL——NULL 在唯一索引中彼此不冲突
// （SQLite/PG 一致），多条已清理行可共存（旧实现置 0 会撞唯一索引）。
func TestCleanupSameOfferingMultipleCleanedRows(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	create := func(authorID uint64) uint64 {
		t.Helper()
		p, err := CreateReview(authorID, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "将被清理", IsAnonymous: false})
		if err != nil {
			t.Fatalf("create review: %v", err)
		}
		if err := DeleteReview(authorID, p.Id); err != nil {
			t.Fatalf("delete review: %v", err)
		}
		backdateReviewDelete(t, p.Id, ReviewCleanupRetentionDays*24*time.Hour+24*time.Hour)
		return p.Id
	}
	id1 := create(1001)
	id2 := create(1002)

	cleaned, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 2 {
		t.Fatalf("cleaned = %d, want 2（多条同 offering 均清理，NULL 不冲突）", cleaned)
	}
	for _, id := range []uint64{id1, id2} {
		ent := loadReviewRow(t, id)
		if ent.Content != "" || ent.AuthorUserId != nil {
			t.Fatalf("row %d not fully cleaned: content=%q author=%v", id, ent.Content, ent.AuthorUserId)
		}
	}
}

// TestEnqueueCleanupTaskDedupe 入队去重：已有 pending/retrying 清理任务时
// 不再重复入队。
func TestEnqueueCleanupTaskDedupe(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task queue: %v", err)
	}
	if err := conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{}).Error; err != nil {
		t.Fatalf("clean task queue: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := EnqueueCleanupTask(); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", TaskTypeCourseReviewCleanup+"%").Count(&count).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("pending cleanup tasks = %d, want 1（去重）", count)
	}
}

// TestRunCleanupTaskWorker worker 处理：消费 pending 任务并清理超窗行。
func TestRunCleanupTaskWorker(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task queue: %v", err)
	}
	if err := conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{}).Error; err != nil {
		t.Fatalf("clean task queue: %v", err)
	}
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "worker 清理正文", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	backdateReviewDelete(t, payload.Id, ReviewCleanupRetentionDays*24*time.Hour+24*time.Hour)

	task := &taskQueue.Entity{Type: TaskTypeCourseReviewCleanup + ".run", TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := RunCleanupTask(context.Background(), task); err != nil {
		t.Fatalf("run cleanup task: %v", err)
	}
	ent := loadReviewRow(t, payload.Id)
	if ent.Content != "" || ent.AuthorID() != 0 {
		t.Fatalf("worker did not clean review: %+v", ent)
	}
}

// TestCleanupWindowAnchorIsDeleteTime 锁定窗口锚点判别性（spec N1 延续）：
// DeleteReview 经 MarkReviewDeletedFromTx 显式写 deleted_at=now。真实路径：
// 创建 100 天前的评价（created_at/updated_at 均回拨）→ 今天真实 DeleteReview
// → 30 天内不清；且断言 deleted_at 已写入（专用锚点，区别于 updated_at 语义）。
// 判别性：回拨用裸 Table().Update（无 schema 无 autoUpdateTime）；
// 若 MarkReviewDeletedFromTx 不写 deleted_at（buggy），deleted_at 为空 →
// COALESCE 回退 updated_at（100 天前）→ 立即被清 → 测试红。
func TestCleanupWindowAnchorIsDeleteTime(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	conn := dbconnect.Connect()

	payload, err := CreateReview(8001, CreateReviewInput{
		OfferingId: offeringId,
		Rating:     5,
		Content:    "百天老评价",
	})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	// 裸 Table().Update 把 created_at/updated_at 回拨 100 天前（模拟老评价）
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	if err := conn.Table((&course.ReviewEntity{}).TableName()).Where("id = ?", payload.Id).
		Updates(map[string]any{"created_at": oldTime, "updated_at": oldTime}).Error; err != nil {
		t.Fatalf("age review timestamps: %v", err)
	}
	// 确认回拨真实生效（防 schema 拦截静默 no-op）
	var preDelete course.ReviewEntity
	if err := conn.First(&preDelete, payload.Id).Error; err != nil {
		t.Fatalf("reload aged review: %v", err)
	}
	if time.Since(preDelete.UpdatedAt) < 90*24*time.Hour {
		t.Fatalf("timestamp rollback was a no-op (schema intercepted): updated_at=%v", preDelete.UpdatedAt)
	}

	// 今天经真实 DeleteReview 删除（MarkReviewDeletedFromTx 写 deleted_at）
	if err := DeleteReview(8001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	var afterDelete course.ReviewEntity
	if err := conn.First(&afterDelete, payload.Id).Error; err != nil {
		t.Fatalf("reload deleted review: %v", err)
	}
	// 关键断言：deleted_at 必须已写入（今天），不是 NULL
	if afterDelete.DeletedAt == nil {
		t.Fatal("DeleteReview did not write deleted_at anchor (window collapsed to updated_at)")
	}
	if since := time.Since(*afterDelete.DeletedAt); since > 24*time.Hour {
		t.Fatalf("deleted_at anchor not refreshed: %v (since %v)", *afterDelete.DeletedAt, since)
	}

	// 清理运行：30 天窗口内 → 不清
	cleaned, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 0 {
		t.Fatalf("cleaned = %d, want 0 (deleted today, window not expired)", cleaned)
	}
	var row course.ReviewEntity
	if err := conn.First(&row, payload.Id).Error; err != nil {
		t.Fatalf("row missing: %v", err)
	}
	if row.Content != "百天老评价" || row.AuthorID() != 8001 {
		t.Fatalf("in-window review touched: content=%q author=%d", row.Content, row.AuthorID())
	}
}
