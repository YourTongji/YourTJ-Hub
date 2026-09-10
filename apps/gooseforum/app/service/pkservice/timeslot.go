package pkservice

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// getPkTimeSlotsBySection 把 PK 行分组 section 映射到节次集合（对齐上游）。
func getPkTimeSlotsBySection(section int) []int {
	switch section {
	case 1:
		return []int{1, 2}
	case 2:
		return []int{3, 4}
	case 3:
		return []int{5, 6}
	case 4:
		return []int{7, 8}
	case 5:
		return []int{9}
	case 6:
		return []int{10}
	default:
		return nil
	}
}

// CoursesByTimeResult P10 courses-by-time 输出：AuxiliaryReady 标记 timeslots 是否就绪。
type CoursesByTimeResult struct {
	AuxiliaryReady bool               `json:"auxiliaryReady"`
	Courses        []SearchCourseItem `json:"courses"`
}

// FindCoursesByTime P10：按时间段查课。timeslots 未就绪时触发后台构建并降级 LIKE 查询。
func FindCoursesByTime(calendarId, day, section int) (CoursesByTimeResult, error) {
	slotSections := getPkTimeSlotsBySection(section)
	if len(slotSections) == 0 {
		return CoursesByTimeResult{}, ErrInvalidParams
	}
	if day < 1 || day > 7 {
		return CoursesByTimeResult{}, ErrInvalidParams
	}

	ready := isPkAuxiliaryReady()
	if !ready {
		TriggerPkAuxiliaryBuild()
	}

	var rows []pk.CourseAggRow
	var err error
	if ready {
		rows, err = pk.ListTimeslotCoursesBySlot(calendarId, day, slotSections, OPTIONAL_LABEL_NAMES)
	} else {
		rows, err = pk.ListTimeslotCoursesByLike(calendarId, buildTimeLikePatterns(day, section), OPTIONAL_LABEL_NAMES)
	}
	if err != nil {
		return CoursesByTimeResult{}, err
	}
	return CoursesByTimeResult{
		AuxiliaryReady: ready,
		Courses:        aggregateSearchCourses(rows),
	}, nil
}

// buildTimeLikePatterns 降级查询的 arrange_info_text LIKE 模式（对齐上游 optCourseQueryListGenerator）。
func buildTimeLikePatterns(day, section int) []string {
	dayTexts := []string{"", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}
	if day < 0 || day >= len(dayTexts) {
		return nil
	}
	dayText := dayTexts[day]
	switch section {
	case 1:
		return []string{fmt.Sprintf("%%%s1-2%%", dayText)}
	case 2:
		return []string{fmt.Sprintf("%%%s3-4%%", dayText)}
	case 3:
		return []string{fmt.Sprintf("%%%s5-6%%", dayText)}
	case 4:
		return []string{fmt.Sprintf("%%%s7-8%%", dayText)}
	case 5:
		return []string{fmt.Sprintf("%%%s9-%%", dayText)}
	case 6:
		return []string{fmt.Sprintf("%%%s10-11%%", dayText), fmt.Sprintf("%%%s10-12%%", dayText)}
	default:
		return nil
	}
}
