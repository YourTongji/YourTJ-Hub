package lineage

import (
	"encoding/json"
	"sort"
	"time"
)

// CardSummary 卡级沿革候选输入：课程目录一张 (primary_code, teacher_id) 课程卡。
//
// 与教学班级级 course-lineage-scan（输入 pk_course_detail.id）互补：
// scan 在 PK 域内两两配对，产出的是教学班 id 候选，无法直接落 course_relations
// （其 from/to 为 course.id）；卡级种子（course-lineage-seed）在课程目录内配对，
// 产出的 from/to 即 course_relations 需要的课程卡 id。
type CardSummary struct {
	ID           uint64
	PrimaryCode  string
	Name         string
	TeacherCode  string // 课程卡身份教师工号（course_instructor.teacher_code）
	TeacherName  string
	CreditX10    int
	CreatedAt    time.Time
	PkCourseCode []string // 卡全部可见 offering 经 teaching_class_id 关联的一系统 course_code（去重）
	PkNewCode    []string // 一系统 new_course_code（2026 改制新编码，去重）
	Terms        []string // 开课学期码 YYYY-YYYY-N（去重，证据展示）
}

// CardCandidate 卡级沿革候选：from（冗余/旧卡）→ to（规范/当前卡）。
// RelationType 仅 EQUIVALENT / RENAMED_FROM 可触发合并（管理端 MergeCourses）；
// SPLIT_FROM / RELATED 只表达标注，经管理端 approve 后在详情页沿革区块展示。
type CardCandidate struct {
	FromCardID   uint64
	ToCardID     uint64
	RelationType string
	Source       string
	Confidence   float64
	Evidence     string // JSON 证据快照（教师/课程码/名称/学分/学期）
}

// cardEvidence 卡级候选证据载荷（存 evidence_json，供管理端审核追溯）。
type cardEvidence struct {
	TeacherCode     string   `json:"teacherCode,omitempty"`
	TeacherName     string   `json:"teacherName,omitempty"`
	Rule            string   `json:"rule,omitempty"`
	FromCode        string   `json:"fromCode,omitempty"`
	ToCode          string   `json:"toCode,omitempty"`
	SharedPkCourse  []string `json:"sharedPkCourse,omitempty"`
	SharedPkNewCode []string `json:"sharedPkNewCode,omitempty"`
	NormalizedName  string   `json:"normalizedName,omitempty"`
	CreditX10       int      `json:"creditX10,omitempty"`
	FromVariant     string   `json:"fromVariant,omitempty"`
	ToVariant       string   `json:"toVariant,omitempty"`
	FamilyKey       string   `json:"familyKey,omitempty"`
	FromTerms       []string `json:"fromTerms,omitempty"`
	ToTerms         []string `json:"toTerms,omitempty"`
	FromCreatedAt   string   `json:"fromCreatedAt,omitempty"`
	ToCreatedAt     string   `json:"toCreatedAt,omitempty"`
}

// marshalCardEvidence 序列化卡级证据载荷（折叠错误，与 marshalEvidence 同风格）。
func marshalCardEvidence(ev cardEvidence) string {
	data, err := json.Marshal(ev)
	if err != nil {
		return `{"marshalError":"` + err.Error() + `"}`
	}
	return string(data)
}

// 卡级规则来源标记（evidence.rule，便于审核追溯规则分支）。
const (
	cardRuleEquivalent = "CARD-E1" // 冗余卡并入规范卡（同师 + 同名同学分 + 共享一系统课程码）
	cardRuleSplit      = "CARD-E2" // 家族内变体/层次标注（同师 + 同家族 + 变体不同）
	cardRuleRelated    = "CARD-E3" // 学分巨变弱关联（同师 + 同名 + 学分巨变）
)

// EvaluateCards 在课程卡层面运行卡级沿革规则，返回去重、方向稳定（旧 → 新）的候选。
//
// 配对限制在同教师工号组内（跨教师的同名/同码课程卡是合法分班，不产候选，避免
// 体育/外语等大班多教师噪音）。三类规则：
//
//   - E1（EQUIVALENT，conf 0.9）：同 teacherCode + 归一名称一致 + 学分一致，
//     且开课实例共享至少一个一系统 course_code / new_course_code —— 课程改革码型
//     归一后，旧卡（带教学班后缀码）与规范码卡是同一门课的冗余卡。对每个「同码
//     连通分量」选一张规范卡为目标（primary_code 命中分量共享码优先，其次最新，
//     再次最小 id），其余全部指向它——保证候选无环、无合并链（卡不会既是 A 的
//     from 又是 B 的 target）。
//   - E2（SPLIT_FROM，conf 0.5）：同 teacherCode + 同家族 + 变体不同（A1/A2/B、
//     基础/进阶、上/下、实验/理论，以及 2026 分层中「无变体 generic → A1」这类
//     层次重组）→ 拆分/层次标注，绝不合并（改版前后课程卡互链，详情页沿革区块）。
//   - E3（RELATED，conf 0.2）：同 teacherCode + 同归一名称 + 学分巨变 → 弱关联
//     标注，供人工核查是否改课（学时学分调整、无码型证据的同名卡）。
//
// 纯内存计算，不读写数据库；输出按 (FromCardID, ToCardID, RelationType) 稳定排序。
func EvaluateCards(cards []CardSummary) []CardCandidate {
	teacherGroups := map[string][]int{}
	for i, c := range cards {
		if c.TeacherCode != "" {
			teacherGroups[c.TeacherCode] = append(teacherGroups[c.TeacherCode], i)
		}
	}

	type candidateSet map[string]CardCandidate
	byKey := candidateSet{}
	addCandidate := func(from, to CardSummary, relationType, rule string, conf float64, ev cardEvidence) {
		if from.ID == to.ID || from.ID == 0 || to.ID == 0 {
			return
		}
		key := u64Key(from.ID, to.ID, relationType)
		if _, dup := byKey[key]; dup {
			return
		}
		ev.Rule = rule
		byKey[key] = CardCandidate{
			FromCardID:   from.ID,
			ToCardID:     to.ID,
			RelationType: relationType,
			Source:       "rule",
			Confidence:   conf,
			Evidence:     marshalCardEvidence(ev),
		}
	}

	for _, group := range teacherGroups {
		if len(group) < 2 {
			continue
		}
		groupCards := make([]CardSummary, 0, len(group))
		for _, i := range group {
			groupCards = append(groupCards, cards[i])
		}
		sort.Slice(groupCards, func(i, j int) bool { return groupCards[i].ID < groupCards[j].ID })

		// ---- E1：同码连通分量 → 单目标冗余卡 ----
		// 桶 =（归一名称, 学分）：同师组内名称学分一致的卡才可能互为冗余。
		buckets := map[string][]int{}
		for gi := range groupCards {
			k := NormalizeCourseName(groupCards[gi].Name) + "\x00" + intString(groupCards[gi].CreditX10)
			buckets[k] = append(buckets[k], gi)
		}
		bucketKeys := make([]string, 0, len(buckets))
		for k := range buckets {
			bucketKeys = append(bucketKeys, k)
		}
		sort.Strings(bucketKeys)

		handledByE1 := map[uint64]bool{}
		for _, bk := range bucketKeys {
			members := buckets[bk]
			if len(members) < 2 {
				continue
			}
			// 桶内并查集：两卡共享任一 pk code 即同一冗余分量。
			parent := make([]int, len(groupCards))
			for i := range parent {
				parent[i] = i
			}
			var find func(int) int
			find = func(x int) int {
				if parent[x] != x {
					parent[x] = find(parent[x])
				}
				return parent[x]
			}
			union := func(a, b int) {
				ra, rb := find(a), find(b)
				if ra != rb {
					parent[rb] = ra
				}
			}
			codeOwner := map[string]int{}
			for _, m := range members {
				for _, code := range allCodes(groupCards[m]) {
					if prev, ok := codeOwner[code]; ok {
						union(prev, m)
					} else {
						codeOwner[code] = m
					}
				}
			}
			components := map[int][]int{}
			for _, m := range members {
				r := find(m)
				components[r] = append(components[r], m)
			}
			for _, comp := range components {
				if len(comp) < 2 {
					continue
				}
				// 分量内全部共享码（规范卡匹配用）。
				codeSet := map[string]bool{}
				for _, m := range comp {
					for _, code := range allCodes(groupCards[m]) {
						codeSet[code] = true
					}
				}
				// 选目标：primary_code 命中分量共享码优先；其次最新；再次最小 id。
				target := comp[0]
				for _, m := range comp[1:] {
					if betterE1Target(groupCards[m], groupCards[target], codeSet) {
						target = m
					}
				}
				norm := NormalizeCourseName(groupCards[target].Name)
				to := groupCards[target]
				for _, m := range comp {
					if m == target {
						continue
					}
					from := groupCards[m]
					addCandidate(from, to, RelationEquivalent, cardRuleEquivalent, 0.9,
						cardEvidence{
							TeacherCode:     from.TeacherCode,
							TeacherName:     from.TeacherName,
							FromCode:        from.PrimaryCode,
							ToCode:          to.PrimaryCode,
							SharedPkCourse:  intersectCodes(from.PkCourseCode, to.PkCourseCode),
							SharedPkNewCode: intersectCodes(from.PkNewCode, to.PkNewCode),
							NormalizedName:  norm,
							CreditX10:       to.CreditX10,
							FromTerms:       from.Terms,
							ToTerms:         to.Terms,
							FromCreatedAt:   from.CreatedAt.Format(time.DateOnly),
							ToCreatedAt:     to.CreatedAt.Format(time.DateOnly),
						})
					handledByE1[from.ID] = true
				}
				handledByE1[to.ID] = true
			}
		}

		// ---- E2 / E3：家族内变体配对（E1 已处理的冗余卡跳过，防噪音） ----
		for a := 0; a < len(groupCards); a++ {
			for b := a + 1; b < len(groupCards); b++ {
				ca, cb := groupCards[a], groupCards[b]
				if handledByE1[ca.ID] || handledByE1[cb.ID] {
					continue
				}
				fa, fb := FamilyKey(ca.Name), FamilyKey(cb.Name)
				if fa == "" || fa != fb {
					continue
				}
				na, nb := NormalizeCourseName(ca.Name), NormalizeCourseName(cb.Name)
				from, to := olderFirst(ca, cb)
				if na == nb {
					// 同名同师：E1 无共享码证据时，学分巨变才产 RELATED（弱关联核查）。
					if creditChangedDramaticallyCard(from.CreditX10, to.CreditX10) {
						addCandidate(from, to, RelationRelated, cardRuleRelated, 0.2, pairEvidence(from, to, fa))
					}
					continue
				}
				// 变体不同 → SPLIT_FROM 标注（含 generic→A1 层次重组与 A1/A2/B、
				// 基础/进阶、实验/理论硬分隔；一律不合并，只做沿革互链）。
				addCandidate(from, to, RelationSplitFrom, cardRuleSplit, 0.5, pairEvidence(from, to, fa))
			}
		}
	}

	out := make([]CardCandidate, 0, len(byKey))
	for _, c := range byKey {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromCardID != out[j].FromCardID {
			return out[i].FromCardID < out[j].FromCardID
		}
		if out[i].ToCardID != out[j].ToCardID {
			return out[i].ToCardID < out[j].ToCardID
		}
		return out[i].RelationType < out[j].RelationType
	})
	return out
}

// olderFirst 返回方向化的 from/to（created 旧 → 新；同 created 取 id 小者）。
func olderFirst(ca, cb CardSummary) (CardSummary, CardSummary) {
	if ca.CreatedAt.After(cb.CreatedAt) || (ca.CreatedAt.Equal(cb.CreatedAt) && ca.ID > cb.ID) {
		return cb, ca
	}
	return ca, cb
}

// pairEvidence 组装 from→to 的家族/变体证据（方向化后取值，避免方向交换错位）。
func pairEvidence(from, to CardSummary, family string) cardEvidence {
	return cardEvidence{
		TeacherCode:   from.TeacherCode,
		TeacherName:   from.TeacherName,
		FromCode:      from.PrimaryCode,
		ToCode:        to.PrimaryCode,
		FamilyKey:     family,
		FromVariant:   VariantKey(from.Name),
		ToVariant:     VariantKey(to.Name),
		CreditX10:     from.CreditX10,
		FromTerms:     from.Terms,
		ToTerms:       to.Terms,
		FromCreatedAt: from.CreatedAt.Format(time.DateOnly),
		ToCreatedAt:   to.CreatedAt.Format(time.DateOnly),
	}
}

// allCodes 返回卡的全部一系统码（course_code + new_course_code，去重保序）。
func allCodes(c CardSummary) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range append(append([]string{}, c.PkCourseCode...), c.PkNewCode...) {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// betterE1Target 判断 a 是否比 b 更适合作 E1 合并目标：
// 先比 primary_code 是否命中分量共享码（规范码卡优先），再比 created 新、id 小。
func betterE1Target(a, b CardSummary, componentCodes map[string]bool) bool {
	aHit := componentCodes[a.PrimaryCode]
	bHit := componentCodes[b.PrimaryCode]
	if aHit != bHit {
		return aHit
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	return a.ID < b.ID
}

// intString 非负整数十进制渲染（避免 strconv 导入噪音）。
func intString(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		tmp[i] = '-'
	}
	return string(tmp[i:])
}

// creditChangedDramaticallyCard 学分巨变（R5 硬分隔同风格，credit_x10 整数版）：
// |Δ| ≥ 20（即 2 学分）或翻倍/减半以上。
func creditChangedDramaticallyCard(a, b int) bool {
	delta := a - b
	if delta < 0 {
		delta = -delta
	}
	if delta >= 20 {
		return true
	}
	if a > 0 {
		return b >= 2*a || b*2 <= a
	}
	return false
}

// intersectCodes 返回两个字符串切片共有的元素（去重，保持 a 的顺序）。
func intersectCodes(a, b []string) []string {
	set := map[string]bool{}
	for _, v := range b {
		set[v] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, v := range a {
		if set[v] && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// u64Key 拼一个稳定的字符串键（配对去重/防环用）。
func u64Key(a, b uint64, tag string) string {
	key := make([]byte, 0, 40)
	key = appendUint64(key, a)
	key = append(key, '-')
	key = appendUint64(key, b)
	key = append(key, ':')
	key = append(key, tag...)
	return string(key)
}

// appendUint64 将 uint64 以十进制追加到字节切片（无分配热点路径）。
func appendUint64(b []byte, v uint64) []byte {
	if v == 0 {
		return append(b, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(b, tmp[i:]...)
}
