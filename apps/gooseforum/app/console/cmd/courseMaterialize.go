package cmd

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-materialize <calendarId|学期>...",
		Short: "将已同步学期的 PK 课程物化到课程目录（本地只读 PK 域，无需一系统 cookie）",
		Long: `将已同步学期的 PK 课程物化到课程目录（course 域 offering 权威写入源）。

与 course-pk-sync --materialize 的区别：本命令只消费本地已同步的 PK 域数据，
不抓取一系统、不需要 cookie——适合学期已同步（管理端 / course-pk-sync）但
物化因 cookie 过期/网络不可用而失败或遗漏后的补跑。

<学期> 接受两种形式：
  - 一系统数字 calendarId（如 122）
  - 学期名（如 2026-2027-1），经 pk_calendar.calendar_id_i18n 反查

物化为幂等 upsert：同 teaching_class_id 已存在则更新字段（不写 status、不复活
管理员隐藏的 offering/课程），可安全重复执行。`,
		Args: cobra.MinimumNArgs(1),
		RunE: runCourseMaterialize,
	}
	appendCommand(cmd)
}

func runCourseMaterialize(cmd *cobra.Command, args []string) error {
	calendarIds := make([]uint64, 0, len(args))
	for _, arg := range args {
		id, err := resolvePkCalendarId(arg, 0)
		if err != nil {
			return fmt.Errorf("course-materialize: %w", err)
		}
		calendarIds = append(calendarIds, id)
	}

	report, err := courseservice.MaterializeFromPk(cmd.Context(), calendarIds)
	if err != nil {
		return fmt.Errorf("course-materialize: %w", err)
	}
	fmt.Printf("course-materialize: calendars=%v coursesInserted=%d coursesUpdated=%d instructorsInserted=%d aliasesInserted=%d aliasesSkipped=%d offeringsInserted=%d offeringsUpdated=%d\n",
		calendarIds, report.CoursesInserted, report.CoursesUpdated, report.InstructorsInserted,
		report.AliasesInserted, report.AliasesSkipped, report.OfferingsInserted, report.OfferingsUpdated)
	return nil
}
