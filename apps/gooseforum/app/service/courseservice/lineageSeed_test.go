package courseservice

import (
	"context"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// seedLineageModels SeedLineage 装配/写库涉及的 course 域 + pk 教学班表。
var seedLineageModels = []any{
	&course.Entity{},
	&course.TermEntity{},
	&course.OfferingEntity{},
	&course.InstructorEntity{},
	&course.RelationEntity{},
	&pk.CourseDetailEntity{},
}

// setupSeedLineageTest 迁移并清空 SeedLineage 相关表。
func setupSeedLineageTest(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(seedLineageModels...); err != nil {
		t.Fatalf("migrate seed lineage tables: %v", err)
	}
	for _, model := range seedLineageModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean seed lineage table: %v", err)
		}
	}
}

// seedLineageInstructor 创建教师并返回其 id。
func seedLineageInstructor(t *testing.T, code, name string) uint64 {
	t.Helper()
	ins := course.InstructorEntity{TeacherCode: code, Name: name, NormalizedName: name}
	if err := db.Connect().Create(&ins).Error; err != nil {
		t.Fatalf("create instructor: %v", err)
	}
	return ins.Id
}

// seedLineageCard 创建一张课程卡（含可见 offering + pk 教学班）。dayOffset 控制创建
// 时间先后（负=更早）。返回课程卡 id。
func seedLineageCard(t *testing.T, code, name string, creditX10 int, teacherId uint64, teachingClassId uint64, pkCourseCode string, dayOffset int) uint64 {
	t.Helper()
	conn := db.Connect()
	c := course.Entity{
		PrimaryCode: code,
		Name:        name,
		CreditX10:   creditX10,
		TeacherId:   teacherId,
		Status:      course.StatusVisible,
	}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	// 更新创建时间以表达新旧卡（GORM 默认填 now，这里显式覆盖）。
	if err := conn.Model(&course.Entity{}).Where("id = ?", c.Id).
		Update("created_at", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, dayOffset)).Error; err != nil {
		t.Fatalf("set created_at: %v", err)
	}
	offering := course.OfferingEntity{
		CourseId:        c.Id,
		TeachingClassId: teachingClassId,
		Status:          course.OfferingStatusVisible,
	}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	if teachingClassId > 0 {
		detail := pk.CourseDetailEntity{Id: teachingClassId, CourseCode: pkCourseCode}
		if err := conn.Create(&detail).Error; err != nil {
			t.Fatalf("create pk detail: %v", err)
		}
	}
	return c.Id
}

// TestSeedLineageDryRunAndWriteEquiv：E1 冗余卡（同师同名同学分、共享 pk code）
// dry-run 报告且不写库；--write 后落一条 pending EQUIVALENT；重复 write 幂等跳过。
func TestSeedLineageDryRunAndWriteEquiv(t *testing.T) {
	setupSeedLineageTest(t)
	teacher := seedLineageInstructor(t, "T001", "张三")
	// 旧卡带教学班后缀码；新卡为规范码；两卡共享 pk course_code=122144。
	oldID := seedLineageCard(t, "12214403", "复变函数与积分变换", 30, teacher, 9001, "122144", -100)
	newID := seedLineageCard(t, "122144", "复变函数与积分变换", 30, teacher, 9002, "122144", 0)
	_ = oldID
	_ = newID

	// dry-run：报告 1 条 EQUIVALENT 候选，不写库。
	report, candidates, _, err := SeedLineage(context.Background(), SeedLineageOptions{})
	if err != nil {
		t.Fatalf("seed dry-run: %v", err)
	}
	if report.CardsLoaded != 2 {
		t.Errorf("cardsLoaded = %d, want 2", report.CardsLoaded)
	}
	if report.EquivCandidates != 1 {
		t.Errorf("equivCandidates = %d, want 1", report.EquivCandidates)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].FromCardID != oldID || candidates[0].ToCardID != newID {
		t.Errorf("equiv direction = %d→%d, want %d→%d", candidates[0].FromCardID, candidates[0].ToCardID, oldID, newID)
	}
	var cnt int64
	db.Connect().Table("course_relations").Count(&cnt)
	if cnt != 0 {
		t.Fatalf("dry-run wrote %d relations, want 0", cnt)
	}

	// --write：落 1 条 pending；再次 --write 幂等跳过。
	report, _, _, err = SeedLineage(context.Background(), SeedLineageOptions{Write: true})
	if err != nil {
		t.Fatalf("seed write: %v", err)
	}
	if report.EquivInserted != 1 || report.EquivSkipped != 0 {
		t.Errorf("write report = inserted %d skipped %d, want 1/0", report.EquivInserted, report.EquivSkipped)
	}
	report, _, _, err = SeedLineage(context.Background(), SeedLineageOptions{Write: true})
	if err != nil {
		t.Fatalf("seed rewrite: %v", err)
	}
	if report.EquivInserted != 0 || report.EquivSkipped != 1 {
		t.Errorf("rewrite report = inserted %d skipped %d, want 0/1（幂等）", report.EquivInserted, report.EquivSkipped)
	}

	var rel course.RelationEntity
	if err := db.Connect().Table("course_relations").
		Where("from_course_id = ? AND to_course_id = ?", oldID, newID).
		First(&rel).Error; err != nil {
		t.Fatalf("relation not found: %v", err)
	}
	if rel.RelationType != string(course.RelationEquivalent) ||
		rel.Status != string(course.RelationStatusPending) || rel.Source != course.RelationSourceRule {
		t.Errorf("relation = type %s status %s source %s, want EQUIVALENT/pending/rule", rel.RelationType, rel.Status, rel.Source)
	}
	if rel.Confidence != 0.9 || rel.EvidenceJson == "" {
		t.Errorf("relation evidence/confidence missing: conf=%v", rel.Confidence)
	}
}

// TestSeedLineageFamilyWriteGate：SPLIT/RELATED（家族标注）默认不落库，仅
// --write-family 才写入（双轨门控）。
func TestSeedLineageFamilyWriteGate(t *testing.T) {
	setupSeedLineageTest(t)
	teacher := seedLineageInstructor(t, "T002", "李四")
	// 同师同家族：generic 旧卡与 A1 新卡（名称学分不同 → SPLIT_FROM，不合并）。
	seedLineageCard(t, "50002440016", "高级语言程序设计", 20, teacher, 0, "", -300)
	seedLineageCard(t, "50007220036", "高级语言程序设计A1", 30, teacher, 0, "", 0)

	// 仅 --write：EQUIVALENT 无候选，SPLIT 被门控 → 0 条写入。
	report, _, _, err := SeedLineage(context.Background(), SeedLineageOptions{Write: true})
	if err != nil {
		t.Fatalf("seed write: %v", err)
	}
	if report.SplitCandidates != 1 {
		t.Fatalf("splitCandidates = %d, want 1", report.SplitCandidates)
	}
	if report.FamilyInserted != 0 {
		t.Fatalf("familyInserted = %d, want 0（SPLIT 须 --write-family 才落库）", report.FamilyInserted)
	}
	var cnt int64
	db.Connect().Table("course_relations").Count(&cnt)
	if cnt != 0 {
		t.Fatalf("relations written without --write-family = %d, want 0", cnt)
	}

	// --write-family：SPLIT 标注落库。
	report, _, _, err = SeedLineage(context.Background(), SeedLineageOptions{Write: true, WriteFamily: true})
	if err != nil {
		t.Fatalf("seed write-family: %v", err)
	}
	if report.FamilyInserted != 1 {
		t.Errorf("familyInserted = %d, want 1", report.FamilyInserted)
	}
	db.Connect().Table("course_relations").Count(&cnt)
	if cnt != 1 {
		t.Fatalf("relations count = %d, want 1", cnt)
	}
}
