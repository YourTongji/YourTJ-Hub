package courseservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// setupStatsReadTest 迁移 course 域表并清空，供课评摘要读取测试使用。
func setupStatsReadTest(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(reviewTestModels...); err != nil {
		t.Fatalf("migrate stats read tables: %v", err)
	}
	for _, model := range reviewTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean stats read table: %v", err)
		}
	}
}

// TestGetCourseStatsByPrimaryCodesExcludesHidden 回归 PR #197 P13：
// 公开未鉴权的 course-review-brief 不得泄漏 CourseManager 隐藏课程的名称与课评统计。
// GetCourseStatsByPrimaryCodes 只应返回 StatusVisible 的课程；隐藏课程即使有统计也不得出现在结果中。
func TestGetCourseStatsByPrimaryCodesExcludesHidden(t *testing.T) {
	setupStatsReadTest(t)
	conn := dbconnect.Connect()

	visible := course.Entity{PrimaryCode: "100001", Name: "高等数学(A)上", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&visible).Error; err != nil {
		t.Fatalf("create visible course: %v", err)
	}
	hidden := course.Entity{PrimaryCode: "100002", Name: "被隐藏的课程", Department: "某学院", Status: course.StatusHidden}
	if err := conn.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden course: %v", err)
	}
	// 可见与隐藏课程都写入课评统计，确保隐藏课程的摘要不会因"无统计"而碰巧漏掉。
	for _, id := range []uint64{visible.Id, hidden.Id} {
		if err := conn.Create(&course.CourseStatsEntity{CourseId: id, RatingCount: 1, RatingSum: 5, ReviewCount: 1}).Error; err != nil {
			t.Fatalf("create stats for course %d: %v", id, err)
		}
	}

	got, err := GetCourseStatsByPrimaryCodes([]string{"100001", "100002", "999999"})
	if err != nil {
		t.Fatalf("GetCourseStatsByPrimaryCodes: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("result size = %d, want 1 (only visible course); got %+v", len(got), got)
	}
	brief, ok := got["100001"]
	if !ok {
		t.Fatalf("visible course 100001 missing from result: %+v", got)
	}
	if brief.Name != "高等数学(A)上" {
		t.Fatalf("visible course name = %q, want 高等数学(A)上", brief.Name)
	}
	if brief.ReviewCount != 1 {
		t.Fatalf("visible course reviewCount = %d, want 1", brief.ReviewCount)
	}
	if _, ok := got["100002"]; ok {
		t.Fatalf("hidden course 100002 must NOT appear in public brief: %+v", got)
	}
}
