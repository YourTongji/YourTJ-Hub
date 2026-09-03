package lineage

import (
	"testing"
	"time"
)

// mkCard 构造测试用 CardSummary。created 传天数偏移（基准 2026-01-01）。
func mkCard(id uint64, code, name, teacherCode, teacherName string, creditX10, dayOffset int, pkCodes ...string) CardSummary {
	return CardSummary{
		ID:           id,
		PrimaryCode:  code,
		Name:         name,
		TeacherCode:  teacherCode,
		TeacherName:  teacherName,
		CreditX10:    creditX10,
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, dayOffset),
		PkCourseCode: pkCodes,
		Terms:        []string{"2025-2026-1"},
	}
}

// findBy 按 (from,to,type) 找候选。
func findBy(t *testing.T, cands []CardCandidate, from, to uint64, typ string) *CardCandidate {
	t.Helper()
	for i := range cands {
		if cands[i].FromCardID == from && cands[i].ToCardID == to && cands[i].RelationType == typ {
			return &cands[i]
		}
	}
	return nil
}

// TestEvaluateCardsE1RedundantToCanonical：旧卡（带教学班后缀码）与新卡（规范码）
// 同师同名同学分、共享一系统课程码 → 产出旧→新的 EQUIVALENT。
func TestEvaluateCardsE1RedundantToCanonical(t *testing.T) {
	cards := []CardSummary{
		mkCard(100, "12214403", "复变函数与积分变换", "T06078", "周羚君", 30, -100, "122144"),
		mkCard(101, "122144", "复变函数与积分变换", "T06078", "周羚君", 30, 0, "122144"),
	}
	cands := EvaluateCards(cards)
	if len(cands) != 1 {
		t.Fatalf("candidates = %d, want 1", len(cands))
	}
	c := cands[0]
	if c.FromCardID != 100 || c.ToCardID != 101 {
		t.Errorf("E1 direction = %d→%d, want 100→101（旧卡并入规范卡）", c.FromCardID, c.ToCardID)
	}
	if c.RelationType != RelationEquivalent || c.Source != "rule" {
		t.Errorf("E1 type/source = %s/%s, want EQUIVALENT/rule", c.RelationType, c.Source)
	}
	if c.Confidence != 0.9 {
		t.Errorf("E1 confidence = %v, want 0.9", c.Confidence)
	}
}

// TestEvaluateCardsGenericToA1SplitOnly：2026 改制 generic → A1（名称学分均变）不构成
// EQUIVALENT 冗余；同师同家族变体重组只产 SPLIT_FROM 标注，绝不合并。
func TestEvaluateCardsGenericToA1SplitOnly(t *testing.T) {
	cards := []CardSummary{
		mkCard(200, "5000244001603", "高级语言程序设计", "T1817", "沈坚", 20, -300, "CST1201"),
		mkCard(201, "50007220036", "高级语言程序设计A1", "T1817", "沈坚", 30, 0, "CST1216"),
	}
	cands := EvaluateCards(cards)
	if eq := findBy(t, cands, 200, 201, RelationEquivalent); eq != nil {
		t.Fatalf("generic→A1 名称/学分不同，不得产 EQUIVALENT")
	}
	if sp := findBy(t, cands, 200, 201, RelationSplitFrom); sp == nil {
		t.Fatalf("want SPLIT_FROM 200→201（generic 旧卡 → A1 新卡分层标注）, got %+v", cands)
	}
	if len(cands) != 1 {
		t.Fatalf("candidates = %d, want 1（仅 SPLIT 标注）", len(cands))
	}
}

// TestEvaluateCardsE1TargetCanonicalPriority：分量内 primary_code 命中共享码的卡优先作目标。
func TestEvaluateCardsE1TargetCanonicalPriority(t *testing.T) {
	cards := []CardSummary{
		mkCard(300, "5000295003017", "习近平新时代中国特色社会主义思想概论", "T21074", "任博", 30, -200, "CMA1110"),
		mkCard(301, "50002950030", "习近平新时代中国特色社会主义思想概论", "T21074", "任博", 30, 0, "CMA1110"),
	}
	cands := EvaluateCards(cards)
	c := findBy(t, cands, 300, 301, RelationEquivalent)
	if c == nil {
		t.Fatalf("want EQUIVALENT 300→301, got %+v", cands)
	}
}

// TestEvaluateCardsE1NoChain：A→B、B→C 同码链必须收敛为单目标（A→C、B→C），
// 不允许 A→B→C 的链式合并候选。
func TestEvaluateCardsE1NoChain(t *testing.T) {
	// 三张卡共享同一 pk code：任选其一作目标，其余指向它。
	cards := []CardSummary{
		mkCard(400, "36002901", "军事理论", "T09127", "袁品仕", 20, -200, "360029"),
		mkCard(401, "36002902", "军事理论", "T09127", "袁品仕", 20, -100, "360029"),
		mkCard(402, "360029", "军事理论", "T09127", "袁品仕", 20, 0, "360029"),
	}
	cands := EvaluateCards(cards)
	// 恰好 2 条 EQUIVALENT，且 402（规范码）为目标；无 from 指向 400/401 之外的卡。
	if len(cands) != 2 {
		t.Fatalf("EQUIVALENT count = %d, want 2（3 卡共享码收敛到单目标）", len(cands))
	}
	for _, c := range cands {
		if c.ToCardID != 402 {
			t.Errorf("target = %d, want 402（规范码卡作目标）", c.ToCardID)
		}
		if c.RelationType != RelationEquivalent {
			t.Errorf("unexpected type %s", c.RelationType)
		}
	}
}

// TestEvaluateCardsNoCrossTeacher：同码同名师不同 → 不产候选（合法分班）。
func TestEvaluateCardsNoCrossTeacher(t *testing.T) {
	cards := []CardSummary{
		mkCard(500, "110277", "大学英语四级", "T00795", "孙丹", 20, -100, "110277"),
		mkCard(501, "110277", "大学英语四级", "T09999", "其他老师", 20, 0, "110277"),
	}
	cands := EvaluateCards(cards)
	if len(cands) != 0 {
		t.Fatalf("cross-teacher candidates = %d, want 0", len(cands))
	}
}

// TestEvaluateCardsE2SplitVariants：同师同家族变体不同 → SPLIT_FROM 标注（不合并）。
func TestEvaluateCardsE2SplitVariants(t *testing.T) {
	cards := []CardSummary{
		mkCard(600, "50002440016", "高级语言程序设计", "T1817", "沈坚", 20, -300),
		mkCard(601, "50007220036", "高级语言程序设计A1", "T1817", "沈坚", 30, 0),
		mkCard(602, "50007220037", "高级语言程序设计A2", "T1817", "沈坚", 30, 5),
	}
	cands := EvaluateCards(cards)
	// 600(generic)↔601(A1)、600↔602(A2)、601(A1)↔602(A2) 三对 SPLIT。
	if len(cands) != 3 {
		t.Fatalf("candidates = %d, want 3 SPLIT", len(cands))
	}
	for _, c := range cands {
		if c.RelationType != RelationSplitFrom {
			t.Errorf("type = %s, want SPLIT_FROM", c.RelationType)
		}
	}
	// 方向：generic(旧,600) 应指向 A1(601) 而非相反。
	if findBy(t, cands, 600, 601, RelationSplitFrom) == nil {
		t.Errorf("want SPLIT 600→601（generic 旧卡 → A1 新卡）")
	}
}

// TestEvaluateCardsE3RelatedCreditChange：同师同名同学分巨变 → RELATED 弱关联。
func TestEvaluateCardsE3RelatedCreditChange(t *testing.T) {
	cards := []CardSummary{
		mkCard(700, "100717", "高级语言程序设计实验", "T1817", "沈坚", 10, -100),
		mkCard(701, "10071703", "高级语言程序设计实验", "T1817", "沈坚", 40, 0),
	}
	cands := EvaluateCards(cards)
	c := findBy(t, cands, 700, 701, RelationRelated)
	if c == nil {
		t.Fatalf("want RELATED 700→701, got %+v", cands)
	}
}

// TestEvaluateCardsE1SharedNewCodeEquiv：同师同名同学分 + 共享 new_course_code → EQUIVALENT
// （2026 改制码型归一：旧班后缀码卡与新规范码卡同码）。
func TestEvaluateCardsE1SharedNewCodeEquiv(t *testing.T) {
	c3 := mkCard(810, "12201104", "概率论与数理统计", "T16064", "余磊", 30, -100)
	c3.PkNewCode = []string{"CMS1207"}
	c4 := mkCard(811, "122011", "概率论与数理统计", "T16064", "余磊", 30, 0)
	c4.PkNewCode = []string{"CMS1207"}
	cands := EvaluateCards([]CardSummary{c3, c4})
	if len(cands) != 1 || cands[0].RelationType != RelationEquivalent {
		t.Fatalf("want EQUIVALENT 810→811, got %+v", cands)
	}
	if cands[0].FromCardID != 810 || cands[0].ToCardID != 811 {
		t.Errorf("direction = %d→%d, want 810→811（旧码卡并入规范码卡）", cands[0].FromCardID, cands[0].ToCardID)
	}
}

// TestEvaluateCardsEmptyTeacher：无教师卡不参与配对。
func TestEvaluateCardsEmptyTeacher(t *testing.T) {
	cards := []CardSummary{
		mkCard(900, "A001", "专业导论", "", "", 20, 0, "A001"),
		mkCard(901, "A001", "专业导论", "", "", 20, 0, "A001"),
	}
	if cands := EvaluateCards(cards); len(cands) != 0 {
		t.Fatalf("teacher-less candidates = %d, want 0", len(cands))
	}
}
