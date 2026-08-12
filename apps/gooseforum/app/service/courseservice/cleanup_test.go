package courseservice

import (
	"context"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// backdateReviewDelete 把 deleted_at/updated_at 回拨到隔离窗口之外，模拟"删除已超 30 天"。
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

// loadReviewRow 用 Table 查询读取行（不受软删过滤影响；课评领域不启用 GORM 软删）。
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

// TestCleanupExpiredReviewClearsContentAndAuthor 验收 1：删除超 30 天的课评，
// 清理后 content 为空、作者关联断开、行保留（status 仍为 deleted）、占位释放。
func TestCleanupExpiredReviewClearsContentAndAuthor(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "将被清理的正文", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	backdateReviewDelete(t, payload.Id, ReviewCleanupWindow+24*time.Hour)

	cleaned, err := CleanupExpiredReviewsBatch(10)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("cleaned = %d, want 1", cleaned)
	}
	ent := loadReviewRow(t, payload.Id)
	if ent.Status != course.ReviewStatusDeleted {
		t.Fatalf("status after cleanup = %d, want deleted(2)（行保留可审计）", ent.Status)
	}
	if ent.Content != "" {
		t.Fatalf("content after cleanup = %q, want empty", ent.Content)
	}
	if ent.AuthorID() != 0 {
		t.Fatalf("author after cleanup = %d, want 0（作者关联断开）", ent.AuthorID())
	}
	// 统计投影不受清理影响（删除时已扣减，清理不改 status）。
	courseStats, err := course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 0 || courseStats.RatingSum != 0 || courseStats.ReviewCount != 0 {
		t.Fatalf("stats corrupted by cleanup: %+v", courseStats)
	}
	// 幂等：再次清理不再选中该行。
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

// TestCleanupFallbackToUpdatedAt 存量 deleted 行（本功能上线前删除，deleted_at 为空）
// 以 updated_at 近似窗口起点，超期后同样被清理。
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
	old := time.Now().Add(-(ReviewCleanupWindow + 24*time.Hour))
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
// 可对同一 offering 重新写评（新建行，不与唯一索引冲突）。
func TestCleanupAllowsRecreateAfterCleanup(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	first, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "第一次", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create first review: %v", err)
	}
	if err := DeleteReview(1001, first.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	backdateReviewDelete(t, first.Id, ReviewCleanupWindow+24*time.Hour)
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
	backdateReviewDelete(t, payload.Id, ReviewCleanupWindow+24*time.Hour)
	if _, err := CleanupExpiredReviewsBatch(10); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete after cleanup must be idempotent: %v", err)
	}
}

// TestEnqueueCourseReviewCleanupTaskDedupe 入队去重：已有 pending/retrying
// 清理任务时不再重复入队。
func TestEnqueueCourseReviewCleanupTaskDedupe(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task queue: %v", err)
	}
	if err := conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{}).Error; err != nil {
		t.Fatalf("clean task queue: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := EnqueueCourseReviewCleanupTask(); err != nil {
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

// TestRunCourseReviewCleanupTaskWorker worker 处理：消费 pending 任务并清理，
// 任务标记成功后不再有残留。
func TestRunCourseReviewCleanupTaskWorker(t *testing.T) {
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
	backdateReviewDelete(t, payload.Id, ReviewCleanupWindow+24*time.Hour)

	task := &taskQueue.Entity{Type: TaskTypeCourseReviewCleanup + "sweep", TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := RunCourseReviewCleanupTask(context.Background(), task); err != nil {
		t.Fatalf("run cleanup task: %v", err)
	}
	ent := loadReviewRow(t, payload.Id)
	if ent.Content != "" || ent.AuthorID() != 0 {
		t.Fatalf("worker did not clean review: %+v", ent)
	}
	reloaded, err := taskQueue.GetByID(task.Id)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if reloaded.Status != taskQueue.StatusPending && reloaded.Status != taskQueue.StatusRunning {
		t.Fatalf("task status = %d, want pending/running（框架负责终态标记）", reloaded.Status)
	}
}
