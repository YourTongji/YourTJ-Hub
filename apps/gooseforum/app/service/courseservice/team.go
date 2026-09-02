package courseservice

import (
	"sort"
	"strings"
)

// TeacherSet 一门课程跨学期的教师集合（已去重）。
// TeamKey 非空表示该课程已归入某教学团队；双方 TeamKey 相同的课程对不再产出团队候选。
type TeacherSet struct {
	CourseID uint64
	TeamKey  string
	Teachers []string
}

// TeamCandidate 团队候选：两门课程教师集合的 Jaccard 相似度达到阈值。
type TeamCandidate struct {
	CourseA uint64  `json:"courseA"`
	CourseB uint64  `json:"courseB"`
	Jaccard float64 `json:"jaccard"`
}

// MergeTeachers 合并多学期教师名单：去重（忽略空白项），保留首次出现顺序。
func MergeTeachers(lists ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, list := range lists {
		for _, name := range list {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

// JaccardSimilarity 计算两个教师集合的 Jaccard 相似度：|A∩B| / |A∪B|。
// 按集合语义（去重、忽略空白）；任一集合为空返回 0（无重叠证据，不成团队）。
func JaccardSimilarity(a, b []string) float64 {
	setA := toNameSet(a)
	setB := toNameSet(b)
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}
	intersection := 0
	for name := range setA {
		if _, ok := setB[name]; ok {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	return float64(intersection) / float64(union)
}

// TeamCandidates 判定团队候选：两两配对（i<j 只保留一次），Jaccard >= threshold 且 > 0
// 产出候选；双方已归组（TeamKey 非空且相同）的课程对跳过。
// 返回按 Jaccard 降序（同分按 CourseA/CourseB 升序）稳定排序。
func TeamCandidates(sets []TeacherSet, threshold float64) []TeamCandidate {
	var candidates []TeamCandidate
	for i := 0; i < len(sets); i++ {
		for j := i + 1; j < len(sets); j++ {
			a, b := sets[i], sets[j]
			if a.TeamKey != "" && a.TeamKey == b.TeamKey {
				continue
			}
			jaccard := JaccardSimilarity(a.Teachers, b.Teachers)
			if jaccard <= 0 || jaccard < threshold {
				continue
			}
			candidates = append(candidates, TeamCandidate{CourseA: a.CourseID, CourseB: b.CourseID, Jaccard: jaccard})
		}
	}
	sort.SliceStable(candidates, func(x, y int) bool {
		if candidates[x].Jaccard != candidates[y].Jaccard {
			return candidates[x].Jaccard > candidates[y].Jaccard
		}
		if candidates[x].CourseA != candidates[y].CourseA {
			return candidates[x].CourseA < candidates[y].CourseA
		}
		return candidates[x].CourseB < candidates[y].CourseB
	})
	return candidates
}

// toNameSet 教师名单转集合：去重、忽略空白项。
func toNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}
