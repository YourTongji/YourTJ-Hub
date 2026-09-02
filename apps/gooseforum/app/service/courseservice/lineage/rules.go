package lineage

import (
	"encoding/json"
	"fmt"
	"math"
)

// LineageCandidate 课程沿革候选：从 FromCourse 指向 ToCourse（时间上旧 → 新）。
// Evidence 是 JSON 字符串（教师重叠 teacherCode、名称 diff、学分课时、学期连续性），
// 供人工确认与后续 Phase 落库时追溯判定依据。
type LineageCandidate struct {
	FromCourseID uint64  `json:"fromCourseId"`
	ToCourseID   uint64  `json:"toCourseId"`
	RelationType string  `json:"relationType"`
	Source       string  `json:"source"`
	Evidence     string  `json:"evidence"`
	Confidence   float64 `json:"confidence"`
}

// 候选关系类型（R1-R5 产出）。
const (
	RelationEquivalent  = "EQUIVALENT"
	RelationRenamedFrom = "RENAMED_FROM"
	RelationSplitFrom   = "SPLIT_FROM"
	RelationRelated     = "RELATED"
)

// CourseSummary 课程沿革规则输入：单个课程的结构化摘要（来自 PK 域/课程目录）。
// Semester 为学期标记（如 "2025-2026-1"），相邻学期用于 R3 学期连续性判定；
// HourX10 为总课时×10（规避浮点，与 CreditX10 同风格）。
type CourseSummary struct {
	ID            uint64
	CourseCode    string  // 一系统 courseCode（旧编码体系）
	NewCourseCode string  // 新课程编码（2026 改制后；可为空）
	TeacherCode   string  // 教师工号（教师重叠判定）
	Name          string  // 课程名（原始形态，规则内部归一）
	Credit        float64 // 学分
	HourX10       int     // 总课时×10（未知为 0）
	Semester      string  // 学期标记（如 "2025-2026-1"）
}

// evidence 规则的 Evidence JSON 载荷（字段与 Blueprint 证据清单对应）。
type evidence struct {
	TeacherCodeOverlap bool    `json:"teacherCodeOverlap,omitempty"`
	CourseCode         string  `json:"courseCode,omitempty"`
	NewCourseCode      string  `json:"newCourseCode,omitempty"`
	NormalizedNameDiff string  `json:"normalizedNameDiff,omitempty"`
	CreditDiff         float64 `json:"creditDiff,omitempty"`
	CreditDelta        float64 `json:"creditDelta,omitempty"`
	HourDeltaRatio     float64 `json:"hourDeltaRatio,omitempty"`
	SemesterAdjacent   bool    `json:"semesterAdjacent,omitempty"`
	Semesters          string  `json:"semesters,omitempty"`
	FamilyKey          string  `json:"familyKey,omitempty"`
	VariantFrom        string  `json:"variantFrom,omitempty"`
	VariantTo          string  `json:"variantTo,omitempty"`
}

// marshalEvidence 序列化证据载荷；结构体字段全部可序列化，错误只可能来自
// 内部类型误用，此处折叠为字符串渲染，避免污染规则返回值。
func marshalEvidence(ev evidence) string {
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Sprintf(`{"marshalError":%q}`, err.Error())
	}
	return string(data)
}

// R1 同 teacherCode + 同 courseCode + 归一名称一致 + 学分一致 → EQUIVALENT。
// 同一教师同一旧课程码，名称与学分全同：视为同一课程的重复学期记录。
func R1(from, to CourseSummary) *LineageCandidate {
	if !sameTeacher(from, to) || from.CourseCode == "" || from.CourseCode != to.CourseCode {
		return nil
	}
	if NormalizeCourseName(from.Name) != NormalizeCourseName(to.Name) {
		return nil
	}
	if !sameCredit(from.Credit, to.Credit) {
		return nil
	}
	return &LineageCandidate{
		FromCourseID: from.ID,
		ToCourseID:   to.ID,
		RelationType: RelationEquivalent,
		Source:       "R1",
		Evidence: marshalEvidence(evidence{
			TeacherCodeOverlap: true,
			CourseCode:         from.CourseCode,
			NormalizedNameDiff: "",
			CreditDiff:         0,
		}),
		Confidence: 0.95,
	}
}

// R2 同 teacherCode + newCourseCode 相同 + 名称学分一致 → EQUIVALENT。
// 2026 改制后新旧课程码不同，但新编码相同、名称学分一致：视为同一课程。
func R2(from, to CourseSummary) *LineageCandidate {
	if !sameTeacher(from, to) || from.NewCourseCode == "" || from.NewCourseCode != to.NewCourseCode {
		return nil
	}
	if NormalizeCourseName(from.Name) != NormalizeCourseName(to.Name) {
		return nil
	}
	if !sameCredit(from.Credit, to.Credit) {
		return nil
	}
	return &LineageCandidate{
		FromCourseID: from.ID,
		ToCourseID:   to.ID,
		RelationType: RelationEquivalent,
		Source:       "R2",
		Evidence: marshalEvidence(evidence{
			TeacherCodeOverlap: true,
			NewCourseCode:      from.NewCourseCode,
			NormalizedNameDiff: "",
			CreditDiff:         0,
		}),
		Confidence: 0.9,
	}
}

// R3 同 teacherCode + 归一名称相同 + 学分接近(±0.5) + 课时接近(±25%) + 学期相邻
// → RENAMED_FROM。名称一致但课时/学分轻微调整、且学期连续：改名延续。
// 课时未知（HourX10=0）或相同时视为满足课时条件；学期无法解析（Semester 空）
// 时不产出（学期相邻是 R3 的必要条件）。
func R3(from, to CourseSummary) *LineageCandidate {
	if !sameTeacher(from, to) {
		return nil
	}
	if NormalizeCourseName(from.Name) != NormalizeCourseName(to.Name) {
		return nil
	}
	creditDelta := to.Credit - from.Credit
	if math.Abs(creditDelta) > 0.5 {
		return nil
	}
	if !hoursClose(from.HourX10, to.HourX10) {
		return nil
	}
	if !semestersAdjacent(from.Semester, to.Semester) {
		return nil
	}
	return &LineageCandidate{
		FromCourseID: from.ID,
		ToCourseID:   to.ID,
		RelationType: RelationRenamedFrom,
		Source:       "R3",
		Evidence: marshalEvidence(evidence{
			TeacherCodeOverlap: true,
			CreditDelta:        creditDelta,
			HourDeltaRatio:     hourDeltaRatio(from.HourX10, to.HourX10),
			SemesterAdjacent:   true,
			Semesters:          from.Semester + "→" + to.Semester,
		}),
		Confidence: 0.75,
	}
}

// R4 family_key 相同 + variant 不同 → SPLIT_FROM 候选。
// 同一课程家族下变体不同（A1/A2/B、基础/进阶、上/下 等）：课程拆分/重组候选。
// 两门课名相同（无变体差异）不产出。
func R4(from, to CourseSummary) *LineageCandidate {
	fromFamily, toFamily := FamilyKey(from.Name), FamilyKey(to.Name)
	if fromFamily == "" || fromFamily != toFamily {
		return nil
	}
	fromVariant, toVariant := VariantKey(from.Name), VariantKey(to.Name)
	if fromVariant == "" || fromVariant == toVariant {
		return nil
	}
	return &LineageCandidate{
		FromCourseID: from.ID,
		ToCourseID:   to.ID,
		RelationType: RelationSplitFrom,
		Source:       "R4",
		Evidence: marshalEvidence(evidence{
			FamilyKey:          fromFamily,
			VariantFrom:        fromVariant,
			VariantTo:          toVariant,
			TeacherCodeOverlap: sameTeacher(from, to),
			Semesters:          from.Semester + "→" + to.Semester,
		}),
		Confidence: 0.5,
	}
}

// R5 硬分隔：变体语义冲突（A1≠A2≠B、基础≠进阶、实验≠理论、
// 课程设计/实习≠普通课堂）、学分课时巨变、newCourseCode 历史复用、
// 名称语义明显不同 → 永不产出 EQUIVALENT，仅 RELATED 或忽略。
//
// 返回 nil 表示忽略（不产出任何候选）；RELATED 表示弱关联（同家族或同教师同码，
// 但存在硬分隔，绝不 EQUIVALENT）。调用顺序上 R5 是最终守卫：R1/R2 命中后仍须
// 经 R5 复核，命中硬分隔时降级为 RELATED。
func R5(from, to CourseSummary) *LineageCandidate {
	hard := ""
	switch {
	case isHardSemanticVariant(VariantKey(from.Name), VariantKey(to.Name)):
		hard = "variantSemanticConflict"
	case creditChangedDramatically(from.Credit, to.Credit):
		hard = "creditChangedDramatically"
	case hoursChangedDramatically(from.HourX10, to.HourX10):
		hard = "hoursChangedDramatically"
	case newCourseCodeReused(from, to):
		hard = "newCourseCodeReused"
	case namesSemanticallyDifferent(from.Name, to.Name):
		hard = "namesSemanticallyDifferent"
	}
	if hard == "" {
		return nil
	}
	// 硬分隔成立但家族相同（或同教师同码）时仍给 RELATED 弱关联，便于人工核查；
	// 家族不同视为课程完全不同，直接忽略。
	if FamilyKey(from.Name) != FamilyKey(to.Name) {
		return nil
	}
	return &LineageCandidate{
		FromCourseID: from.ID,
		ToCourseID:   to.ID,
		RelationType: RelationRelated,
		Source:       "R5",
		Evidence: marshalEvidence(evidence{
			TeacherCodeOverlap: sameTeacher(from, to),
			CourseCode:         from.CourseCode,
			FamilyKey:          FamilyKey(from.Name),
			VariantFrom:        VariantKey(from.Name),
			VariantTo:          VariantKey(to.Name),
			Semesters:          from.Semester + "→" + to.Semester,
		}),
		Confidence: 0.2,
	}
}

// Evaluate 对一对课程运行 R1-R5，返回候选列表。
// 顺序约定：先跑 R1/R2/R3/R4 产出正候选，再跑 R5 硬分隔复核——
// R5 命中（硬分隔成立）时移除已收集的 EQUIVALENT 候选（等价断言被硬分隔推翻，
// 保留会误导人工审核），并追加 R5 的 RELATED 弱关联；RENAMED_FROM/SPLIT_FROM
// 保持（它们本身不是等价断言）。
func Evaluate(from, to CourseSummary) []LineageCandidate {
	var candidates []LineageCandidate
	appendIf := func(c *LineageCandidate) {
		if c != nil {
			candidates = append(candidates, *c)
		}
	}
	appendIf(R1(from, to))
	appendIf(R2(from, to))
	appendIf(R3(from, to))
	appendIf(R4(from, to))
	if r5 := R5(from, to); r5 != nil {
		// R5 永不产出 EQUIVALENT（仅 RELATED 或忽略），命中即硬分隔成立：
		// 把 R1/R2 已收集的 EQUIVALENT 移除，再追加 RELATED。
		filtered := candidates[:0]
		for _, c := range candidates {
			if c.RelationType != RelationEquivalent {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
		appendIf(r5)
	}
	return candidates
}

// ---- 判定辅助 ----

// sameTeacher 教师重叠：teacherCode 均非空且相等。
func sameTeacher(from, to CourseSummary) bool {
	return from.TeacherCode != "" && from.TeacherCode == to.TeacherCode
}

// sameCredit 学分一致（浮点直接比较，输入来自同一数据源的小数字面量）。
func sameCredit(a, b float64) bool {
	return a == b
}

// hoursClose 课时接近：双方均有值（HourX10>0）时差 ≤ 25%；任一未知或相等视为满足。
func hoursClose(from, to int) bool {
	if from == 0 || to == 0 || from == to {
		return true
	}
	ratio := float64(to) / float64(from)
	return ratio >= 0.75 && ratio <= 1.25
}

// hourDeltaRatio 课时差比例（to/from-1），未知课时返回 0（仅用于证据展示）。
func hourDeltaRatio(from, to int) float64 {
	if from == 0 {
		return 0
	}
	return math.Round((float64(to)/float64(from)-1)*100) / 100
}

// creditChangedDramatically 学分巨变（R5 硬分隔）：|Δ| ≥ 2 或翻倍/减半以上。
func creditChangedDramatically(from, to float64) bool {
	delta := math.Abs(to - from)
	if delta >= 2 {
		return true
	}
	if from > 0 {
		ratio := to / from
		return ratio >= 2 || ratio <= 0.5
	}
	return false
}

// hoursChangedDramatically 课时巨变（R5 硬分隔）：双方有值且差 ≥ 50%。
func hoursChangedDramatically(from, to int) bool {
	if from == 0 || to == 0 {
		return false
	}
	ratio := float64(to) / float64(from)
	return ratio >= 1.5 || ratio <= 1.0/1.5
}

// newCourseCodeReused 历史复用（R5 硬分隔）：双方 newCourseCode 相同但课程名
// 归一后语义不同——同一新编码被不同课程复用，不能据编码判等价。
func newCourseCodeReused(from, to CourseSummary) bool {
	if from.NewCourseCode == "" || from.NewCourseCode != to.NewCourseCode {
		return false
	}
	return NormalizeCourseName(from.Name) != NormalizeCourseName(to.Name)
}

// namesSemanticallyDifferent 名称语义明显不同（R5 硬分隔）：家族不同视为
// 完全不同的课程（硬分隔的兜底判定，供 RELATED/忽略决策）。
func namesSemanticallyDifferent(from, to string) bool {
	f, t := FamilyKey(from), FamilyKey(to)
	return f != "" && t != "" && f != t
}

// semestersAdjacent 学期相邻：双方均为 "YYYY-YYYY-N" 形式时差一个学期。
// 任一侧无法解析返回 false（R3 依赖学期连续性，不猜测）。
func semestersAdjacent(a, b string) bool {
	ai, aok := semesterIndex(a)
	bi, bok := semesterIndex(b)
	return aok && bok && absInt(ai-bi) == 1
}

// semesterIndex 把 "YYYY-YYYY-N" 解析为绝对学期序号：以起始年×3 为基准（每年 3 学期），
// N=1 → start*3+1，N=2 → +2，N=3 → +3，跨年自然顺延（2025-2026-1 紧随 2024-2025-3）。
// 解析失败返回 (0, false)。
func semesterIndex(s string) (int, bool) {
	// 形态 "YYYY-YYYY-N"：年份段两个四位数，学期段 1-3。
	start, _, term, ok := parseSemesterParts(s)
	if !ok {
		return 0, false
	}
	return start*3 + term, true
}

// parseSemesterParts 拆解学期串为 (起始年, 结束年, 学期数)；格式不符返回 false。
func parseSemesterParts(s string) (int, int, int, bool) {
	startIdx, endIdx, termIdx := -1, -1, -1
	dashCount := 0
	partStart := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '-' {
			part := s[partStart:i]
			switch dashCount {
			case 0:
				if !isFourDigits(part) {
					return 0, 0, 0, false
				}
				startIdx = partStart
			case 1:
				if !isFourDigits(part) {
					return 0, 0, 0, false
				}
				endIdx = partStart
			case 2:
				if part == "1" || part == "2" || part == "3" {
					termIdx = partStart
				}
			}
			dashCount++
			partStart = i + 1
		}
	}
	if startIdx < 0 || endIdx < 0 || termIdx < 0 {
		return 0, 0, 0, false
	}
	var start, end, term int
	if _, err := fmt.Sscanf(s[startIdx:startIdx+4], "%d", &start); err != nil {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(s[endIdx:endIdx+4], "%d", &end); err != nil {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(s[termIdx:termIdx+1], "%d", &term); err != nil {
		return 0, 0, 0, false
	}
	if end != start+1 {
		return 0, 0, 0, false
	}
	return start, end, term, true
}

// isFourDigits 判断切片是否恰为四位数字。
func isFourDigits(s string) bool {
	if len(s) != 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// absInt 整数绝对值。
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
