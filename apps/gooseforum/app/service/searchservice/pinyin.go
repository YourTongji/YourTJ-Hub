package searchservice

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mozillazg/go-pinyin"
)

// PinyinFields 生成输入文本的全拼（无音调）与首字母。
// 行为：
//   - 汉字：逐字拼音连续拼接（"校园生活" -> "xiaoyuanshenghuo"）
//   - 连续英文字母/数字：作为一段保留（小写）（"zhangsan" -> "zhangsan"）
//   - 空白与标点：跳过（不作为拼音内容，仅作分段边界）
//
// 全拼用于拼音搜索（如搜 "xiaoyuan" 命中"校园生活"），
// 首字母用于缩写搜索（如搜 "xy" 或 "XYSH" 命中"校园生活"）。
func PinyinFields(input string) (full, initials string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", ""
	}
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal

	var fullBuilder, initialsBuilder strings.Builder
	var nonHanBuilder strings.Builder

	flushNonHan := func() {
		if nonHanBuilder.Len() == 0 {
			return
		}
		seg := nonHanBuilder.String()
		fullBuilder.WriteString(strings.ToLower(seg))
		r, _ := utf8.DecodeRuneInString(seg)
		if isAsciiLetter(r) {
			initialsBuilder.WriteString(strings.ToUpper(string(r)))
		} else {
			initialsBuilder.WriteRune(r)
		}
		nonHanBuilder.Reset()
	}

	for _, r := range input {
		switch {
		case unicode.Is(unicode.Han, r):
			flushNonHan()
			py := pinyin.LazyPinyin(string(r), args)
			if len(py) > 0 && py[0] != "" {
				fullBuilder.WriteString(py[0])
				first, _ := utf8.DecodeRuneInString(py[0])
				initialsBuilder.WriteString(strings.ToUpper(string(first)))
			}
		case unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			// 空白/标点作为分段边界
			flushNonHan()
		default:
			nonHanBuilder.WriteRune(r)
		}
	}
	flushNonHan()

	return fullBuilder.String(), initialsBuilder.String()
}

// UserPinyinFields 生成用户可搜字段的拼音辅助字段（username + nickname，含首字母）。
func UserPinyinFields(username, nickname string) (usernamePinyin, usernameInitials, nicknamePinyin, nicknameInitials string) {
	usernamePinyin, usernameInitials = PinyinFields(username)
	nicknamePinyin, nicknameInitials = PinyinFields(nickname)
	return usernamePinyin, usernameInitials, nicknamePinyin, nicknameInitials
}

// CategoryPinyinFields 生成分类名的拼音辅助字段。
func CategoryPinyinFields(name string) (namePinyin, nameInitials string) {
	return PinyinFields(name)
}

func isAsciiLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
