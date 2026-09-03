package cmd

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
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

<学期> 接受两种形式（解析失败即中止，写入前全量校验）：
  - 一系统数字 calendarId（如 122）
  - 学期名（标准码 2026-2027-1 或中文学期名 2026-2027学年第1学期 均可，
    按 pk_calendar.calendar_id_i18n 归一化反查）

物化为幂等 upsert：同 teaching_class_id 已存在则更新字段（不写 status、不复活
管理员隐藏的 offering/课程），可安全重复执行。重复参数自动去重。`,
		Args: cobra.MinimumNArgs(1),
		RunE: runCourseMaterialize,
	}
	cmd.Flags().Bool("dry-run", false, "只校验学期已同步并打印教学班统计，不写库")
	appendCommand(cmd)
}

// resolveMaterializeCalendars 解析并校验物化目标学期列表（写入前全量校验）：
//   - 重复参数去重（幂等无损坏，但避免报告计数虚高，review）；
//   - 每个学期必须已存在于本地 PK 域（pk_calendar 有记录）——本命令只物化已同步
//     学期，误传未同步/错拼的数字 ID 必须显式报错而非空成功（review P2）。
func resolveMaterializeCalendars(args []string) ([]uint64, error) {
	calendarIds := make([]uint64, 0, len(args))
	seen := map[uint64]bool{}
	for _, arg := range args {
		id, err := resolvePkCalendarId(arg, 0)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, err := pk.GetCalendarByID(id); err != nil {
			return nil, fmt.Errorf("学期 %d 未同步到本地 PK 域（pk_calendar 无记录）：course-materialize 只物化已同步学期，请先 course-pk-sync / 管理端同步（review P2）", id)
		}
		calendarIds = append(calendarIds, id)
	}
	return calendarIds, nil
}

func runCourseMaterialize(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	calendarIds, err := resolveMaterializeCalendars(args)
	if err != nil {
		return fmt.Errorf("course-materialize: %w", err)
	}
	if dryRun {
		for _, cid := range calendarIds {
			rows, err := pk.ListCourseDetailsByCalendar(cid)
			if err != nil {
				return fmt.Errorf("course-materialize: 统计教学班（calendar %d）：%w", cid, err)
			}
			fmt.Printf("course-materialize --dry-run: calendar=%d teachingClasses=%d（未写库；去掉 --dry-run 执行物化）\n", cid, len(rows))
		}
		return nil
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
