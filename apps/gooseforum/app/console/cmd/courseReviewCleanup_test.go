package cmd

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// TestRunCourseReviewCleanupRespectsWindow 锁定 spec F2-CLI：CLI 手动触发
// 必须按 ReviewCleanupRetentionDays*24h 计算窗口——此前误写 *24（720ns）
// 会把窗口内（刚删除）的行一并清掉，直接违反验收 2。
func TestRunCourseReviewCleanupRespectsWindow(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&course.Entity{}, &course.TermEntity{}, &course.OfferingEntity{}, &course.ReviewEntity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, model := range []any{&course.ReviewEntity{}, &course.OfferingEntity{}, &course.TermEntity{}, &course.Entity{}} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean tables: %v", err)
		}
	}
	c := course.Entity{PrimaryCode: "100003", Name: "CLI 窗口测试课", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2026-2027-1", Name: "2026-2027 第一学期", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}

	// 窗口内行：今天删除（updated_at = now），内容/作者保留
	rating := 4
	author9001 := uint64(9001)
	author9002 := uint64(9002)
	recent := &course.ReviewEntity{
		OfferingId:   offering.Id,
		AuthorUserId: &author9001,
		Rating:       &rating,
		Content:      "刚删除的评价正文",
		Status:       course.ReviewStatusDeleted,
	}
	if err := conn.Create(recent).Error; err != nil {
		t.Fatalf("create recent deleted review: %v", err)
	}
	// 超窗行：40 天前删除
	old := &course.ReviewEntity{
		OfferingId:   offering.Id,
		AuthorUserId: &author9002,
		Rating:       &rating,
		Content:      "超窗的评价正文",
		Status:       course.ReviewStatusDeleted,
	}
	if err := conn.Create(old).Error; err != nil {
		t.Fatalf("create old deleted review: %v", err)
	}
	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	if err := conn.Model(old).Update("updated_at", oldTime).Error; err != nil {
		t.Fatalf("age old review: %v", err)
	}

	if err := runCourseReviewCleanup(nil, nil); err != nil {
		t.Fatalf("runCourseReviewCleanup: %v", err)
	}

	var recentRow, oldRow course.ReviewEntity
	if err := conn.First(&recentRow, recent.Id).Error; err != nil {
		t.Fatalf("recent row missing: %v", err)
	}
	if err := conn.First(&oldRow, old.Id).Error; err != nil {
		t.Fatalf("old row missing: %v", err)
	}
	// 窗口内行必须原样保留（验收 2）
	if recentRow.Content != "刚删除的评价正文" || recentRow.AuthorID() != 9001 {
		t.Fatalf("CLI cleanup touched in-window review: content=%q author=%d", recentRow.Content, recentRow.AuthorID())
	}
	// 超窗行被清理
	if oldRow.Content != "" || oldRow.AuthorID() != 0 {
		t.Fatalf("CLI cleanup did not anonymize expired review: content=%q author=%d", oldRow.Content, oldRow.AuthorID())
	}
}
