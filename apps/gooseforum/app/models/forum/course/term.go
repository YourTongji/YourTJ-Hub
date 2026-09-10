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

// canonicalTermCodePattern 课程域标准学期码形：YYYY-YYYY-N（N 为 1~2 位数字）。
var canonicalTermCodePattern = regexp.MustCompile(`^\d{4}-\d{4}-\d{1,2}$`)

// IsCanonicalTermCode 判断学期码是否为课程域标准形（如 "2026-2027-1"）。
// 物化链用它决定是否允许 getOrCreateTermTx 建行：非标准形（如无法识别的中文
// 短学期标记）一律不建 course_term 行、保持 term_id=0，杜绝垃圾学期行（review LOW）。
func IsCanonicalTermCode(code string) bool {
	return canonicalTermCodePattern.MatchString(code)
}

// TermLabelCandidates 返回学期标记匹配 course_term.code 的候选列表：trim 后的原标记 +
// 规范化后的标准码（不同则追加，去重）。排课器 course-review-brief 的 term 反查与
// 物化共用同一权威解析（review Should：双份实现会让两条链路后续分歧）；调用方按序
// 尝试每个候选——先精确后规范化，兼容历史把中文学期名直接写进 code 的行。
func TermLabelCandidates(label string) []string {
	label = strings.TrimSpace(label)
	if label == "" {
		return nil
	}
	candidates := []string{label}
	if normalized := NormalizeTermLabel(label); normalized != label {
		candidates = append(candidates, normalized)
	}
	return candidates
}

// cnNumeralToArabic 把中文数字学期序数转阿拉伯数字（"二"→"2"）；已是阿拉伯数字原样
// 返回。支持学期序数实际范围（一~十二）：单字（"二"）与十字组合（"十"→"10"、
// "十一"→"11"、"二十"→"20"、"二十三"→"23"）。
func cnNumeralToArabic(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= '0' && s[0] <= '9' {
		return s
	}
	runes := []rune(s)
	switch len(runes) {
	case 1:
		if n, ok := cnDigits[runes[0]]; ok {
			return strconv.Itoa(n)
		}
		if runes[0] == '十' {
			return "10"
		}
	case 2:
		if runes[0] == '十' {
			// 十一 ~ 十九
			if n, ok := cnDigits[runes[1]]; ok {
				return strconv.Itoa(10 + n)
			}
		} else if runes[1] == '十' {
			// 二十 ~ 九十
			if n, ok := cnDigits[runes[0]]; ok {
				return strconv.Itoa(10 * n)
			}
		}
	case 3:
		// 二十三 等「数十+个位」组合
		if tens, ok := cnDigits[runes[0]]; ok && runes[1] == '十' {
			if ones, ok2 := cnDigits[runes[2]]; ok2 {
				return strconv.Itoa(10*tens + ones)
			}
		}
	}
	return s
}

// cnDigits 中文数字单字 → 阿拉伯数字。
var cnDigits = map[rune]int{
	'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
	'六': 6, '七': 7, '八': 8, '九': 9,
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
