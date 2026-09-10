// Package lineage 课程沿革规则引擎（Blueprint「课评聚合与 2026 课程沿革」Phase 4）。
// 纯函数、无副作用：输入课程的结构化摘要，输出候选沿革关系，不读写数据库。
// 用于 2026 年课程改制后（courseCode 重编）将新老课程对齐，产出 EQUIVALENT /
// RENAMED_FROM / SPLIT_FROM / RELATED 候选，供后续人工确认与写入。
package lineage

import (
	"slices"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// 硬语义 token：课程名尾部的变体标记，携带课程难度/层次/修读形式语义。
// 顺序即优先级：A1/A2 先于 A（避免 A1 被 A 吞掉），「上/下」先于「基础/进阶」。
// 「基础/进阶」同时是尾部独立词（高级语言程序设计进阶）与括号组合词（（进阶）），
// 前者由 tailVariantTokens 剥离，后者由 variantTokens 在括号内识别。
var variantTokens = []string{
	"课程设计", "实习",
	"实验", "实践",
	"上", "下",
	"基础", "进阶",
	"荣", "卓", "英",
	"A1", "A2", "B", "C", "D", "A",
	"I", "II", "III", "IV", "V",
}

// tailVariantTokens 是仅出现在名称尾部（含括号）的硬语义 token；剥离后即课程家族名。
var tailVariantTokens = []string{
	"课程设计", "实习",
	"实验", "实践",
	"上", "下",
	"基础", "进阶",
	"荣", "卓", "英",
	"A1", "A2", "B", "C", "D", "A",
	"I", "II", "III", "IV", "V",
}

// NormalizeCourseName 对课程名做沿革匹配归一化：
// Unicode NFKC（全角/半角、兼容字符统一）、空白归一、英文小写、标点轻量清洗
// （括号统一为半角 ()、去除尾部句点），但保留硬语义 token 的语义与间距。
//
// 与 courseservice.Normalize（搜索用，连标点与空格一并剥掉）不同：沿革匹配必须
// 保留 A1/A2/B、上/下、基础/进阶、实验/实践/课程设计/实习 等 token 的边界，
// 否则「高等数学A」与「高等数学A1」会被错误合并。间距统一为单个空格。
func NormalizeCourseName(name string) string {
	s := norm.NFKC.String(name)
	s = strings.Map(func(r rune) rune {
		switch r {
		case '（', '）':
			return '('
		case '　', '\t', '\n', '\r':
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	s = strings.Join(fields, " ")
	s = strings.ToLower(s)
	s = strings.TrimRight(s, ".。·")
	return s
}

// FamilyKey 提取课程家族名：去掉尾部的变体 token（含括号组合），如
// 「高等数学A(I)」→「高等数学」、「高级语言程序设计A1」→「高级语言程序设计」。
// 返回经 NormalizeCourseName 归一的家族名；仅由变体 token 组成的输入返回空串。
func FamilyKey(name string) string {
	s := NormalizeCourseName(name)
	// 先剥「（token）」括号组合（家族名不带括号变体）。
	for {
		stripped := stripBracketVariant(s)
		if stripped == s {
			break
		}
		s = stripped
	}
	// 再剥尾部裸 token（上/下 在前，避免「上下」双字组合被拆错序）。
	for _, tok := range tailVariantTokens {
		lower := strings.ToLower(tok)
		for {
			trimmed := strings.TrimSuffix(s, lower)
			if trimmed == s {
				break
			}
			s = strings.TrimRight(trimmed, " ")
		}
	}
	return strings.TrimSpace(s)
}

// stripBracketVariant 去掉字符串末尾形如「(token)」的括号变体组合（可带尾随空格），
// 无匹配时原样返回。
func stripBracketVariant(s string) string {
	for {
		open := strings.LastIndex(s, "(")
		if open < 0 {
			return s
		}
		inner := s[open+1:]
		closeIdx := strings.Index(inner, ")")
		if closeIdx < 0 {
			return s
		}
		content := strings.TrimSpace(inner[:closeIdx])
		matched := false
		for _, tok := range variantTokens {
			if strings.EqualFold(content, tok) {
				matched = true
				break
			}
		}
		if !matched {
			return s
		}
		rest := strings.TrimRight(s[:open], " ")
		if rest == "" {
			return s
		}
		s = rest
	}
}

// VariantKey 提取结构化变体：归一名称去掉家族名后的剩余部分，
// 如「高等数学A(I)」→「a(i)」、「高等数学（上）A1」→「a1 (上)」。
// 家族名为空（名称全是变体 token）时原样返回归一名称；无变体返回空串。
// 变体保留硬语义：A1≠A2≠B、基础≠进阶、实验≠理论、I/II/III、上/下 均不合并。
func VariantKey(name string) string {
	s := NormalizeCourseName(name)
	family := FamilyKey(s)
	if family == "" {
		return s
	}
	rest := strings.TrimPrefix(s, family)
	rest = strings.TrimSpace(rest)
	// 家族剥离可能残留半括号（如「高等数学（上）」中「（」在 family 后），
	// 统一括号形态为半角并清理空白，保证 VariantKey 稳定可比较。
	rest = strings.Map(func(r rune) rune {
		switch r {
		case '（':
			return '('
		case '）':
			return ')'
		case '　':
			return ' '
		}
		return r
	}, rest)
	return strings.Join(strings.Fields(rest), " ")
}

// isHardSemanticVariant 判断两个变体是否构成硬语义分隔（R5）：
// A1/A2/B/C/D 字母档互斥、I/II/III 层级互斥、上/下 互斥、基础/进阶 互斥、
// 实验/实践 与 理论（无变体）互斥、课程设计/实习 与普通课堂互斥。
// 两变体相同（含均空）不算冲突；空变体（理论）与实验/课程设计/实习 冲突。
// 变体含未知片段（无法完整解析为已知 token）时保守判定为不冲突。
func isHardSemanticVariant(a, b string) bool {
	na, nb := NormalizeCourseName(a), NormalizeCourseName(b)
	if na == nb {
		return false
	}
	if na == "" || nb == "" {
		return isPracticeVariant(na) || isPracticeVariant(nb)
	}
	ta, tb := parseVariantTokens(na), parseVariantTokens(nb)
	if ta == nil || tb == nil {
		return false
	}
	return !slices.Equal(ta, tb)
}

// knownVariantTokens 全部已知硬语义 token（小写归一形态）：变体解析与硬分隔判定共用。
var knownVariantTokens = map[string]bool{
	"a1": true, "a2": true, "a": true, "b": true, "c": true, "d": true,
	"i": true, "ii": true, "iii": true, "iv": true, "v": true,
	"上": true, "下": true,
	"基础": true, "进阶": true,
	"实验": true, "实践": true, "课程设计": true, "实习": true,
	"荣": true, "卓": true, "英": true,
}

// parseVariantTokens 把变体串拆解为已知 token 序列：按空白与括号切分，
// 每个子 token 必须精确命中 knownVariantTokens；含未知片段返回 nil。
func parseVariantTokens(s string) []string {
	s = strings.Map(func(r rune) rune {
		if r == '(' || r == ')' {
			return ' '
		}
		return r
	}, s)
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ToLower(f)
		if !knownVariantTokens[f] {
			return nil
		}
		tokens = append(tokens, f)
	}
	return tokens
}

// isPracticeVariant 变体是否纯由实践/项目类 token（实验/实践/课程设计/实习）组成，
// 与空变体（理论课堂）构成硬分隔。
func isPracticeVariant(s string) bool {
	tokens := parseVariantTokens(s)
	if len(tokens) == 0 {
		return false
	}
	for _, tok := range tokens {
		switch tok {
		case "实验", "实践", "课程设计", "实习":
		default:
			return false
		}
	}
	return true
}
