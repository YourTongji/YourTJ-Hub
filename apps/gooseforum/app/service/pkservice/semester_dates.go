package pkservice

import (
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/spf13/cast"
)

// 学期起止日期配置（[pk.semester_dates]）：一系统 manualArrange 响应不含学期日期
// （已核实），故由部署 config 维护，course-pk-sync 写入 pk_calendar 时按
// calendar_id_i18n 命中填充 start_date/end_date。未配置的学期两列为 NULL。
//
// 配置形态（键 = calendar_id_i18n，值为 start/end 纯日期）：
//
//	[pk.semester_dates."2025-2026-1"]
//	start = "2025-09-08"
//	end = "2026-01-18"
const semesterDateLayout = "2006-01-02"

// semesterDateRange 单个学期的起止日期（任一端可缺，非法配置静默忽略）。
type semesterDateRange struct {
	Start *time.Time
	End   *time.Time
}

// parseSemesterDate 解析单个日期串；空白/格式非法返回 nil（不阻断同步）。
func parseSemesterDate(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	t, err := time.Parse(semesterDateLayout, trimmed)
	if err != nil {
		return nil
	}
	return &t
}

// parseSemesterDates 解析 [pk.semester_dates] 原始值为 {i18n: {start, end}} 映射。
// 兼容 viper 展开的 map[string]any（内层 map[string]any / map[string]string）；
// 非法结构或非法日期静默跳过对应端点（fail-open：日期是展示增强，不值得让同步失败）。
func parseSemesterDates(raw any) map[string]semesterDateRange {
	result := map[string]semesterDateRange{}
	outer, ok := raw.(map[string]any)
	if !ok {
		return result
	}
	for key, value := range outer {
		i18n := strings.TrimSpace(key)
		if i18n == "" {
			continue
		}
		var startStr, endStr string
		switch inner := value.(type) {
		case map[string]any:
			startStr = cast.ToString(inner["start"])
			endStr = cast.ToString(inner["end"])
		case map[string]string:
			startStr = inner["start"]
			endStr = inner["end"]
		default:
			continue
		}
		parsed := semesterDateRange{
			Start: parseSemesterDate(startStr),
			End:   parseSemesterDate(endStr),
		}
		if parsed.Start == nil && parsed.End == nil {
			continue
		}
		result[i18n] = parsed
	}
	return result
}

// loadSemesterDates 读取当前配置的学期日期表（viper 内存查找，无 IO）。
// 包级 var 便于测试注入，避免污染全局 viper 状态。
var loadSemesterDates = func() map[string]semesterDateRange {
	return parseSemesterDates(preferences.GetRaw("pk.semester_dates"))
}
