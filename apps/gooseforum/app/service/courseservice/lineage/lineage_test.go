package lineage

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCourseName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"高等数学A(I)", "高等数学a(i)"},                             // 全角括号 → 半角、大写 → 小写
		{"高级语言程序设计A１", "高级语言程序设计a1"},                         // 全角数字 → 半角（NFKC）
		{"Ａdvanced Ｃourse　Design", "advanced course design"}, // 全角字母/空格归一
		{"大学物理（上）", "大学物理(上)"},                               // 全角括号统一
		{"  线性代数   ", "线性代数"},                                // 空白归一
		{"理论力学A.", "理论力学a"},                                  // 尾部句点清洗
		{"实验", "实验"},                                         // 硬语义 token 保留
		{"高等数学A1", "高等数学a1"},                                 // A1 不并入 A
	}
	for _, c := range cases {
		if got := NormalizeCourseName(c.in); got != c.want {
			t.Errorf("NormalizeCourseName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFamilyKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"高等数学A(I)", "高等数学"},
		{"高等数学（上）", "高等数学"},
		{"高等数学上", "高等数学"},
		{"高等数学A1", "高等数学"},
		{"高等数学A2", "高等数学"},
		{"高等数学B", "高等数学"},
		{"高级语言程序设计A1", "高级语言程序设计"},
		{"高级语言程序设计进阶", "高级语言程序设计"},
		{"数据结构（课程设计）", "数据结构"},
		{"大学物理实验", "大学物理"},
		{"英语（荣）", "英语"},
		{"高等数学I", "高等数学"},
		{"高等数学II", "高等数学"},
		{"高等数学Ⅲ", "高等数学"}, // NFKC 罗马数字兼容字符
		{"数学分析", "数学分析"},  // 无变体
		{"A1", ""},        // 全是变体 token
	}
	for _, c := range cases {
		if got := FamilyKey(c.in); got != c.want {
			t.Errorf("FamilyKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestVariantKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"高等数学A(I)", "a(i)"},
		{"高等数学（上）", "(上)"},
		{"高等数学上", "上"},
		{"高等数学A1", "a1"},
		{"高等数学A2", "a2"},
		{"高等数学B", "b"},
		{"高等数学基础", "基础"},
		{"高等数学进阶", "进阶"},
		{"高等数学实验", "实验"},
		{"高等数学", ""},
		{"A1", "a1"}, // 全变体名称原样归一
	}
	for _, c := range cases {
		if got := VariantKey(c.in); got != c.want {
			t.Errorf("VariantKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 硬语义 token 保留：变体之间不得互相合并/等价。
func TestHardSemanticVariants(t *testing.T) {
	pairs := []struct {
		a, b string
		want bool
	}{
		{"a1", "a2", true},
		{"a1", "b", true},
		{"a", "b", true},
		{"b", "c", true},
		{"a", "a1", true},
		{"i", "ii", true},
		{"ii", "iii", true},
		{"上", "下", true},
		{"基础", "进阶", true},
		{"实验", "", true},    // 实验 ≠ 理论
		{"实践", "", true},    // 实践 ≠ 理论
		{"课程设计", "", true},  // 课程设计 ≠ 普通课堂
		{"实习", "", true},    // 实习 ≠ 普通课堂
		{"a1", "a1", false}, // 相同不冲突
		{"", "", false},     // 均空（理论）不冲突
		{"a", "i", true},    // 字母档 ≠ 罗马档
		{"基础", "上", true},   // 层次 ≠ 分册
		{"a1", "实验", true},  // 字母档 ≠ 实践
	}
	for _, p := range pairs {
		if got := isHardSemanticVariant(p.a, p.b); got != p.want {
			t.Errorf("isHardSemanticVariant(%q, %q) = %v, want %v", p.a, p.b, got, p.want)
		}
	}
}

func TestR1(t *testing.T) {
	base := CourseSummary{
		ID: 1, CourseCode: "MATH101", TeacherCode: "T1001",
		Name: "高等数学A(I)", Credit: 5, Semester: "2024-2025-1",
	}
	dup := base
	dup.ID, dup.Semester = 2, "2024-2025-2"
	if c := R1(base, dup); c == nil || c.RelationType != RelationEquivalent || c.Confidence != 0.95 {
		t.Fatalf("R1 same course repeat = %+v, want EQUIVALENT", c)
	}
	diffName := dup
	diffName.Name = "高等数学A(II)"
	if c := R1(base, diffName); c != nil {
		t.Fatalf("R1 name differs = %+v, want nil", c)
	}
	diffCredit := dup
	diffCredit.Credit = 6
	if c := R1(base, diffCredit); c != nil {
		t.Fatalf("R1 credit differs = %+v, want nil", c)
	}
	diffCode := dup
	diffCode.CourseCode = "MATH102"
	if c := R1(base, diffCode); c != nil {
		t.Fatalf("R1 courseCode differs = %+v, want nil", c)
	}
	diffTeacher := dup
	diffTeacher.TeacherCode = "T1002"
	if c := R1(base, diffTeacher); c != nil {
		t.Fatalf("R1 teacher differs = %+v, want nil", c)
	}
}

func TestR2(t *testing.T) {
	old := CourseSummary{
		ID: 1, CourseCode: "MATH101", NewCourseCode: "NC2026-001", TeacherCode: "T1001",
		Name: "高等数学A(I)", Credit: 5, Semester: "2024-2025-1",
	}
	neu := CourseSummary{
		ID: 2, CourseCode: "XM123456", NewCourseCode: "NC2026-001", TeacherCode: "T1001",
		Name: "高等数学A(I)", Credit: 5, Semester: "2025-2026-1",
	}
	if c := R2(old, neu); c == nil || c.RelationType != RelationEquivalent {
		t.Fatalf("R2 same newCourseCode = %+v, want EQUIVALENT", c)
	}
	diffNewCode := neu
	diffNewCode.NewCourseCode = "NC2026-002"
	if c := R2(old, diffNewCode); c != nil {
		t.Fatalf("R2 newCourseCode differs = %+v, want nil", c)
	}
	diffTeacher := neu
	diffTeacher.TeacherCode = "T1002"
	if c := R2(old, diffTeacher); c != nil {
		t.Fatalf("R2 teacher differs = %+v, want nil", c)
	}
	diffCredit := neu
	diffCredit.Credit = 4.5
	if c := R2(old, diffCredit); c != nil {
		t.Fatalf("R2 credit differs = %+v, want nil", c)
	}
}

func TestR3(t *testing.T) {
	from := CourseSummary{
		ID: 1, TeacherCode: "T1001", Name: "程序设计基础", Credit: 4,
		HourX10: 640, Semester: "2024-2025-3",
	}
	// 学分 +0.5、课时 +20%、学期相邻（第3学期 → 下一学年第一学期）→ RENAMED_FROM
	to := CourseSummary{
		ID: 2, TeacherCode: "T1001", Name: "程序设计基础", Credit: 4.5,
		HourX10: 768, Semester: "2025-2026-1",
	}
	if c := R3(from, to); c == nil || c.RelationType != RelationRenamedFrom {
		t.Fatalf("R3 renamed = %+v, want RENAMED_FROM", c)
	}
	// 学期不相邻 → nil
	toFar := to
	toFar.Semester = "2026-2027-1"
	if c := R3(from, toFar); c != nil {
		t.Fatalf("R3 non-adjacent semester = %+v, want nil", c)
	}
	// 学分差超限 → nil
	toCredit := to
	toCredit.Credit = 6
	if c := R3(from, toCredit); c != nil {
		t.Fatalf("R3 credit too far = %+v, want nil", c)
	}
	// 课时差超限 → nil
	toHours := to
	toHours.HourX10 = 960
	if c := R3(from, toHours); c != nil {
		t.Fatalf("R3 hours too far = %+v, want nil", c)
	}
	// 教师不同 → nil
	toTeacher := to
	toTeacher.TeacherCode = "T1002"
	if c := R3(from, toTeacher); c != nil {
		t.Fatalf("R3 teacher differs = %+v, want nil", c)
	}
	// 学期空 → nil（不猜测连续性）
	toEmptySem := to
	toEmptySem.Semester = ""
	if c := R3(from, toEmptySem); c != nil {
		t.Fatalf("R3 empty semester = %+v, want nil", c)
	}
}

func TestR4(t *testing.T) {
	from := CourseSummary{ID: 1, Name: "高等数学A1", Credit: 5, Semester: "2024-2025-1"}
	to := CourseSummary{ID: 2, Name: "高等数学A2", Credit: 5, Semester: "2025-2026-1"}
	if c := R4(from, to); c == nil || c.RelationType != RelationSplitFrom || c.Confidence != 0.5 {
		t.Fatalf("R4 split A1→A2 = %+v, want SPLIT_FROM", c)
	}
	// 基础/进阶 拆分
	fromAdv := CourseSummary{ID: 1, Name: "高级语言程序设计基础"}
	toAdv := CourseSummary{ID: 2, Name: "高级语言程序设计进阶"}
	if c := R4(fromAdv, toAdv); c == nil {
		t.Fatalf("R4 split 基础→进阶 = %+v, want SPLIT_FROM", c)
	}
	// 上/下 拆分
	if c := R4(
		CourseSummary{ID: 1, Name: "大学物理（上）"},
		CourseSummary{ID: 2, Name: "大学物理（下）"},
	); c == nil {
		t.Fatal("R4 split 上→下 = nil, want SPLIT_FROM")
	}
	// 同变体 → nil
	same := CourseSummary{ID: 2, Name: "高等数学A1"}
	if c := R4(from, same); c != nil {
		t.Fatalf("R4 same variant = %+v, want nil", c)
	}
	// 家族不同 → nil
	other := CourseSummary{ID: 2, Name: "线性代数A"}
	if c := R4(from, other); c != nil {
		t.Fatalf("R4 different family = %+v, want nil", c)
	}
	// 名称相同（均无变体）→ nil
	if c := R4(
		CourseSummary{ID: 1, Name: "数学分析"},
		CourseSummary{ID: 2, Name: "数学分析"},
	); c != nil {
		t.Fatalf("R4 no variant = %+v, want nil", c)
	}
}

// R5 硬分隔：A1≠A2≠B 等变体语义冲突下，即使名称/教师/学分都看似一致也绝不
// 产出 EQUIVALENT——Evaluate 必须把 R1/R2 的 EQUIVALENT 降级为 RELATED。
func TestR5HardSeparations(t *testing.T) {
	// A1 → A2：同教师同码同学分，R1 命中但被 R5 硬分隔降级。
	from := CourseSummary{
		ID: 1, CourseCode: "MATH101", TeacherCode: "T1001",
		Name: "高等数学A1", Credit: 5, Semester: "2024-2025-1",
	}
	to := CourseSummary{
		ID: 2, CourseCode: "MATH101", TeacherCode: "T1001",
		Name: "高等数学A2", Credit: 5, Semester: "2024-2025-2",
	}
	cs := Evaluate(from, to)
	if len(cs) == 0 {
		t.Fatal("Evaluate(A1→A2) = empty, want RELATED (hard separation)")
	}
	for _, c := range cs {
		if c.RelationType == RelationEquivalent {
			t.Errorf("A1→A2 produced EQUIVALENT: %+v, hard separation forbids it", c)
		}
	}
	foundRelated := false
	for _, c := range cs {
		if c.RelationType == RelationRelated {
			foundRelated = true
		}
	}
	if !foundRelated {
		t.Errorf("Evaluate(A1→A2) = %+v, want at least one RELATED", cs)
	}

	// 实验 ≠ 理论：同教师同 newCourseCode 时 R2 命中但被降级。
	theory := CourseSummary{
		ID: 1, NewCourseCode: "NC2026-010", TeacherCode: "T1001",
		Name: "大学物理", Credit: 3, Semester: "2024-2025-1",
	}
	lab := CourseSummary{
		ID: 2, NewCourseCode: "NC2026-010", TeacherCode: "T1001",
		Name: "大学物理实验", Credit: 3, Semester: "2024-2025-2",
	}
	for _, c := range Evaluate(theory, lab) {
		if c.RelationType == RelationEquivalent {
			t.Errorf("理论→实验 produced EQUIVALENT: %+v, hard separation forbids it", c)
		}
	}

	// 基础 → 进阶：R4 产出 SPLIT_FROM；R5 变体语义冲突（基础≠进阶）额外给 RELATED，
	// 但绝不产出 EQUIVALENT。
	basic := CourseSummary{ID: 1, Name: "高级语言程序设计基础", Semester: "2024-2025-1"}
	advanced := CourseSummary{ID: 2, Name: "高级语言程序设计进阶", Semester: "2025-2026-1"}
	got := Evaluate(basic, advanced)
	foundSplit := false
	for _, c := range got {
		if c.RelationType == RelationEquivalent {
			t.Errorf("基础→进阶 produced EQUIVALENT: %+v", c)
		}
		if c.RelationType == RelationSplitFrom {
			foundSplit = true
		}
	}
	if !foundSplit {
		t.Fatalf("Evaluate(基础→进阶) = %+v, want SPLIT_FROM", got)
	}
}

// newCourseCode 历史复用：同一新编码被不同课程复用，绝不产出 EQUIVALENT。
func TestNewCourseCodeReuse(t *testing.T) {
	from := CourseSummary{
		ID: 1, CourseCode: "OLD101", NewCourseCode: "NC2026-777", TeacherCode: "T1001",
		Name: "高等数学", Credit: 5, Semester: "2024-2025-1",
	}
	to := CourseSummary{
		ID: 2, CourseCode: "OLD102", NewCourseCode: "NC2026-777", TeacherCode: "T1001",
		Name: "线性代数", Credit: 5, Semester: "2024-2025-2",
	}
	// R2 的守卫自身已拦：newCourseCode 相同但名称不同 → R2 不产出。
	if c := R2(from, to); c != nil {
		t.Fatalf("R2 reused newCourseCode = %+v, want nil", c)
	}
	// R5 显式标记历史复用为硬分隔（同家族判定为不同家族 → 忽略，即不产出 EQUIVALENT）。
	for _, c := range Evaluate(from, to) {
		if c.RelationType == RelationEquivalent {
			t.Errorf("newCourseCode reuse produced EQUIVALENT: %+v", c)
		}
	}
}

// Evidence 是合法 JSON，且包含判定依据字段。
func TestEvidenceJSON(t *testing.T) {
	from := CourseSummary{
		ID: 1, CourseCode: "MATH101", TeacherCode: "T1001",
		Name: "高等数学A(I)", Credit: 5, Semester: "2024-2025-1",
	}
	to := CourseSummary{
		ID: 2, CourseCode: "MATH101", TeacherCode: "T1001",
		Name: "高等数学A(I)", Credit: 5, Semester: "2024-2025-2",
	}
	c := R1(from, to)
	if c == nil {
		t.Fatal("R1 = nil")
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(c.Evidence), &raw); err != nil {
		t.Fatalf("Evidence not valid JSON: %v\n%s", err, c.Evidence)
	}
	if raw["teacherCodeOverlap"] != true {
		t.Errorf("Evidence teacherCodeOverlap = %v, want true", raw["teacherCodeOverlap"])
	}
	if raw["courseCode"] != "MATH101" {
		t.Errorf("Evidence courseCode = %v, want MATH101", raw["courseCode"])
	}
}

// 学分/课时判定辅助的边界测试。
func TestNumericHelpers(t *testing.T) {
	if !hoursClose(640, 768) { // +20%
		t.Error("hoursClose(640, 768) = false, want true")
	}
	if hoursClose(640, 960) { // +50%
		t.Error("hoursClose(640, 960) = true, want false")
	}
	if !hoursClose(0, 960) { // 未知课时视为满足
		t.Error("hoursClose(0, 960) = false, want true")
	}
	if !creditChangedDramatically(2, 6) {
		t.Error("creditChangedDramatically(2, 6) = false, want true")
	}
	if creditChangedDramatically(4, 4.5) {
		t.Error("creditChangedDramatically(4, 4.5) = true, want false")
	}
	if !hoursChangedDramatically(640, 960) {
		t.Error("hoursChangedDramatically(640, 960) = false, want true")
	}
	if hoursChangedDramatically(0, 960) {
		t.Error("hoursChangedDramatically(0, 960) = true, want false")
	}
}

func TestSemesterAdjacency(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"2024-2025-1", "2024-2025-2", true},
		{"2024-2025-2", "2024-2025-3", true},
		{"2024-2025-3", "2025-2026-1", true},  // 跨年相邻
		{"2024-2025-1", "2024-2025-3", false}, // 隔一学期
		{"2024-2025-1", "2025-2026-1", false},
		{"2024-2025-1", "", false}, // 无法解析
		{"", "", false},
		{"2024-2025-1", "2024-2025-1", false}, // 同学期
	}
	for _, c := range cases {
		if got := semestersAdjacent(c.a, c.b); got != c.want {
			t.Errorf("semestersAdjacent(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
