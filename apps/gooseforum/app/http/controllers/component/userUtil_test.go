package component

import (
	"strings"
	"testing"
)

// PR #552 review（P2）：契约承诺密码长度按「字符」计（6-64 characters），
// ValidatePassword 必须按 Unicode 字符（rune）计数而不是 UTF-8 字节，
// 否则中文密码的最小长度被字节计数稀释、最长限制误拒纯中文长密码。
func TestValidatePasswordCountsCharactersNotBytes(t *testing.T) {
	// 4 个字符（6 个 UTF-8 字节）：按字节计数会被误收，按字符计数必须拒绝。
	if err := ValidatePassword("密码a1", 6); err == nil {
		t.Fatal("4-character password accepted under 6-character minimum (byte-count regression)")
	}

	// 64 个字符（49 汉字 + 14 字母 + 数字）：按字符计合法；
	// 按字节计数（162 字节）会被误拒为超长。
	long := strings.Repeat("密", 49) + strings.Repeat("a", 14) + "1"
	if err := ValidatePassword(long, 6); err != nil {
		t.Fatalf("64-character password rejected: %v", err)
	}

	// 65 个字符：按字符计数必须超长拒绝。
	tooLong := strings.Repeat("密", 50) + strings.Repeat("a", 14) + "1"
	if err := ValidatePassword(tooLong, 6); err == nil {
		t.Fatal("65-character password accepted under 64-character maximum")
	}
}
