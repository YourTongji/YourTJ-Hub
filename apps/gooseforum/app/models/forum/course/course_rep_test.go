package course

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

// setupCourseRepTest 迁移 course 表并清空，供 repository 层测试使用。
func setupCourseRepTest(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate course table: %v", err)
	}
	if err := conn.Unscoped().Where("1 = 1").Delete(&Entity{}).Error; err != nil {
		t.Fatalf("clean course table: %v", err)
	}
}

// TestListCoursesByPrimaryCodesExcludesHidden 回归 PR #197 P13：
// ListCoursesByPrimaryCodes 是 P13 课评摘要匹配的底层查询，必须与 ListCourses 一致
// 只返回 StatusVisible 的课程，隐藏课程即使主课号命中也不得返回。
func TestListCoursesByPrimaryCodesExcludesHidden(t *testing.T) {
	setupCourseRepTest(t)
	conn := dbconnect.Connect()

	visible := Entity{PrimaryCode: "100001", Name: "高等数学(A)上", Department: "数学科学学院", Status: StatusVisible}
	if err := conn.Create(&visible).Error; err != nil {
		t.Fatalf("create visible course: %v", err)
	}
	hidden := Entity{PrimaryCode: "100002", Name: "被隐藏的课程", Department: "某学院", Status: StatusHidden}
	if err := conn.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden course: %v", err)
	}

	got, err := ListCoursesByPrimaryCodes([]string{"100001", "100002", "999999"})
	if err != nil {
		t.Fatalf("ListCoursesByPrimaryCodes: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("result size = %d, want 1 (only visible course); got %+v", len(got), got)
	}
	if got[0].PrimaryCode != "100001" {
		t.Fatalf("primary_code = %q, want 100001", got[0].PrimaryCode)
	}
}
