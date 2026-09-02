package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice/lineage"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-lineage-scan <calendarId...>",
		Short: "扫描指定学期（一系统 calendarId）的 PK 课程数据，dry-run 输出课程沿革候选 JSON",
		Long: `扫描指定学期（一系统 calendarId）的 PK 课程数据，按课程沿革规则引擎
（R1 同码等价 / R2 新码等价 / R3 改名延续 / R4 拆分候选 / R5 硬分隔）对课程两两配对，
输出 LineageCandidate JSON 数组到 stdout。

纯 dry-run：只读 PK 域数据，不写库、不产生任何副作用。
候选的 FromCourseID/ToCourseID 为 pk_course_detail.id（一系统教学班 id）。`,
		Args: cobra.MinimumNArgs(1),
		RunE: runCourseLineageScan,
	}
	appendCommand(cmd)
}

// runCourseLineageScan 读取各学期的教学班与教师，构建课程摘要并跑沿革规则，
// 候选 JSON 直接输出到 stdout（dry-run，不写库）。
func runCourseLineageScan(_ *cobra.Command, args []string) error {
	calendarIds := make([]uint64, 0, len(args))
	for _, arg := range args {
		id, err := strconv.ParseUint(arg, 10, 64)
		if err != nil || id == 0 {
			return fmt.Errorf("course-lineage-scan: 无效的 calendarId %q", arg)
		}
		calendarIds = append(calendarIds, id)
	}

	courses, err := loadLineageSummaries(calendarIds)
	if err != nil {
		return fmt.Errorf("course-lineage-scan: %w", err)
	}
	candidates := lineage.EvaluateAll(courses)

	data, err := json.MarshalIndent(candidates, "", "  ")
	if err != nil {
		return fmt.Errorf("course-lineage-scan: 序列化候选 JSON: %w", err)
	}
	fmt.Println(string(data))
	fmt.Printf("course-lineage-scan: calendars=%v courses=%d candidates=%d (dry-run)\n",
		calendarIds, len(courses), len(candidates))
	return nil
}

// loadLineageSummaries 读取指定学期的 PK 教学班与教师，构建课程沿革摘要。
// 课时取一系统 period（总学时）×10；学期标记取 pk_calendar.calendar_id_i18n。
func loadLineageSummaries(calendarIds []uint64) ([]lineage.CourseSummary, error) {
	var courses []lineage.CourseSummary
	for _, calendarId := range calendarIds {
		cal, err := pk.GetCalendarByID(calendarId)
		if err != nil {
			return nil, fmt.Errorf("读取学期 %d: %w", calendarId, err)
		}
		details, err := pk.ListCourseDetailsByCalendar(calendarId)
		if err != nil {
			return nil, err
		}
		teachers, err := classTeachersByID(details)
		if err != nil {
			return nil, err
		}
		for _, d := range details {
			name := d.CourseName
			if name == "" {
				name = d.Name
			}
			if name == "" {
				name = d.CourseCode
			}
			summary := lineage.CourseSummary{
				ID:            d.Id,
				CourseCode:    d.CourseCode,
				NewCourseCode: d.NewCourseCode,
				Name:          name,
				Semester:      cal.CalendarIdI18n,
			}
			if d.Credit != nil {
				summary.Credit = *d.Credit
			}
			if d.Period != nil {
				summary.HourX10 = int(*d.Period * 10)
			}
			if list := teachers[d.Id]; len(list) > 0 {
				summary.TeacherCode = list[0].TeacherCode // 教学班首位教师
			}
			courses = append(courses, summary)
		}
	}
	return courses, nil
}

// classTeachersByID 批量读取教学班教师，按 teachingClassId 分组（保持教师顺序）。
func classTeachersByID(details []pk.CourseDetailEntity) (map[uint64][]pk.TeacherEntity, error) {
	classIds := make([]uint64, 0, len(details))
	for _, d := range details {
		classIds = append(classIds, d.Id)
	}
	if len(classIds) == 0 {
		return nil, nil
	}
	rows, err := pk.ListTeachersByClassIds(classIds)
	if err != nil {
		return nil, err
	}
	byClass := map[uint64][]pk.TeacherEntity{}
	for _, t := range rows {
		byClass[t.TeachingClassId] = append(byClass[t.TeachingClassId], t)
	}
	return byClass, nil
}
