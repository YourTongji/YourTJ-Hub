package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-pk-sync <学期>",
		Short: "从同济一系统分页同步排课数据到 PK 域（学期/教学班/教师/时间片）",
		Long: `从同济一系统（1.tongji.edu.cn）manualArrange 分页抓取排课数据，事务写入 PK 域，
并按学期重建 teacher_timeslots。研究生受众会分别抓取硕士/博士数据，并使用 X-Token。
可选用 --materialize 联动将课程物化到课程目录。

<学期> 接受两种形式：
  - 一系统数字 calendarId（如 121）
  - 学期名（如 2025-2026-1），经 pk_calendar 反查 calendarId；首次同步请用 --calendar-id

凭证按优先级取：研究生 --onesystem-x-token / 兼容的 --onesystem-cookie 参数 >
受众专用环境变量（研究生 ONESYSTEM_GRADUATE_X_TOKEN 或 ONESYSTEM_X_TOKEN；本科
ONESYSTEM_UNDERGRADUATE_COOKIE）> 旧环境变量/管理端对应设置（加密落库）。`,
		Args: cobra.ExactArgs(1),
		RunE: runCoursePkSync,
	}
	cmd.Flags().Int("depth", 1, "以目标学期为终点向前同步的学期数（默认 1）")
	cmd.Flags().String("onesystem-cookie", "", "本科一系统 Cookie header（研究生兼容作为 X-Token；注意会出现在进程列表，敏感环境慎用）")
	cmd.Flags().String("onesystem-x-token", "", "研究生一系统 X-Token（覆盖环境变量/管理端设置；注意会出现在进程列表，敏感环境慎用）")
	cmd.Flags().String("audience", "undergraduate", "数据来源：undergraduate（本科）或 graduate（研究生）")
	cmd.Flags().Uint64("calendar-id", 0, "显式指定一系统 calendarId（绕过学期名解析）")
	cmd.Flags().Bool("materialize", false, "同步完成后将 PK 课程物化到课程目录（默认关闭）")
	appendCommand(cmd)
}

func runCoursePkSync(cmd *cobra.Command, args []string) error {
	depth, _ := cmd.Flags().GetInt("depth")
	cookieFlag, _ := cmd.Flags().GetString("onesystem-cookie")
	xTokenFlag, _ := cmd.Flags().GetString("onesystem-x-token")
	audienceValue, _ := cmd.Flags().GetString("audience")
	explicitID, _ := cmd.Flags().GetUint64("calendar-id")
	materialize, _ := cmd.Flags().GetBool("materialize")

	audience, ok := pk.ParseAudience(audienceValue)
	if !ok {
		return fmt.Errorf("无效的数据来源 %q：请使用 undergraduate 或 graduate", audienceValue)
	}
	calendarId, err := resolvePkCalendarIdForAudience(args[0], explicitID, audience)
	if err != nil {
		return err
	}
	credentialFlag := cookieFlag
	if audience == pk.AudienceGraduate && strings.TrimSpace(xTokenFlag) != "" {
		credentialFlag = xTokenFlag
	}
	cookie, err := pkservice.ResolveCredentialForAudience(credentialFlag, audience)
	if err != nil {
		return err
	}

	report, err := pkservice.SyncForAudience(cmd.Context(), cookie, audience, calendarId, depth, materialize)
	if err != nil {
		return fmt.Errorf("course-pk-sync: %w", err)
	}

	fmt.Printf("course-pk-sync: audience=%s calendars=%v depth=%d materialize=%v\n", audience, report.CalendarIDs, depth, materialize)
	fmt.Printf("  teachingClass=%d batches=%d pages=%d resumeFromPage=%d\n",
		report.TeachingClassInserted, report.BatchesCommitted, report.FetchedPages, report.ResumedFromPage)
	fmt.Printf("  timeslotsRebuilt=%d materializedCourses=%d\n",
		report.TimeslotsRebuilt, report.MaterializedCourses)
	return nil
}

// resolvePkCalendarId 解析命令行学期参数为数字 calendarId：
//   - --calendar-id 显式优先
//   - 全数字视为 calendarId
//   - 学期名（如 2025-2026-1）经 pk_calendar.calendar_id_i18n 反查
func resolvePkCalendarId(arg string, explicitID uint64) (uint64, error) {
	return resolvePkCalendarIdForAudience(arg, explicitID, pk.AudienceUndergraduate)
}

func resolvePkCalendarIdForAudience(arg string, explicitID uint64, audience pk.Audience) (uint64, error) {
	if explicitID != 0 {
		return explicitID, nil
	}
	arg = strings.TrimSpace(arg)
	if id, err := strconv.ParseUint(arg, 10, 64); err == nil && id > 0 {
		return id, nil
	}
	if id, ok := pk.GetCalendarIdByAudienceI18n(audience, arg); ok {
		return id, nil
	}
	// 归一化反查：pk_calendar.calendar_id_i18n 可能存中文学期名（"2025-2026学年第2学期"）
	// 而调用方传标准码（或相反）——按 NormalizeTermLabel 双向匹配（review P2：CLI 文档
	// 承诺的学期名形式必须对生产中文学期名可用）。
	if normalized := course.NormalizeTermLabel(arg); normalized != "" {
		calendars, err := pk.ListAllCalendarsForAudience(audience)
		if err != nil {
			return 0, fmt.Errorf("列举 pk_calendar 失败：%w", err)
		}
		for _, c := range calendars {
			if course.NormalizeTermLabel(c.CalendarIdI18n) == normalized {
				return pk.ExternalID(audience, c.CalendarId), nil
			}
		}
	}
	return 0, fmt.Errorf("无法解析学期 %q 对应的一系统 calendarId：请先以 --calendar-id 同步一次，或直接传数字 calendarId（如 121）", arg)
}
