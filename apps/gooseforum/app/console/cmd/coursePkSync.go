package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-pk-sync <学期>",
		Short: "从同济一系统分页同步排课数据到 PK 域（学期/教学班/教师/时间片）",
		Long: `从同济一系统（1.tongji.edu.cn）manualArrange 分页抓取排课数据，事务写入 PK 域，
并按学期重建 teacher_timeslots。可选用 --materialize 联动将课程物化到课程目录。

<学期> 接受两种形式：
  - 一系统数字 calendarId（如 121）
  - 学期名（如 2025-2026-1），经 pk_calendar 反查 calendarId；首次同步请用 --calendar-id

凭证按优先级取：--onesystem-cookie 参数 > ONESYSTEM_COOKIE 环境变量 > 管理端设置（加密落库）；
未配置 Cookie 时可用 --onesystem-sno+--onesystem-password（或 ONESYSTEM_SNO/ONESYSTEM_PASSWORD
环境变量）自动 SSO 登录换取会话 Cookie（触发加强认证时仍需 Cookie 凭证）。`,
		Args: cobra.ExactArgs(1),
		RunE: runCoursePkSync,
	}
	cmd.Flags().Int("depth", 1, "以目标学期为终点向前同步的学期数（默认 1）")
	cmd.Flags().String("onesystem-cookie", "", "一系统 Cookie header（覆盖环境变量/管理端设置；注意会出现在进程列表，敏感环境慎用）")
	cmd.Flags().String("onesystem-sno", "", "一系统学号/工号（无 Cookie 时自动 SSO 登录换取会话 Cookie；与 --onesystem-password 成对）")
	cmd.Flags().String("onesystem-password", "", "一系统密码（无 Cookie 时自动 SSO 登录换取会话 Cookie；与 --onesystem-sno 成对）")
	cmd.Flags().Uint64("calendar-id", 0, "显式指定一系统 calendarId（绕过学期名解析）")
	cmd.Flags().Bool("materialize", false, "同步完成后将 PK 课程物化到课程目录（默认关闭）")
	appendCommand(cmd)
}

func runCoursePkSync(cmd *cobra.Command, args []string) error {
	depth, _ := cmd.Flags().GetInt("depth")
	cookieFlag, _ := cmd.Flags().GetString("onesystem-cookie")
	snoFlag, _ := cmd.Flags().GetString("onesystem-sno")
	passwordFlag, _ := cmd.Flags().GetString("onesystem-password")
	explicitID, _ := cmd.Flags().GetUint64("calendar-id")
	materialize, _ := cmd.Flags().GetBool("materialize")

	calendarId, err := resolvePkCalendarId(args[0], explicitID)
	if err != nil {
		return err
	}
	cookie, err := pkservice.ResolveCookie(cookieFlag, snoFlag, passwordFlag)
	if err != nil {
		return err
	}

	report, err := pkservice.Sync(cmd.Context(), cookie, calendarId, depth, materialize)
	if err != nil {
		return fmt.Errorf("course-pk-sync: %w", err)
	}

	fmt.Printf("course-pk-sync: calendars=%v depth=%d materialize=%v\n", report.CalendarIDs, depth, materialize)
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
	if explicitID != 0 {
		return explicitID, nil
	}
	arg = strings.TrimSpace(arg)
	if id, err := strconv.ParseUint(arg, 10, 64); err == nil && id > 0 {
		return id, nil
	}
	if id, ok := pk.GetCalendarIdByI18n(arg); ok {
		return id, nil
	}
	return 0, fmt.Errorf("无法解析学期 %q 对应的一系统 calendarId：请先以 --calendar-id 同步一次，或直接传数字 calendarId（如 121）", arg)
}
