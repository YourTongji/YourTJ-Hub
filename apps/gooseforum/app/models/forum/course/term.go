package course

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// termLabelPattern 匹配一系统的学期标记并提取学年与学期序数：
// "2025-2026-2" / "2025-2026学年第2学期" / "2025-2026 第二学期"。
// 组 1 = 学年（YYYY-YYYY），组 2 = 学期序数（阿拉伯或中文数字）。
var termLabelPattern = regexp.MustCompile(`(\d{4}-\d{4})[^0-9一二三四五六七八九十]*([0-9一二三四五六七八九十]+)[^0-9一二三四五六七八九十]*$`)

// NormalizeTermLabel 将一系统学期标记规范化为课程域标准码 "YYYY-YYYY-N"：
//   - "2025-2026-1"（已是标准码）原样返回；
//   - "2026-2027学年第1学期" / "2025-2026 第二学期" → "2026-2027-1" / "2025-2026-2"；
//   - 无法识别（如 "其他"）返回 trim 后的原标记。
//
// 物化链（course-pk-sync --materialize / course-materialize）用标准码创建/匹配
// course_term：一系统侧写入的是中文学期名，直接落库会把 "2026-2027学年第1学期"
// 当成学期码创建垃圾行，导致存量 offering 的 term_id 被改写、目录学期筛选断裂。
func NormalizeTermLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	if m := termLabelPattern.FindStringSubmatch(label); len(m) == 3 {
		return m[1] + "-" + cnNumeralToArabic(m[2])
	}
	return label
}

// cnNumeralToArabic 把中文数字学期序数转阿拉伯数字（"二"→"2"）；已是阿拉伯数字原样返回。
func cnNumeralToArabic(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= '0' && s[0] <= '9' {
		return s
	}
	if n, ok := map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9, '十': 10}[[]rune(s)[0]]; ok {
		return strconv.Itoa(n)
	}
	return s
}

const termTableName = "course_term"

type TermEntity struct {
	Id        uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Code      string         `gorm:"column:code;type:varchar(32);not null;default:'';uniqueIndex:uniq_course_term_code;" json:"code"`
	Name      string         `gorm:"column:name;type:varchar(64);not null;default:'';" json:"name"`
	StartsOn  *time.Time     `gorm:"column:starts_on;type:date;" json:"startsOn"`
	EndsOn    *time.Time     `gorm:"column:ends_on;type:date;" json:"endsOn"`
	Status    int8           `gorm:"column:status;not null;default:0;" json:"status"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

func (itself *TermEntity) TableName() string {
	return termTableName
}
