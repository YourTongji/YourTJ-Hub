package pkservice

import "testing"

func TestParseMajorString(t *testing.T) {
	grade, code, name := parseMajorString("2025(03074 土木工程(国际班))")
	if grade == nil || *grade != 2025 {
		t.Fatalf("grade = %v, want 2025", grade)
	}
	if code != "03074" {
		t.Errorf("code = %q, want 03074", code)
	}
	if name != "2025(03074 土木工程(国际班))" {
		t.Errorf("name = %q", name)
	}

	// 无 grade/code 前缀
	grade2, code2, _ := parseMajorString("土木工程")
	if grade2 != nil || code2 != "" {
		t.Errorf("plain major: grade=%v code=%q", grade2, code2)
	}
}

func TestComputeNewCode(t *testing.T) {
	newCode := func(code, courseCode string) string {
		_, nc := computeNewCode("122004", code, courseCode)
		return nc
	}
	if got := newCode("12200401", "122004"); got != "12200401" {
		t.Errorf("newCode = %q, want 12200401", got)
	}
	// code 不以 courseCode 开头
	if got := newCode("99999901", "122004"); got != "" {
		t.Errorf("mismatched code newCode = %q, want empty", got)
	}
	// 无 newCourseCode
	if ncc, nc := computeNewCode("", "12200401", "122004"); ncc != "" || nc != "" {
		t.Errorf("no newCourseCode: %q %q", ncc, nc)
	}
}

func TestExtractTeacherArrangeInfo(t *testing.T) {
	info := "张三(T001) 星期一1-2节[1-16周]\n李四(T002) 星期三5-6节[1-16周]"
	// 只保留含目标教师的行
	if got := extractTeacherArrangeInfo(info, "张三"); got != "张三(T001) 星期一1-2节[1-16周]" {
		t.Errorf("filtered = %q", got)
	}
	// 未命中教师 → 全量保留
	if got := extractTeacherArrangeInfo(info, "王五"); got != info {
		t.Errorf("unmatched = %q", got)
	}
}
