package lineage

// EvaluateAll 对一批课程（跨学期）两两配对并运行 R1-R5，返回全部候选。
// 配对剪枝：R1/R2/R3 需要同 teacherCode，R4/R5 需要同 familyKey（或同教师同码），
// 因此只对「同 teacherCode」或「同 familyKey」的课程对跑规则；同一对 (from, to)
// 只评估一次，学期可解析时按学期排序保证候选方向为 旧 → 新（输入顺序不受约束，
// CLI 的 calendarId 顺序未定义）；无法解析学期时保持输入顺序兜底。
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
	evaluatePair := func(i, j int) {
		from, to := courses[i], courses[j]
		// 方向归一：学期可解析且 from 晚于 to 时交换（候选恒为旧 → 新），
		// 不依赖调用方/输入顺序（review P2：CLI calendarId 顺序不受约束，
		// 倒序输入不得丢弃合法配对）。
		if fromIdx, ok := semesterIndex(from.Semester); ok {
			if toIdx, ok2 := semesterIndex(to.Semester); ok2 && fromIdx > toIdx {
				from, to = to, from
			}
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
