package campusservice

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Explicit presentation projections never return upstream objects. A student's
// own name is returned only in the private profile projection for their greeting.
type Metric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit"`
}
type Point struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}
type Event struct {
	Name    string `json:"name"`
	Teacher string `json:"teacher"`
	Room    string `json:"room"`
	Campus  string `json:"campus"`
	Day     int    `json:"day"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Weeks   []int  `json:"weeks"`
	Credits string `json:"credits"`
}
type Dataset struct {
	Key       string           `json:"key"`
	Status    string           `json:"status"`
	UpdatedAt string           `json:"updatedAt"`
	Metrics   []Metric         `json:"metrics"`
	Columns   []string         `json:"columns"`
	Rows      [][]string       `json:"rows"`
	Events    []Event          `json:"events"`
	Series    []Point          `json:"series"`
	Messages  []MessageSummary `json:"messages,omitempty"`
}

func emptyDataset(key, status string) Dataset {
	return Dataset{Key: key, Status: status, UpdatedAt: time.Now().UTC().Format(time.RFC3339), Metrics: []Metric{}, Columns: []string{}, Rows: [][]string{}, Events: []Event{}, Series: []Point{}}
}
func str(v any) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprint(v)
	if s == "null" {
		return ""
	}
	return s
}
func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
func objects(v any) []map[string]any {
	a, _ := v.([]any)
	r := make([]map[string]any, 0, len(a))
	for _, x := range a {
		if m, ok := x.(map[string]any); ok {
			r = append(r, m)
		}
	}
	return r
}
func number(v any) float64 { n, _ := strconv.ParseFloat(str(v), 64); return n }
func chartNumber(v any) (float64, bool) {
	n, err := strconv.ParseFloat(str(v), 64)
	return n, err == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
}
func first(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := str(m[k]); s != "" {
			return s
		}
	}
	return ""
}
func date(v any) string {
	n := int64(number(v))
	if n == 0 {
		return ""
	}
	return time.UnixMilli(n).In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
}
func normalize(key string, value any) Dataset {
	d := emptyDataset(key, "ready")
	m := obj(value)
	metric := func(label string, v any, unit string) {
		if str(v) != "" {
			d.Metrics = append(d.Metrics, Metric{label, str(v), unit})
		}
	}
	switch key {
	case "profile":
		rows := objects(value)
		if len(rows) == 1 {
			metric("姓名", rows[0]["name"], "")
		}
	case "calendar":
		cal := obj(m["schoolCalendar"])
		metric("当前学期", m["name"], "")
		metric("教学周", m["week"], "周")
		metric("学期周数", cal["weekNum"], "周")
		metric("学期开始", date(cal["beginDay"]), "")
		metric("学期结束", date(cal["endDay"]), "")
	case "timetable":
		d.Columns = []string{"课程", "教师", "学分", "校区", "上课安排"}
		for _, r := range objects(value) {
			d.Rows = append(d.Rows, []string{str(r["courseName"]), str(r["teacherName"]), str(r["credits"]), first(r, "campusI18n", "campus"), str(r["classTime"])})
			for _, t := range objects(r["timeTableList"]) {
				weeks := []int{}
				a, _ := t["weeks"].([]any)
				for _, w := range a {
					weeks = append(weeks, int(number(w)))
				}
				d.Events = append(d.Events, Event{Name: str(r["courseName"]), Teacher: first(t, "teacherName"), Room: first(t, "roomIdI18n", "roomId"), Campus: first(t, "campusI18n", "campus"), Day: int(number(t["dayOfWeek"])), Start: int(number(t["timeStart"])), End: int(number(t["timeEnd"])), Weeks: weeks, Credits: str(r["credits"])})
			}
		}
		metric("本学期课程", len(d.Rows), "门")
	case "grades":
		metric("累计绩点", m["totalGradePoint"], "")
		metric("已获学分", m["actualCredit"], "学分")
		metric("未通过课程", m["failingCourseCount"], "门")
		d.Columns = []string{"学期", "课程", "成绩", "学分", "绩点", "考试性质"}
		for _, term := range objects(m["term"]) {
			name := first(term, "termName", "calName")
			if n, ok := chartNumber(term["averagePoint"]); ok {
				d.Series = append(d.Series, Point{name, n})
			}
			for _, r := range objects(term["creditInfo"]) {
				d.Rows = append(d.Rows, []string{name, str(r["courseName"]), str(r["score"]), str(r["credit"]), str(r["gradePoint"]), first(r, "scoreExamTypeI18n", "scoreEaxmTypeI18n", "scoreName")})
			}
		}
	case "summary":
		rows := objects(value)
		if len(rows) > 0 {
			r := rows[0]
			metric("综合 GPA", r["GPA"], "")
			metric("百分制均分", r["hundredMarkScore"], "")
			metric("已修学分", r["completedCredit"], "学分")
			metric("要求学分", r["requiredCredit"], "学分")
		}
	case "cet":
		d.Columns = []string{"考试学期", "考试科目", "笔试成绩", "口试成绩"}
		for _, r := range objects(m["list"]) {
			name := first(r, "writtenSubjectName", "cetType")
			d.Rows = append(d.Rows, []string{str(r["calendarYearTermCn"]), name, str(r["score"]), str(r["oralScore"])})
			if n, ok := chartNumber(r["score"]); ok {
				d.Series = append(d.Series, Point{name + " · " + str(r["calendarYearTermCn"]), n})
			}
		}
	case "terms":
		d.Columns = []string{"学期", "开始日期", "结束日期", "周数"}
		for _, r := range objects(value) {
			d.Rows = append(d.Rows, []string{str(r["fullName"]), date(r["beginDay"]), date(r["endDay"]), str(r["weekNum"])})
		}
	case "messages":
		d.Columns = []string{"标题", "发布单位", "发布时间"}
		d.Messages = messageSummaries(value)
		for _, r := range d.Messages {
			d.Rows = append(d.Rows, []string{r.Title, r.Publisher, r.PublishedAt})
		}
		metric("消息总数", m["total_"], "条")
	default:
		d.Status = "unavailable"
	}
	if d.Status == "ready" && len(d.Rows) == 0 && len(d.Metrics) == 0 && len(d.Events) == 0 && len(d.Series) == 0 {
		d.Status = "empty"
	}
	return d
}
