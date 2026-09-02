package courseservice

import "testing"

// TestJaccardSimilarity Jaccard 集合语义：全同 1.0、半重叠 0.5、无重叠 0、空集合 0。
func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want float64
	}{
		{name: "identical", a: []string{"张三", "李娜"}, b: []string{"李娜", "张三"}, want: 1},
		{name: "half overlap", a: []string{"张三"}, b: []string{"张三", "李娜"}, want: 0.5},
		{name: "no overlap", a: []string{"张三"}, b: []string{"李娜"}, want: 0},
		{name: "empty a", a: nil, b: []string{"张三"}, want: 0},
		{name: "empty b", a: []string{"张三"}, b: nil, want: 0},
		{name: "both empty", a: nil, b: nil, want: 0},
		{name: "duplicates dedup", a: []string{"张三", "张三"}, b: []string{"张三"}, want: 1},
		{name: "blank ignored", a: []string{"", "张三"}, b: []string{"张三"}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JaccardSimilarity(tt.a, tt.b); got != tt.want {
				t.Errorf("JaccardSimilarity(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestMergeTeachers 跨学期教师名单合并：去重、保留首次出现顺序、忽略空白。
func TestMergeTeachers(t *testing.T) {
	got := MergeTeachers([]string{"张三", "李娜"}, []string{" 李娜 ", "", "王强"}, nil)
	want := []string{"张三", "李娜", "王强"}
	if len(got) != len(want) {
		t.Fatalf("MergeTeachers = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MergeTeachers = %v, want %v", got, want)
		}
	}
}

// TestTeamCandidatesThresholdBoundary 阈值边界：恰好等于阈值产出候选，低于阈值不产出。
func TestTeamCandidatesThresholdBoundary(t *testing.T) {
	sets := []TeacherSet{
		{CourseID: 1, Teachers: []string{"张三", "李娜", "王强"}},
		{CourseID: 2, Teachers: []string{"张三", "李娜", "赵敏"}},
	}
	// 交集 2 / 并集 4 = 0.5。
	atThreshold := TeamCandidates(sets, 0.5)
	if len(atThreshold) != 1 || atThreshold[0].Jaccard != 0.5 {
		t.Fatalf("threshold 0.5 candidates = %+v, want single 0.5", atThreshold)
	}
	aboveThreshold := TeamCandidates(sets, 0.49)
	if len(aboveThreshold) != 1 {
		t.Fatalf("threshold 0.49 candidates = %+v, want 1", aboveThreshold)
	}
	belowThreshold := TeamCandidates(sets, 0.51)
	if len(belowThreshold) != 0 {
		t.Fatalf("threshold 0.51 candidates = %+v, want 0", belowThreshold)
	}
}

// TestTeamCandidatesSameTeamKeySkipped 双方已归入同一 team_key 的课程对不再产出团队候选。
func TestTeamCandidatesSameTeamKeySkipped(t *testing.T) {
	sets := []TeacherSet{
		{CourseID: 1, TeamKey: "team-higher-math", Teachers: []string{"张三", "李娜"}},
		{CourseID: 2, TeamKey: "team-higher-math", Teachers: []string{"张三", "李娜"}},
		{CourseID: 3, TeamKey: "", Teachers: []string{"张三", "李娜"}},
	}
	candidates := TeamCandidates(sets, 0.5)
	// (1,2) 同 team_key 跳过；(1,3)/(2,3) 未归组、相似度 1.0 均产出。
	if len(candidates) != 2 {
		t.Fatalf("candidates = %+v, want 2（同 team_key 对跳过）", candidates)
	}
}

// TestTeamCandidatesEmptySets 空教师集合不产出候选。
func TestTeamCandidatesEmptySets(t *testing.T) {
	sets := []TeacherSet{
		{CourseID: 1, Teachers: nil},
		{CourseID: 2, Teachers: []string{"张三"}},
	}
	if candidates := TeamCandidates(sets, 0.1); len(candidates) != 0 {
		t.Fatalf("empty-set candidates = %+v, want 0", candidates)
	}
}

// TestTeamCandidatesSorting 候选按 Jaccard 降序、同分按 CourseA/CourseB 升序稳定排序。
func TestTeamCandidatesSorting(t *testing.T) {
	sets := []TeacherSet{
		{CourseID: 1, Teachers: []string{"张三"}},
		{CourseID: 2, Teachers: []string{"张三"}},
		{CourseID: 3, Teachers: []string{"张三", "李娜"}},
		{CourseID: 4, Teachers: []string{"王强"}},
	}
	candidates := TeamCandidates(sets, 0.2)
	// (1,2) 与 (1,3)/(2,3) 相似度 1.0 与 0.5；无重叠 (1,4) 等不产出。
	if len(candidates) != 3 {
		t.Fatalf("candidates = %+v, want 3", candidates)
	}
	if candidates[0].Jaccard != 1 || candidates[0].CourseA != 1 || candidates[0].CourseB != 2 {
		t.Fatalf("top candidate = %+v, want (1,2) jaccard 1", candidates[0])
	}
	for i := 1; i < len(candidates); i++ {
		if candidates[i].Jaccard > candidates[i-1].Jaccard {
			t.Fatalf("candidates not sorted desc: %+v", candidates)
		}
	}
}
