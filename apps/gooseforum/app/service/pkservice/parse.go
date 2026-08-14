// Package pkservice 实现一系统（同济一系统）排课数据同步管线（Issue #186）：
// CLI course-pk-sync 登录 → 分页抓取 manualArrange/page → 事务批量写入 PK 域 → PkFetchLog 断点续跑
// → teacher_timeslots 重建 →（可选）物化课程目录。
//
// 排课文本解析（splitEndline/dayMap/timeTextToArray/weekTextToArray/ArrangementInfo/arrangementTextToObj）
// 收敛在 arrange.go（Issue #187 查询 API 侧），本文件仅保留同步管线独用的专业名/课号解析与派生函数。
package pkservice

import (
	"regexp"
	"strconv"
	"strings"
)

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
