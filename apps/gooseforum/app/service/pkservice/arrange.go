package pkservice

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ArrangementInfo 教师时间安排解析结果（对齐上游 utils.ts arrangementTextToObj）。
// OccupyTime/OccupyWeek 为 nil 时 JSON 序列化为 null；TeacherAndCode/OccupyRoom 用指针，
// 无值时序列化为 null（对齐上游）而非空字符串。
type ArrangementInfo struct {
	ArrangementText string  `json:"arrangementText"`
	OccupyDay       *int    `json:"occupyDay"`
	OccupyTime      []int   `json:"occupyTime"`
	OccupyWeek      []int   `json:"occupyWeek"`
	OccupyRoom      *string `json:"occupyRoom"`
	TeacherAndCode  *string `json:"teacherAndCode"`
}

var (
	dayMap = map[string]int{
		"星期一": 1, "星期二": 2, "星期三": 3, "星期四": 4,
		"星期五": 5, "星期六": 6, "星期日": 7,
	}
	dayTextRe   = regexp.MustCompile(`^(星期[一二三四五六日])`)
	timeRe      = regexp.MustCompile(`^星期[一二三四五六日]([0-9]{1,2}-[0-9]{1,2}节)`)
	weekBracket = regexp.MustCompile(`\[([^\]]+)\]`)
)

// splitEndline 按换行拆分并去除空白行。
func splitEndline(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// timeTextToArray 把 "1-2" 展开为 [1, 2]。
func timeTextToArray(text string) []int {
	raw := text
	if strings.HasSuffix(raw, "节") {
		raw = raw[:len(raw)-len("节")]
	}
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return nil
	}
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || start <= 0 || end < start {
		return nil
	}
	out := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		out = append(out, i)
	}
	return out
}

// weekTextToArray 解析周次文本（"1-16"、"1-15周(单)"、"2-14周(双) 15-16"）为周号数组。
func weekTextToArray(text string) []int {
	out := []int{}
	for _, part0 := range strings.Fields(text) {
		part := strings.ReplaceAll(part0, "周", "")
		parity := ""
		if strings.Contains(part, "单") {
			parity = "odd"
		} else if strings.Contains(part, "双") {
			parity = "even"
		}
		cleaned := strings.NewReplacer("(", "", ")", "", "（", "", "）", "", "单", "", "双", "").
			Replace(part)
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == "" {
			continue
		}
		if !strings.Contains(cleaned, "-") {
			n, err := strconv.Atoi(cleaned)
			if err == nil {
				out = append(out, n)
			}
			continue
		}
		bounds := strings.Split(cleaned, "-")
		if len(bounds) != 2 {
			continue
		}
		a, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
		b, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
		if err1 != nil || err2 != nil || a <= 0 || b < a {
			continue
		}
		step := 1
		if parity != "" {
			step = 2
		}
		start := a
		if parity == "odd" && start%2 == 0 {
			start++
		}
		if parity == "even" && start%2 == 1 {
			start++
		}
		for i := start; i <= b; i += step {
			out = append(out, i)
		}
	}
	// 去重 + 排序（稳定 UI）。
	seen := make(map[int]struct{}, len(out))
	uniq := out[:0]
	for _, n := range out {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		uniq = append(uniq, n)
	}
	sort.Ints(uniq)
	return uniq
}

// arrangementTextToObj 把一行安排文本解析为结构化对象。
func arrangementTextToObj(text string) ArrangementInfo {
	text = strings.TrimSpace(text)
	info := ArrangementInfo{ArrangementText: text}
	if text == "" {
		return info
	}
	idx := strings.Index(text, " 星期")
	rest := text
	if idx >= 0 {
		v := strings.TrimSpace(text[:idx])
		info.TeacherAndCode = &v
		rest = strings.TrimSpace(text[idx+1:])
	}
	info.ArrangementText = rest

	if dayMatch := dayTextRe.FindStringSubmatch(rest); len(dayMatch) > 0 {
		if day, ok := dayMap[dayMatch[1]]; ok {
			info.OccupyDay = &day
		}
	}
	if timeMatch := timeRe.FindStringSubmatch(rest); len(timeMatch) > 0 {
		info.OccupyTime = timeTextToArray(timeMatch[1])
	}
	if weekMatch := weekBracket.FindStringSubmatch(rest); len(weekMatch) > 0 {
		info.OccupyWeek = weekTextToArray(weekMatch[1])
	}
	if roomIdx := strings.Index(rest, "] "); roomIdx >= 0 {
		v := strings.TrimSpace(rest[roomIdx+2:])
		info.OccupyRoom = &v
	}
	return info
}

// mergeArrangementInfo 合并多教师安排并按（星期，起始节次）排序（对齐上游）。
func mergeArrangementInfo(teachers []teacherRow) []ArrangementInfo {
	lines := []string{}
	for _, t := range teachers {
		lines = append(lines, splitEndline(t.ArrangeInfoText)...)
	}
	uniq := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		uniq = append(uniq, line)
	}
	objs := make([]ArrangementInfo, 0, len(uniq))
	for _, line := range uniq {
		objs = append(objs, arrangementTextToObj(line))
	}
	sort.SliceStable(objs, func(i, j int) bool {
		ad, bd := 99, 99
		if objs[i].OccupyDay != nil {
			ad = *objs[i].OccupyDay
		}
		if objs[j].OccupyDay != nil {
			bd = *objs[j].OccupyDay
		}
		if ad != bd {
			return ad < bd
		}
		at, bt := 99, 99
		if len(objs[i].OccupyTime) > 0 {
			at = objs[i].OccupyTime[0]
		}
		if len(objs[j].OccupyTime) > 0 {
			bt = objs[j].OccupyTime[0]
		}
		return at < bt
	})
	return objs
}

// teacherRow 教学班教师查询行（teacher 表 + 可选的 i18n 列）。
type teacherRow struct {
	TeachingClassId uint64
	TeacherCode     string
	TeacherName     string
	ArrangeInfoText string
}
