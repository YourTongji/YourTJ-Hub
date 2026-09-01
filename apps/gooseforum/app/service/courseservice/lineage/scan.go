package lineage

// EvaluateAll 对一批课程（跨学期）两两配对并运行 R1-R5，返回全部候选。
// 配对剪枝：R1/R2/R3 需要同 teacherCode，R4/R5 需要同 familyKey（或同教师同码），
// 因此只对「同 teacherCode」或「同 familyKey」的课程对跑规则；同一对 (from, to)
// 只评估一次，且限定 from 的学期不晚于 to（无法解析学期时不限制，以输入顺序兜底）。
// 纯内存计算，不读写数据库。
func EvaluateAll(courses []CourseSummary) []LineageCandidate {
	teacherGroups := map[string][]int{}
	familyGroups := map[string][]int{}
	for i, c := range courses {
		if c.TeacherCode != "" {
			teacherGroups[c.TeacherCode] = append(teacherGroups[c.TeacherCode], i)
		}
		if f := FamilyKey(c.Name); f != "" {
			familyGroups[f] = append(familyGroups[f], i)
		}
	}

	var candidates []LineageCandidate
	seen := map[int]map[int]bool{}
	markSeen := func(i, j int) bool {
		if seen[i] == nil {
			seen[i] = map[int]bool{}
		}
		if seen[i][j] {
			return false
		}
		seen[i][j] = true
		return true
	}
	evaluatePair := func(i, j int) {
		from, to := courses[i], courses[j]
		if !orderedBySemester(from, to) || !markSeen(i, j) {
			return
		}
		candidates = append(candidates, Evaluate(from, to)...)
	}

	visited := map[int]map[int]bool{}
	visitGroup := func(group []int) {
		for a := 0; a < len(group); a++ {
			for b := a + 1; b < len(group); b++ {
				i, j := group[a], group[b]
				if j < i {
					i, j = j, i
				}
				if visited[i] == nil {
					visited[i] = map[int]bool{}
				}
				if visited[i][j] {
					continue
				}
				visited[i][j] = true
				evaluatePair(i, j)
			}
		}
	}
	for _, group := range teacherGroups {
		visitGroup(group)
	}
	for _, group := range familyGroups {
		visitGroup(group)
	}
	return candidates
}

// orderedBySemester from 的学期不晚于 to：双方都能解析时比较绝对学期序号，
// 任一侧无法解析时按输入顺序放行（调用方保证 i < j）。
func orderedBySemester(from, to CourseSummary) bool {
	fromIdx, fromOK := semesterIndex(from.Semester)
	toIdx, toOK := semesterIndex(to.Semester)
	if !fromOK || !toOK {
		return true
	}
	return fromIdx <= toIdx
}
