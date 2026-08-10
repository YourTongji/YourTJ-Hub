package courseservice

import (
	"strings"
	"unicode"
)

// Normalize 对课程名/别名/教师名做搜索归一化：
// 全角转半角、统一小写、去除空白与标点（保留字母/数字/汉字）。
// 用于 normalized_name / normalized_value 等列，避免 LIKE 查询被大小写与全半角干扰。
func Normalize(input string) string {
	var b strings.Builder
	for _, r := range input {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E: // 全角 → 半角（随后统一小写）
			b.WriteRune(unicode.ToLower(r - 0xFEE0))
		case r == 0x3000: // 全角空格
			b.WriteRune(' ')
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return strings.TrimSpace(b.String())
}
