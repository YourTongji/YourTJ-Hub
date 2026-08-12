package courseservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// createDeletedReview 造一条 status=deleted 的评价行。删除窗口起点是
// updated_at（status 变 deleted 时 autoUpdateTime 写入；deleted 行删除后
// 无其他写路径），故创建后把 updated_at 覆写为给定时间模拟删除时间。
// 注意：不设置 gorm.DeletedAt（置值会被查询自动软删过滤）。
func createDeletedReview(t *testing.T, offeringId, authorId uint64, deletedAt time.Time) uint64 {
	t.Helper()
	conn := dbconnect.Connect()
	rating := 4
	entity := &course.ReviewEntity{
		OfferingId:   offeringId,
		AuthorUserId: authorId,
		Rating:       &rating,
		Content:      "隐私正文内容",
		IsAnonymous:  false,
		Status:       course.ReviewStatusDeleted,
	}
	if err := conn.Create(entity).Error; err != nil {
		t.Fatalf("create deleted review: %v", err)
	}
	if err := conn.Model(entity).Update("updated_at", deletedAt).Error; err != nil {
		t.Fatalf("set review updated_at: %v", err)
	}
	return entity.Id
}

// TestCleanupDeletedReviewsExpiredWindow 验收 1：删除超 30 天 → content 为空、
// 作者关联断开（author_user_id=0，释放唯一约束占位）、行保留（可审计）。
func TestCleanupDeletedReviewsExpiredWindow(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	conn := dbconnect.Connect()

	// 两条超窗 deleted 行（不同 offering，避免唯一索引 (offering, author=0)
	// 每 offering 至多一条的限制）：31 天前 + 40 天前
	offering2 := createExtraOffering(t)
	oldID := createDeletedReview(t, offeringId, 1001, time.Now().Add(-31*24*time.Hour))
	oldID2 := createDeletedReview(t, offering2, 1002, time.Now().Add(-40*24*time.Hour))

	cleaned, err := CleanupDeletedReviews(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 2 {
		t.Fatalf("cleaned = %d, want 2", cleaned)
	}

	var old1, old2 course.ReviewEntity
	if err := conn.First(&old1, oldID).Error; err != nil {
		t.Fatalf("row %d should be kept (audit): %v", oldID, err)
	}
	if old1.Content != "" {
		t.Fatalf("expired review content = %q, want empty", old1.Content)
	}
	if old1.AuthorUserId != 0 {
		t.Fatalf("expired review author = %d, want 0 (disconnected)", old1.AuthorUserId)
	}
	if old1.Status != course.ReviewStatusDeleted {
		t.Fatalf("expired review status = %d, want deleted (kept for audit)", old1.Status)
	}
	if err := conn.First(&old2, oldID2).Error; err != nil {
		t.Fatalf("row %d should be kept (audit): %v", oldID2, err)
	}
	if old2.Content != "" || old2.AuthorUserId != 0 {
		t.Fatalf("second expired review not anonymized: content=%q author=%d", old2.Content, old2.AuthorUserId)
	}
}

// createExtraOffering 追加一个可见 offering（用于多行清理测试）。
func createExtraOffering(t *testing.T) uint64 {
	t.Helper()
	conn := dbconnect.Connect()
	term := course.TermEntity{Code: "2026-2027-1", Name: "2026-2027 第一学期", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	c := course.Entity{PrimaryCode: "100002", Name: "线性代数", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Campus: "四平路校区", Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	return offering.Id
}

// TestCleanupDeletedReviewsWithinWindow 验收 2：删除未到 30 天 → 不动该行
// （content/作者关联原样保留，允许窗口内恢复重写）。
func TestCleanupDeletedReviewsWithinWindow(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	conn := dbconnect.Connect()

	recentID := createDeletedReview(t, offeringId, 2001, time.Now().Add(-10*24*time.Hour))
	noDeletedAtID := func() uint64 {
		rating := 5
		entity := &course.ReviewEntity{
			OfferingId:   offeringId,
			AuthorUserId: 2002,
			Rating:       &rating,
			Content:      "未写入删除时间戳的旧行",
			Status:       course.ReviewStatusDeleted,
		}
		if err := conn.Create(entity).Error; err != nil {
			t.Fatalf("create legacy deleted review: %v", err)
		}
		return entity.Id
	}()

	cleaned, err := CleanupDeletedReviews(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 0 {
		t.Fatalf("cleaned = %d, want 0 (window not expired)", cleaned)
	}

	var recent, legacy course.ReviewEntity
	if err := conn.First(&recent, recentID).Error; err != nil {
		t.Fatalf("recent row missing: %v", err)
	}
	if recent.Content != "隐私正文内容" || recent.AuthorUserId != 2001 {
		t.Fatalf("recent review touched: content=%q author=%d", recent.Content, recent.AuthorUserId)
	}
	if err := conn.First(&legacy, noDeletedAtID).Error; err != nil {
		t.Fatalf("legacy row missing: %v", err)
	}
	if legacy.Content != "未写入删除时间戳的旧行" || legacy.AuthorUserId != 2002 {
		t.Fatalf("legacy review touched: content=%q author=%d", legacy.Content, legacy.AuthorUserId)
	}
}

// TestCleanupDeletedReviewsPlaceholderReleased 验收 1 的占位释放语义：
// 清理后同 offering 同用户可重新写评（不再命中 deleted 行复用路径）。
func TestCleanupDeletedReviewsPlaceholderReleased(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	oldID := createDeletedReview(t, offeringId, 3001, time.Now().Add(-35*24*time.Hour))

	if _, err := CleanupDeletedReviews(30 * 24 * time.Hour); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	// 清理后：FindReviewByOfferingAndUser 不应再命中 author_user_id=3001 的行
	existing, err := course.FindReviewByOfferingAndUserTx(dbconnect.Connect(), offeringId, 3001)
	if err == nil && existing.Id > 0 && existing.Id == oldID {
		t.Fatalf("placeholder still occupies unique key: review %d still matched", oldID)
	}
	// 新写评应走新建路径（不再复用 deleted 行），而不是 ErrReviewDuplicate
	payload, err := CreateReview(3001, CreateReviewInput{
		OfferingId: offeringId,
		Rating:     5,
		Content:    "窗口后重新写评",
	})
	if err != nil {
		t.Fatalf("re-create review after cleanup: %v", err)
	}
	if payload.Id == 0 || payload.Id == oldID {
		t.Fatalf("re-created review id = %d, want a new row (old=%d)", payload.Id, oldID)
	}
}

// TestCleanupTaskEnqueueAndWorker 验收 3（taskQueue 语义）：EnqueueCleanupTask
// 入队 pending 任务且幂等（重复入队不产生第二条）；RunCleanupTask 执行清理。
// 任务状态流转（Success/retrying/failed）由 backgroundservice.RunWorker 的
// processTask 负责（dev 当前朴素领取版），此处验证 worker handler 本身。
func TestCleanupTaskEnqueueAndWorker(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	conn := dbconnect.Connect()
	oldID := createDeletedReview(t, offeringId, 4001, time.Now().Add(-35*24*time.Hour))

	if err := EnqueueCleanupTask(); err != nil {
		t.Fatalf("enqueue cleanup task: %v", err)
	}
	// 幂等：重复入队不产生第二条 pending 任务
	if err := EnqueueCleanupTask(); err != nil {
		t.Fatalf("re-enqueue cleanup task: %v", err)
	}
	tasks := taskQueue.GetPendingTasksByType(TaskTypeCourseReviewCleanup, 10)
	if len(tasks) != 1 {
		t.Fatalf("pending cleanup tasks = %d, want 1", len(tasks))
	}

	if err := RunCleanupTask(nil, tasks[0]); err != nil {
		t.Fatalf("run cleanup task: %v", err)
	}
	// worker handler 效果：超窗行已被脱敏
	var old course.ReviewEntity
	if err := conn.First(&old, oldID).Error; err != nil {
		t.Fatalf("cleaned row missing: %v", err)
	}
	if old.Content != "" || old.AuthorUserId != 0 {
		t.Fatalf("cleanup task did not anonymize: content=%q author=%d", old.Content, old.AuthorUserId)
	}
}

// TestCleanupDeletedReviewsSameOfferingMultiRows 同 offering 多条 deleted 行：
// content 全部清空；author 逐行置 0，撞唯一索引的行保留原作者（可复用路径
// 仍可用），至少一条 author 置 0（占位释放）。
func TestCleanupDeletedReviewsSameOfferingMultiRows(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	conn := dbconnect.Connect()

	oldID1 := createDeletedReview(t, offeringId, 5001, time.Now().Add(-35*24*time.Hour))
	oldID2 := createDeletedReview(t, offeringId, 5002, time.Now().Add(-36*24*time.Hour))

	cleaned, err := CleanupDeletedReviews(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleaned != 2 {
		t.Fatalf("cleaned = %d, want 2 (both contents cleared)", cleaned)
	}

	var r1, r2 course.ReviewEntity
	if err := conn.First(&r1, oldID1).Error; err != nil {
		t.Fatalf("row1 missing: %v", err)
	}
	if err := conn.First(&r2, oldID2).Error; err != nil {
		t.Fatalf("row2 missing: %v", err)
	}
	if r1.Content != "" || r2.Content != "" {
		t.Fatalf("content not fully cleared: r1=%q r2=%q", r1.Content, r2.Content)
	}
	// 至少一条作者断开（每 offering 至多一条 author=0，其余保留原值）
	if r1.AuthorUserId != 0 && r2.AuthorUserId != 0 {
		t.Fatalf("no author disconnected: r1=%d r2=%d", r1.AuthorUserId, r2.AuthorUserId)
	}
	// 断开的行（author=0）用户可重新写评（新建不撞索引）
	var zeroRow course.ReviewEntity
	if r1.AuthorUserId == 0 {
		zeroRow = r1
	} else {
		zeroRow = r2
	}
	payload, err := CreateReview(zeroRow.AuthorUserId+1000, CreateReviewInput{
		OfferingId: offeringId,
		Rating:     5,
		Content:    "窗口后重新写评",
	})
	_ = payload
	if err != nil {
		t.Fatalf("re-create after cleanup: %v", err)
	}
}
