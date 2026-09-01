package courseservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// setupMergeTest 迁移并清空合并服务相关表（manageTestModels + course_relations）。
func setupMergeTest(t *testing.T) {
	t.Helper()
	models := append([]any{&course.RelationEntity{}}, manageTestModels...)
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

	if _, err := MergeCourses(rel1.Id); err != ErrMergeConflict {
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
	if _, err := MergeCourses(rel.Id); err != ErrRelationNotMergeable {
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
	if _, err := UndoMergeCourse(relation.Id); err != ErrRelationNotMergeable {
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
	if _, err := AdminRelationApprove(equiv.Id); err != ErrRelationNotMergeable {
		t.Fatalf("approve equiv = %v, want ErrRelationNotMergeable", err)
	}
}
