package course

import (
	"os"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestUpsertCourseStatsConcurrentPostgreSQL 在真实 PostgreSQL 上验证统计投影并发原子累加不丢更新。
// read-modify-write 版本会在并发事务中读到同一行后各自写回、互相覆盖；改为
// INSERT ... ON CONFLICT DO UPDATE + delta 后，N 次并发 upsert 的合计必须精确等于 N。
// 依赖 YOURTJ_TEST_PG_URL（与 migration 包真实 PG 迁移测试同一门控），未设置时跳过。
func TestUpsertCourseStatsConcurrentPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL concurrent stats test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.AutoMigrate(&CourseStatsEntity{}); err != nil {
		t.Fatalf("AutoMigrate course stats on postgres failed: %v", err)
	}
	if err := db.Unscoped().Where("1 = 1").Delete(&CourseStatsEntity{}).Error; err != nil {
		t.Fatalf("clean course stats on postgres: %v", err)
	}

	const courseId uint64 = 900001
	const workers = 8
	const perWorker = 25
	const total = workers * perWorker

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				if err := UpsertCourseStatsTx(db, courseId, 1, 5, 1); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		t.Fatalf("concurrent upsert failed: %v", firstErr)
	}

	var stats CourseStatsEntity
	if err := db.Where("course_id = ?", courseId).First(&stats).Error; err != nil {
		t.Fatalf("load stats from postgres: %v", err)
	}
	if stats.RatingCount != total || stats.RatingSum != 5*total || stats.ReviewCount != total {
		t.Fatalf("concurrent upsert lost updates: got RatingCount=%d RatingSum=%d ReviewCount=%d, want %d/%d/%d",
			stats.RatingCount, stats.RatingSum, stats.ReviewCount, total, 5*total, total)
	}
}
