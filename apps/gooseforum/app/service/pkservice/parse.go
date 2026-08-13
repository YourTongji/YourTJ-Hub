// Package pkservice 实现一系统（同济一系统）排课数据同步管线（Issue #186）：
// CLI course-pk-sync 登录 → 分页抓取 manualArrange/page → 事务批量写入 PK 域 → PkFetchLog 断点续跑
// → teacher_timeslots 重建 →（可选）物化课程目录。
package pkservice

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// splitEndline 按换行拆解并剔除空行（上游 utils.splitEndline）。
func splitEndline(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

var dayMap = map[string]int{
	"星期一": 1, "星期二": 2, "星期三": 3, "星期四": 4, "星期五": 5, "星期六": 6, "星期日": 7,
}

// timeTextToArray 解析 "1-2节" → [1,2]（上游 utils.timeTextToArray）。
func timeTextToArray(text string) []int {
	raw := strings.TrimSuffix(text, "节")
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return nil
	}
	start, err1 := strconv.Atoi(parts[0])
	end, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || start <= 0 || end < start {
		return nil
	}
	out := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		out = append(out, i)
	}
	return out
}

// weekTextToArray 解析周次文本（"1-16"、"1-15周(单)"、"2-14周(双) 15-16"）→ 周次数组。
func weekTextToArray(text string) []int {
	seen := map[int]bool{}
	var out []int
	for _, part0 := range strings.Fields(text) {
		part := strings.ReplaceAll(part0, "周", "")
		parity := ""
		if strings.Contains(part, "单") {
			parity = "odd"
		} else if strings.Contains(part, "双") {
			parity = "even"
		}
		cleaned := strings.NewReplacer("(", "", ")", "", "（", "", "）", "", "单", "", "双", "").Replace(part)
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == "" {
			continue
		}
		if !strings.Contains(cleaned, "-") {
			if n, err := strconv.Atoi(cleaned); err == nil {
				seen[n] = true
			}
			continue
		}
		rangeParts := strings.Split(cleaned, "-")
		if len(rangeParts) != 2 {
			continue
		}
		a, errA := strconv.Atoi(rangeParts[0])
		b, errB := strconv.Atoi(rangeParts[1])
		if errA != nil || errB != nil || a <= 0 || b < a {
			continue
		}
		step := 1
		start := a
		if parity == "odd" {
			step = 2
			if start%2 == 0 {
				start++
			}
		} else if parity == "even" {
			step = 2
			if start%2 == 1 {
				start++
			}
		}
		for i := start; i <= b; i += step {
			seen[i] = true
		}
	}
	for w := range seen {
		out = append(out, w)
	}
	sort.Ints(out)
	return out
}

// ArrangementInfo 单行排课文本的解析结果。
type ArrangementInfo struct {
	ArrangementText string
	OccupyDay       int
	OccupyTime      []int
	OccupyWeek      []int
	OccupyRoom      string
	TeacherAndCode  string
}

// arrangementTextToObj 解析形如 "教师(T001) 星期一1-2节[1-16周] 校区 教室" 的排课行。
// 移植上游 utils.arrangementTextToObj，仅保留 timeslots 重建所需的字段。
func arrangementTextToObj(text string) ArrangementInfo {
	text = strings.TrimSpace(text)
	if text == "" {
		return ArrangementInfo{}
	}
	idx := strings.Index(text, " 星期")
	rest := text
	var teacherAndCode string
	if idx >= 0 {
		teacherAndCode = strings.TrimSpace(text[:idx])
		rest = strings.TrimSpace(text[idx+1:])
	}
	info := ArrangementInfo{ArrangementText: rest, TeacherAndCode: teacherAndCode}

	dayText := ""
	for day, n := range dayMap {
		if strings.HasPrefix(rest, day) {
			dayText = day
			info.OccupyDay = n
			break
		}
	}

	// 节次："星期X1-2节" 或 "星期X3-4节"
	afterDay := strings.TrimPrefix(rest, dayText)
	if m := timeRangePattern.FindStringSubmatch(afterDay); m != nil {
		info.OccupyTime = timeTextToArray(m[1])
	}

	if m := weekPattern.FindStringSubmatch(rest); m != nil {
		info.OccupyWeek = weekTextToArray(m[1])
	}

	if roomIdx := strings.Index(rest, "] "); roomIdx >= 0 {
		info.OccupyRoom = strings.TrimSpace(rest[roomIdx+2:])
	}
	return info
}

var timeRangePattern = regexp.MustCompile(`^([0-9]{1,2}-[0-9]{1,2}节)`)
var weekPattern = regexp.MustCompile(`\[([^\]]+)\]`)

// parseMajorString 解析专业名 "2025(03074 土木工程(国际班))" → grade/code/name。
func parseMajorString(major string) (grade *int, code string, name string) {
	name = strings.TrimSpace(major)
	if len(name) >= 4 {
		if n, err := strconv.Atoi(name[:4]); err == nil {
			grade = &n
		}
	}
	if m := majorCodePattern.FindStringSubmatch(name); m != nil {
		code = m[1]
	}
	return grade, code, name
}

var majorCodePattern = regexp.MustCompile(`\(([0-9A-Za-z]{3,16})\s`)

// computeNewCode 计算派生课号：newCode = newCourseCode + code 末两位（当 code 以 courseCode 开头时）。
func computeNewCode(newCourseCodeRaw, codeRaw, courseCodeRaw string) (newCourseCode string, newCode string) {
	newCourseCode = strings.TrimSpace(newCourseCodeRaw)
	if newCourseCode == "" {
		return "", ""
	}
	code := strings.TrimSpace(codeRaw)
	courseCode := strings.TrimSpace(courseCodeRaw)
	if code == "" || courseCode == "" || !strings.HasPrefix(code, courseCode) || len(code) < 2 {
		return newCourseCode, ""
	}
	suffix := code[len(code)-2:]
	if suffix == "" {
		return newCourseCode, ""
	}
	return newCourseCode, newCourseCode + suffix
}

// extractTeacherArrangeInfo 只保留与当前教师同行的排课文本（上游 sync.extractTeacherArrangeInfo）。
func extractTeacherArrangeInfo(arrangeInfo, teacherName string) string {
	lines := splitEndline(arrangeInfo)
	if teacherName == "" {
		return strings.Join(lines, "\n")
	}
	var matched []string
	for _, line := range lines {
		if strings.Contains(line, teacherName) {
			matched = append(matched, line)
		}
	}
	if len(matched) > 0 {
		return strings.Join(matched, "\n")
	}
	return strings.Join(lines, "\n")
}
