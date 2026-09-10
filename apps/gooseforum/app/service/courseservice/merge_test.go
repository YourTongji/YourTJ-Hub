package courseservice

import (
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// setupMergeTest 迁移并清空合并服务相关表（manageTestModels + course_relations
// + course_user_action + course_ai_summary——合并会迁移收藏并失效 AI 总结缓存）。
func setupMergeTest(t *testing.T) {
	t.Helper()
	models := append([]any{
		&course.RelationEntity{},
		&course.CourseUserActionEntity{},
		&course.CourseAiSummaryEntity{},
	}, manageTestModels...)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate merge tables: %v", err)
	}
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean merge table: %v", err)
		}
	}
}

// seedMergePair 创建 from（旧卡）/to（新卡）两张课程卡 + offering + 评价 + 别名。
// 返回 (fromId, toId, offeringId)。
func seedMergePair(t *testing.T) (uint64, uint64, uint64) {
	t.Helper()
	conn := dbconnect.Connect()
	from := course.Entity{
		PrimaryCode:    "M101",
		Name:           "高等数学(A)上",
		NormalizedName: Normalize("高等数学(A)上"),
		Status:         course.StatusVisible,
		CreditX10:      50,
	}
	if err := conn.Create(&from).Error; err != nil {
		t.Fatalf("create from course: %v", err)
	}
	// to 卡使用新课程码（2026 改制后的新编码）：(code, teacher) 复合身份下与旧码不冲突。
	to := course.Entity{
		PrimaryCode:    "M102",
		Name:           "高等数学(A)I",
		NormalizedName: Normalize("高等数学(A)I"),
		Status:         course.StatusVisible,
		CreditX10:      50,
	}
	if err := conn.Create(&to).Error; err != nil {
		t.Fatalf("create to course: %v", err)
	}
	offering := course.OfferingEntity{
		CourseId: from.Id,
		TermId:   0,
		Status:   course.OfferingStatusVisible,
	}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	authorID := uint64(1)
	rating := 5
	review := course.ReviewEntity{
		OfferingId:   offering.Id,
		AuthorUserId: &authorID,
		Rating:       &rating,
		Content:      "很好",
		Status:       0,
	}
	if err := conn.Create(&review).Error; err != nil {
		t.Fatalf("create review: %v", err)
	}
	alias := course.AliasEntity{
		CourseId:        from.Id,
		Kind:            course.AliasKindName,
		Value:           "高数A上",
		NormalizedValue: Normalize("高数A上"),
		Source:          "import",
	}
	if err := conn.Create(&alias).Error; err != nil {
		t.Fatalf("create alias: %v", err)
	}
	return from.Id, to.Id, offering.Id
}

// TestMergeCoursesMovesOfferingsAndAliases 合并后评价/offering 随 offering 迁移零丢失、
// alias 迁移、from 卡隐藏、relations 置 merged、搜索/统计重建入队。
func TestMergeCoursesMovesOfferingsAndAliases(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, offeringId := seedMergePair(t)
	conn := dbconnect.Connect()
	relation := course.RelationEntity{
		FromCourseId: fromId,
		ToCourseId:   toId,
		RelationType: string(course.RelationEquivalent),
		Source:       course.RelationSourceRule,
		EvidenceJson: `{"teacherCodeOverlap":true,"courseCode":"M101"}`,
		Status:       string(course.RelationStatusPending),
	}
	if err := conn.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}

	result, err := MergeCourses(relation.Id)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if result.MovedOfferings != 1 {
		t.Errorf("movedOfferings = %d, want 1", result.MovedOfferings)
	}
	if result.MigratedAliases != 1 {
		t.Errorf("migratedAliases = %d, want 1", result.MigratedAliases)
	}
	// offering 迁移到 to 卡。
	var offering course.OfferingEntity
	if err := conn.First(&offering, offeringId).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	if offering.CourseId != toId {
		t.Errorf("offering course_id = %d, want %d", offering.CourseId, toId)
	}
	// 评价仍挂在 offering 下（零丢失）。
	var reviewCount int64
	if err := conn.Model(&course.ReviewEntity{}).Where("offering_id = ?", offeringId).Count(&reviewCount).Error; err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if reviewCount != 1 {
		t.Errorf("reviews = %d, want 1（评价零丢失）", reviewCount)
	}
	// alias 迁移到 to 卡。
	var alias course.AliasEntity
	if err := conn.Where("value = ?", "高数A上").First(&alias).Error; err != nil {
		t.Fatalf("find alias: %v", err)
	}
	if alias.CourseId != toId {
		t.Errorf("alias course_id = %d, want %d", alias.CourseId, toId)
	}
	// from 卡隐藏、to 卡保持可见。
	var from course.Entity
	if err := conn.First(&from, fromId).Error; err != nil {
		t.Fatalf("find from course: %v", err)
	}
	if from.Status != course.StatusHidden {
		t.Errorf("from status = %d, want hidden", from.Status)
	}
	var to course.Entity
	if err := conn.First(&to, toId).Error; err != nil {
		t.Fatalf("find to course: %v", err)
	}
	if to.Status != course.StatusVisible {
		t.Errorf("to status = %d, want visible", to.Status)
	}
	// relations 置 merged + manual。
	var rel course.RelationEntity
	if err := conn.First(&rel, relation.Id).Error; err != nil {
		t.Fatalf("find relation: %v", err)
	}
	if rel.Status != string(course.RelationStatusMerged) {
		t.Errorf("relation status = %q, want merged", rel.Status)
	}
	if !rel.Manual {
		t.Errorf("relation manual = false, want true")
	}
	if rel.EvidenceJson == "" || rel.EvidenceJson == `{"teacherCodeOverlap":true,"courseCode":"M101"}` {
		t.Errorf("relation evidence_json = %q, want merge snapshot", rel.EvidenceJson)
	}
	// 搜索任务入队（from + to 各一条）。
	if got := countTasksByType(t, "course-search."); got != 2 {
		t.Errorf("course-search tasks = %d, want 2", got)
	}
	// 统计重建任务入队（去重 1 条）。
	if got := countTasksByType(t, "course-stats."); got != 1 {
		t.Errorf("course-stats tasks = %d, want 1", got)
	}
}

// TestMergeCoursesAliasConflictSkipped alias 冲突时跳过并记录，不中断合并。
func TestMergeCoursesAliasConflictSkipped(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	conn := dbconnect.Connect()
	// 唯一索引 (kind, normalized_value) 下正常数据不可能出现跨课程同别名；真实冲突只
	// 可能来自索引建立前的 legacy 重复数据。本用例先删索引再制造冲突，验证防御性跳过。
	if err := conn.Migrator().DropIndex(&course.AliasEntity{}, "uniq_course_alias_kind_value"); err != nil {
		t.Fatalf("drop alias unique index: %v", err)
	}
	t.Cleanup(func() {
		// 清理重复行并恢复唯一索引，避免污染后续测试的 AutoMigrate。
		_ = conn.Unscoped().Where("kind = ? AND normalized_value = ?", course.AliasKindName, Normalize("高数A上")).
			Delete(&course.AliasEntity{}).Error
		_ = conn.Migrator().CreateIndex(&course.AliasEntity{}, "uniq_course_alias_kind_value")
	})
	// to 卡已占用同 normalized 别名（kind=name）。
	conflict := course.AliasEntity{
		CourseId:        toId,
		Kind:            course.AliasKindName,
		Value:           "高数A上",
		NormalizedValue: Normalize("高数A上"),
		Source:          "admin",
	}
	if err := conn.Create(&conflict).Error; err != nil {
		t.Fatalf("create conflicting alias: %v", err)
	}
	relation := course.RelationEntity{
		FromCourseId: fromId,
		ToCourseId:   toId,
		RelationType: string(course.RelationEquivalent),
		Source:       course.RelationSourceRule,
		Status:       string(course.RelationStatusPending),
	}
	if err := conn.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}

	if _, err := MergeCourses(relation.Id); err != nil {
		t.Fatalf("merge: %v", err)
	}
	// 冲突别名仍属 to 卡（未被覆盖）；from 卡的重复别名被跳过（未迁移）。
	var toAlias course.AliasEntity
	if err := conn.Where("value = ? AND course_id = ?", "高数A上", toId).First(&toAlias).Error; err != nil {
		t.Fatalf("find to-card alias: %v", err)
	}
	var fromAlias course.AliasEntity
	if err := conn.Where("value = ? AND course_id = ?", "高数A上", fromId).First(&fromAlias).Error; err != nil {
		t.Fatalf("find from-card alias: %v", err)
	}
}

// TestMergeCoursesConflictGuard from 卡存在其他 pending 合并候选时拒绝合并。
func TestMergeCoursesConflictGuard(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	conn := dbconnect.Connect()
	// 另一张 to2 卡也有 pending 等价候选（同 from）。
	to2 := course.Entity{
		PrimaryCode:    "M103",
		Name:           "高等数学(A)I",
		NormalizedName: Normalize("高等数学(A)I"),
		Status:         course.StatusVisible,
	}
	if err := conn.Create(&to2).Error; err != nil {
		t.Fatalf("create to2 course: %v", err)
	}
	rel1 := course.RelationEntity{FromCourseId: fromId, ToCourseId: toId, RelationType: string(course.RelationEquivalent), Source: course.RelationSourceRule, Status: string(course.RelationStatusPending)}
	if err := conn.Create(&rel1).Error; err != nil {
		t.Fatalf("create rel1: %v", err)
	}
	rel2 := course.RelationEntity{FromCourseId: fromId, ToCourseId: to2.Id, RelationType: string(course.RelationEquivalent), Source: course.RelationSourceRule, Status: string(course.RelationStatusPending)}
	if err := conn.Create(&rel2).Error; err != nil {
		t.Fatalf("create rel2: %v", err)
	}

	if _, err := MergeCourses(rel1.Id); !errors.Is(err, ErrMergeConflict) {
		t.Fatalf("merge = %v, want ErrMergeConflict", err)
	}
}

// TestMergeCoursesRejectsNonMergeable SPLIT_FROM/RELATED 等不允许合并。
func TestMergeCoursesRejectsNonMergeable(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	conn := dbconnect.Connect()
	rel := course.RelationEntity{FromCourseId: fromId, ToCourseId: toId, RelationType: string(course.RelationSplit), Source: course.RelationSourceRule, Status: string(course.RelationStatusPending)}
	if err := conn.Create(&rel).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}
	if _, err := MergeCourses(rel.Id); !errors.Is(err, ErrRelationNotMergeable) {
		t.Fatalf("merge = %v, want ErrRelationNotMergeable", err)
	}
	// from 卡未被隐藏。
	var from course.Entity
	if err := conn.First(&from, fromId).Error; err != nil {
		t.Fatalf("find from course: %v", err)
	}
	if from.Status != course.StatusVisible {
		t.Errorf("from status = %d, want visible（非合并类型不得动课程）", from.Status)
	}
}

// TestUndoMergeCourse 撤销合并：offering/alias 迁回 from 卡、from 卡恢复可见、relations 回 approved。
func TestUndoMergeCourse(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, offeringId := seedMergePair(t)
	conn := dbconnect.Connect()
	relation := course.RelationEntity{
		FromCourseId: fromId,
		ToCourseId:   toId,
		RelationType: string(course.RelationEquivalent),
		Source:       course.RelationSourceRule,
		Status:       string(course.RelationStatusPending),
	}
	if err := conn.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}
	if _, err := MergeCourses(relation.Id); err != nil {
		t.Fatalf("merge: %v", err)
	}

	result, err := UndoMergeCourse(relation.Id)
	if err != nil {
		t.Fatalf("undo: %v", err)
	}
	if result.MovedOfferings != 1 {
		t.Errorf("undo movedOfferings = %d, want 1", result.MovedOfferings)
	}
	// offering 迁回 from 卡。
	var offering course.OfferingEntity
	if err := conn.First(&offering, offeringId).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	if offering.CourseId != fromId {
		t.Errorf("offering course_id = %d, want %d（迁回）", offering.CourseId, fromId)
	}
	// alias 迁回 from 卡。
	var alias course.AliasEntity
	if err := conn.Where("value = ?", "高数A上").First(&alias).Error; err != nil {
		t.Fatalf("find alias: %v", err)
	}
	if alias.CourseId != fromId {
		t.Errorf("alias course_id = %d, want %d（迁回）", alias.CourseId, fromId)
	}
	// from 卡恢复可见。
	var from course.Entity
	if err := conn.First(&from, fromId).Error; err != nil {
		t.Fatalf("find from course: %v", err)
	}
	if from.Status != course.StatusVisible {
		t.Errorf("from status = %d, want visible（恢复）", from.Status)
	}
	// relations 回 approved + evidence 还原。
	var rel course.RelationEntity
	if err := conn.First(&rel, relation.Id).Error; err != nil {
		t.Fatalf("find relation: %v", err)
	}
	if rel.Status != string(course.RelationStatusApproved) {
		t.Errorf("relation status = %q, want approved", rel.Status)
	}
	if !rel.Manual {
		t.Errorf("relation manual = false, want true（人工确认记录保留）")
	}
	// 再次撤销被拒绝。
	if _, err := UndoMergeCourse(relation.Id); !errors.Is(err, ErrRelationNotMergeable) {
		t.Fatalf("second undo = %v, want ErrRelationNotMergeable", err)
	}
}

// TestAdminRelationCreateAndList 手动建关系 + 分页列表。
func TestAdminRelationCreateAndList(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	created, err := AdminRelationCreate(AdminRelationCreateInput{
		FromCourseId: fromId,
		ToCourseId:   toId,
		RelationType: string(course.RelationRelated),
		Evidence:     `{"manual":true}`,
		Confidence:   1,
	})
	if err != nil {
		t.Fatalf("create relation: %v", err)
	}
	if created.Source != course.RelationSourceManual {
		t.Errorf("source = %q, want manual", created.Source)
	}
	if created.Status != string(course.RelationStatusPending) {
		t.Errorf("status = %q, want pending", created.Status)
	}
	// 幂等：同 (from,to,type) 再建返回已存在行。
	again, err := AdminRelationCreate(AdminRelationCreateInput{
		FromCourseId: fromId,
		ToCourseId:   toId,
		RelationType: string(course.RelationRelated),
	})
	if err != nil {
		t.Fatalf("re-create relation: %v", err)
	}
	if again.Id != created.Id {
		t.Errorf("re-created id = %d, want %d（幂等）", again.Id, created.Id)
	}
	// 分页列表。
	page, err := AdminRelationList(course.RelationQuery{Status: string(course.RelationStatusPending), Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("list relations: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("total = %d, want 1", page.Total)
	}
	if len(page.List) != 1 {
		t.Fatalf("list len = %d, want 1", len(page.List))
	}
	item := page.List[0]
	if item.FromCourse == nil || item.ToCourse == nil {
		t.Fatalf("list item missing course briefs: from=%+v to=%+v", item.FromCourse, item.ToCourse)
	}
	if item.FromCourse.Id != fromId || item.FromCourse.Name != "高等数学(A)上" || item.FromCourse.PrimaryCode != "M101" {
		t.Errorf("from brief = %+v, want M101 高等数学(A)上", item.FromCourse)
	}
	if item.ToCourse.Id != toId || item.ToCourse.Name != "高等数学(A)I" || item.ToCourse.PrimaryCode != "M102" {
		t.Errorf("to brief = %+v, want M102 高等数学(A)I", item.ToCourse)
	}
	if item.FromCourse.Status != course.StatusVisible || item.ToCourse.Status != course.StatusVisible {
		t.Errorf("brief status = from %d / to %d, want both visible", item.FromCourse.Status, item.ToCourse.Status)
	}
	// 忽略后分页（pending 不再出现）。
	if _, err := AdminRelationIgnore(created.Id); err != nil {
		t.Fatalf("ignore relation: %v", err)
	}
	page, err = AdminRelationList(course.RelationQuery{Status: string(course.RelationStatusPending), Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("list pending relations: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("pending total = %d, want 0", page.Total)
	}
}

// TestAdminRelationApprove SPLIT_FROM 可 approved；EQUIVALENT 拒绝 approve（必须走合并）。
func TestAdminRelationApprove(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	conn := dbconnect.Connect()
	split := course.RelationEntity{FromCourseId: fromId, ToCourseId: toId, RelationType: string(course.RelationSplit), Source: course.RelationSourceRule, Status: string(course.RelationStatusPending)}
	if err := conn.Create(&split).Error; err != nil {
		t.Fatalf("create split relation: %v", err)
	}
	updated, err := AdminRelationApprove(split.Id)
	if err != nil {
		t.Fatalf("approve split: %v", err)
	}
	if updated.Status != string(course.RelationStatusApproved) {
		t.Errorf("status = %q, want approved", updated.Status)
	}
	// EQUIVALENT 不得 approve（必须走合并）。
	equiv := course.RelationEntity{FromCourseId: toId, ToCourseId: fromId, RelationType: string(course.RelationEquivalent), Source: course.RelationSourceRule, Status: string(course.RelationStatusPending)}
	if err := conn.Create(&equiv).Error; err != nil {
		t.Fatalf("create equiv relation: %v", err)
	}
	if _, err := AdminRelationApprove(equiv.Id); !errors.Is(err, ErrRelationNotMergeable) {
		t.Fatalf("approve equiv = %v, want ErrRelationNotMergeable", err)
	}
}

// TestMergeCoursesMigratesBookmarksAndInvalidatesAiSummary 合并迁移收藏：
// - 仅 from 收藏的用户 → to 行以 from 收藏时间写入（原 to 未收藏）
// - 双卡均收藏的用户 → 保留 to 原时间、删除 from 行
// - AI 总结缓存（from/to）随合并失效
// 撤销后 from/to 收藏恢复原状、AI 缓存再次失效。
func TestMergeCoursesMigratesBookmarksAndInvalidatesAiSummary(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	conn := dbconnect.Connect()

	now := time.Now()
	userA := uint64(1001) // 仅收藏 from
	userB := uint64(1002) // from + to 均收藏
	fromBookmarkA := course.CourseUserActionEntity{UserId: userA, CourseId: fromId, BookmarkedAt: &now}
	fromBookmarkB := course.CourseUserActionEntity{UserId: userB, CourseId: fromId, BookmarkedAt: &now}
	if err := conn.Create(&fromBookmarkA).Error; err != nil {
		t.Fatalf("create from bookmark A: %v", err)
	}
	if err := conn.Create(&fromBookmarkB).Error; err != nil {
		t.Fatalf("create from bookmark B: %v", err)
	}
	toTime := now.Add(-time.Hour)
	toBookmarkB := course.CourseUserActionEntity{UserId: userB, CourseId: toId, BookmarkedAt: &toTime}
	if err := conn.Create(&toBookmarkB).Error; err != nil {
		t.Fatalf("create to bookmark B: %v", err)
	}
	// AI 总结缓存：from/to 各一行。
	summary := func(courseId uint64) *course.CourseAiSummaryEntity {
		return &course.CourseAiSummaryEntity{CourseId: courseId, SummaryJson: `{"summary":"x"}`, Status: course.AiSummaryRowStatusGenerated}
	}
	if err := conn.Create(summary(fromId)).Error; err != nil {
		t.Fatalf("create ai summary from: %v", err)
	}
	if err := conn.Create(summary(toId)).Error; err != nil {
		t.Fatalf("create ai summary to: %v", err)
	}

	relation := course.RelationEntity{
		FromCourseId: fromId,
		ToCourseId:   toId,
		RelationType: string(course.RelationEquivalent),
		Source:       course.RelationSourceRule,
		Status:       string(course.RelationStatusPending),
	}
	if err := conn.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}
	if _, err := MergeCourses(relation.Id); err != nil {
		t.Fatalf("merge: %v", err)
	}

	// from 卡收藏行已清空；A 的 to 行以 from 时间收藏；B 的 to 行保留原时间。
	var fromCount int64
	if err := conn.Model(&course.CourseUserActionEntity{}).Where("course_id = ?", fromId).Count(&fromCount).Error; err != nil {
		t.Fatalf("count from bookmarks: %v", err)
	}
	if fromCount != 0 {
		t.Errorf("from bookmarks = %d, want 0（已全部迁移）", fromCount)
	}
	toA := course.GetCourseUserActionTx(conn, userA, toId)
	if toA.Id == 0 || toA.BookmarkedAt == nil {
		t.Errorf("user A to bookmark missing/not set: %+v", toA)
	}
	toB := course.GetCourseUserActionTx(conn, userB, toId)
	if toB.Id == 0 || toB.BookmarkedAt == nil || !toB.BookmarkedAt.Equal(toTime) {
		t.Errorf("user B to bookmark = %+v, want original toTime preserved", toB)
	}
	// AI 总结缓存全部失效。
	for _, id := range []uint64{fromId, toId} {
		var cnt int64
		if err := conn.Model(&course.CourseAiSummaryEntity{}).Where("course_id = ?", id).Count(&cnt).Error; err != nil {
			t.Fatalf("count ai summary %d: %v", id, err)
		}
		if cnt != 0 {
			t.Errorf("ai summary course %d = %d rows, want 0（失效）", id, cnt)
		}
	}

	// 撤销：from 收藏行恢复（A/B 各一）、to 行恢复原状（A 未收藏、B 保留 toTime）。
	if _, err := UndoMergeCourse(relation.Id); err != nil {
		t.Fatalf("undo: %v", err)
	}
	fromA := course.GetCourseUserActionTx(conn, userA, fromId)
	if fromA.Id == 0 || fromA.BookmarkedAt == nil {
		t.Errorf("user A from bookmark after undo missing: %+v", fromA)
	}
	fromB := course.GetCourseUserActionTx(conn, userB, fromId)
	if fromB.Id == 0 || fromB.BookmarkedAt == nil {
		t.Errorf("user B from bookmark after undo missing: %+v", fromB)
	}
	toA2 := course.GetCourseUserActionTx(conn, userA, toId)
	if toA2.Id != 0 && toA2.BookmarkedAt != nil {
		t.Errorf("user A to bookmark after undo = %+v, want cleared", toA2)
	}
	toB2 := course.GetCourseUserActionTx(conn, userB, toId)
	if toB2.Id == 0 || toB2.BookmarkedAt == nil || !toB2.BookmarkedAt.Equal(toTime) {
		t.Errorf("user B to bookmark after undo = %+v, want toTime restored", toB2)
	}
}

// TestAdminRelationIgnoreRejectsNonPending 已批准/已合并的行不可忽略（review P1：
// merged 行被置 ignored 会破坏 UndoMergeCourse 的撤销识别）。
func TestAdminRelationIgnoreRejectsNonPending(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	conn := dbconnect.Connect()
	approved := course.RelationEntity{FromCourseId: fromId, ToCourseId: toId, RelationType: string(course.RelationSplit), Source: course.RelationSourceRule, Status: string(course.RelationStatusApproved)}
	if err := conn.Create(&approved).Error; err != nil {
		t.Fatalf("create approved relation: %v", err)
	}
	if _, err := AdminRelationIgnore(approved.Id); !errors.Is(err, ErrRelationNotMergeable) {
		t.Fatalf("ignore approved = %v, want ErrRelationNotMergeable", err)
	}
	var after course.RelationEntity
	if err := conn.First(&after, approved.Id).Error; err != nil {
		t.Fatalf("find approved relation: %v", err)
	}
	if after.Status != string(course.RelationStatusApproved) {
		t.Errorf("approved status = %q, want unchanged approved", after.Status)
	}

	// 已合并行：同一对卡再建 pending EQUIVALENT 候选（(from,to,type) 唯一索引允许
	// 与 SPLIT_FROM 共存）并合并，再尝试忽略 → 拒绝。
	merged := course.RelationEntity{FromCourseId: fromId, ToCourseId: toId, RelationType: string(course.RelationEquivalent), Source: course.RelationSourceRule, Status: string(course.RelationStatusPending)}
	if err := conn.Create(&merged).Error; err != nil {
		t.Fatalf("create merged relation: %v", err)
	}
	if _, err := MergeCourses(merged.Id); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if _, err := AdminRelationIgnore(merged.Id); !errors.Is(err, ErrRelationNotMergeable) {
		t.Fatalf("ignore merged = %v, want ErrRelationNotMergeable", err)
	}
	// 撤销仍可用（merged 状态未被破坏）。
	if _, err := UndoMergeCourse(merged.Id); err != nil {
		t.Fatalf("undo after ignore attempt: %v", err)
	}
}

// TestAdminRelationCreateConfidenceValidation 置信度越界被服务端拒绝（review P2）。
func TestAdminRelationCreateConfidenceValidation(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	for _, conf := range []float64{-0.1, 1.5} {
		if _, err := AdminRelationCreate(AdminRelationCreateInput{
			FromCourseId: fromId,
			ToCourseId:   toId,
			RelationType: string(course.RelationRelated),
			Confidence:   conf,
		}); !errors.Is(err, ErrRelationConfidenceInvalid) {
			t.Fatalf("confidence %v = %v, want ErrRelationConfidenceInvalid", conf, err)
		}
	}
}

// TestUndoMergeCourseRestoresPreMergeStatus 撤销合并还原 from 卡合并前状态：
// 合并前已被其它原因隐藏的卡，撤销后保持隐藏（review Should），而非无条件恢复可见。
func TestUndoMergeCourseRestoresPreMergeStatus(t *testing.T) {
	setupMergeTest(t)
	fromId, toId, _ := seedMergePair(t)
	conn := dbconnect.Connect()
	if err := conn.Model(&course.Entity{}).Where("id = ?", fromId).
		Update("status", course.StatusHidden).Error; err != nil {
		t.Fatalf("pre-hide from course: %v", err)
	}
	relation := course.RelationEntity{
		FromCourseId: fromId,
		ToCourseId:   toId,
		RelationType: string(course.RelationEquivalent),
		Source:       course.RelationSourceRule,
		Status:       string(course.RelationStatusPending),
	}
	if err := conn.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}
	if _, err := MergeCourses(relation.Id); err != nil {
		t.Fatalf("merge: %v", err)
	}
	var mergedFrom course.Entity
	if err := conn.First(&mergedFrom, fromId).Error; err != nil {
		t.Fatalf("find merged from course: %v", err)
	}
	if mergedFrom.Status != course.StatusHidden {
		t.Fatalf("from status after merge = %d, want hidden", mergedFrom.Status)
	}
	if _, err := UndoMergeCourse(relation.Id); err != nil {
		t.Fatalf("undo: %v", err)
	}
	var from course.Entity
	if err := conn.First(&from, fromId).Error; err != nil {
		t.Fatalf("find from course: %v", err)
	}
	if from.Status != course.StatusHidden {
		t.Errorf("from status after undo = %d, want hidden（还原合并前状态）", from.Status)
	}
}
