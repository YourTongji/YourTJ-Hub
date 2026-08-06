package searchservice

import "testing"

func TestPinyinFields(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantFull    string
		wantInitial string
	}{
		{name: "pure chinese", input: "校园生活", wantFull: "xiaoyuanshenghuo", wantInitial: "XYSH"},
		{name: "mixed chinese english", input: "GoLang 学习", wantFull: "golangxuexi", wantInitial: "GXX"},
		{name: "punctuation skipped", input: "你好，世界", wantFull: "nihaoshijie", wantInitial: "NHSJ"},
		{name: "digits", input: "123abc", wantFull: "123abc", wantInitial: "1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			full, initials := PinyinFields(tc.input)
			if full != tc.wantFull {
				t.Fatalf("PinyinFields(%q) full = %q, want %q", tc.input, full, tc.wantFull)
			}
			if initials != tc.wantInitial {
				t.Fatalf("PinyinFields(%q) initials = %q, want %q", tc.input, initials, tc.wantInitial)
			}
		})
	}
}

func TestUserPinyinFields(t *testing.T) {
	usernamePinyin, usernameInitials, nicknamePinyin, nicknameInitials := UserPinyinFields("zhangsan", "张三")
	if usernamePinyin != "zhangsan" {
		t.Fatalf("usernamePinyin = %q, want zhangsan", usernamePinyin)
	}
	if usernameInitials != "Z" {
		t.Fatalf("usernameInitials = %q, want Z", usernameInitials)
	}
	if nicknamePinyin != "zhangsan" {
		t.Fatalf("nicknamePinyin = %q, want zhangsan", nicknamePinyin)
	}
	if nicknameInitials != "ZS" {
		t.Fatalf("nicknameInitials = %q, want ZS", nicknameInitials)
	}
}

func TestCategoryPinyinFields(t *testing.T) {
	namePinyin, nameInitials := CategoryPinyinFields("校园生活")
	if namePinyin != "xiaoyuanshenghuo" {
		t.Fatalf("namePinyin = %q, want xiaoyuanshenghuo", namePinyin)
	}
	if nameInitials != "XYSH" {
		t.Fatalf("nameInitials = %q, want XYSH", nameInitials)
	}
}
