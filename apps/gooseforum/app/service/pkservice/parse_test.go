package pkservice

import (
	"reflect"
	"testing"
)

func TestTimeTextToArray(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"1-2节", []int{1, 2}},
		{"3-4节", []int{3, 4}},
		{"10-12节", []int{10, 11, 12}},
		{"1节", nil},
		{"0-2节", nil},
		{"3-1节", nil},
		{"abc", nil},
	}
	for _, c := range cases {
		if got := timeTextToArray(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("timeTextToArray(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestWeekTextToArray(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"1-16", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
		{"1-15周(单)", []int{1, 3, 5, 7, 9, 11, 13, 15}},
		{"2-14周(双) 15-16", []int{2, 4, 6, 8, 10, 12, 14, 15, 16}},
		{"3", []int{3}},
		{"1-2 1-2", []int{1, 2}},
	}
	for _, c := range cases {
		if got := weekTextToArray(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("weekTextToArray(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestArrangementTextToObj(t *testing.T) {
	info := arrangementTextToObj("张三(T001) 星期一3-4节[1-16周] 四平路校区 汇文楼101")
	if info.OccupyDay != 1 {
		t.Errorf("OccupyDay = %d, want 1", info.OccupyDay)
	}
	if !reflect.DeepEqual(info.OccupyTime, []int{3, 4}) {
		t.Errorf("OccupyTime = %v, want [3 4]", info.OccupyTime)
	}
	if !reflect.DeepEqual(info.OccupyWeek, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}) {
		t.Errorf("OccupyWeek = %v", info.OccupyWeek)
	}
	if info.OccupyRoom != "四平路校区 汇文楼101" {
		t.Errorf("OccupyRoom = %q", info.OccupyRoom)
	}
	if info.TeacherAndCode != "张三(T001)" {
		t.Errorf("TeacherAndCode = %q", info.TeacherAndCode)
	}

	// 无教师前缀（缺少 " 星期" 分隔）
	info2 := arrangementTextToObj("星期五5-6节[1-8周] 教室A")
	if info2.OccupyDay != 5 || !reflect.DeepEqual(info2.OccupyTime, []int{5, 6}) {
		t.Errorf("no-teacher parse = %+v", info2)
	}

	// 空文本
	if empty := arrangementTextToObj(""); empty.OccupyDay != 0 || len(empty.OccupyTime) != 0 {
		t.Errorf("empty parse = %+v", empty)
	}
}

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
