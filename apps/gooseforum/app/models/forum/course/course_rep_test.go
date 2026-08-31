package course

import (
	"slices"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

// courseRepTestModels 目录筛选测试用到的表。
var courseRepTestModels = []any{
	&Entity{},
	&OfferingEntity{},
	&InstructorEntity{},
	&OfferingInstructorEntity{},
	&CourseStatsEntity{},
	&TermEntity{},
}

// setupCourseRepTest 迁移并清空目录筛选相关表（共享全局连接，与 course 域其它测试一致）。
func setupCourseRepTest(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(courseRepTestModels...); err != nil {
		t.Fatalf("migrate course rep tables: %v", err)
	}
	for _, model := range courseRepTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course rep table: %v", err)
		}
	}
	return conn
}

// createCourse 创建一门可见课程，返回课程 ID。
func createCourse(t *testing.T, conn *gorm.DB, code, department string) uint64 {
	t.Helper()
	c := Entity{PrimaryCode: code, Name: "课程" + code, Department: department, Status: StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	return c.Id
}

// linkCourseInstructor 为课程创建可见 offering 并关联教师（教师记录需已存在）。
func linkCourseInstructor(t *testing.T, conn *gorm.DB, courseId, instructorId uint64) {
	t.Helper()
	offering := OfferingEntity{CourseId: courseId, Status: OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	link := OfferingInstructorEntity{OfferingId: offering.Id, InstructorId: instructorId}
	if err := conn.Create(&link).Error; err != nil {
		t.Fatalf("create offering-instructor link: %v", err)
	}
}

// createTestInstructor 创建教师（覆盖归一化名/拼音/首字母四列，用于验证 LIKE 匹配路径）。
func createTestInstructor(t *testing.T, conn *gorm.DB, name, normalized, pinyin, initials string) uint64 {
	t.Helper()
	ins := InstructorEntity{Name: name, NormalizedName: normalized, NamePinyin: pinyin, NameInitials: initials}
	if err := conn.Create(&ins).Error; err != nil {
		t.Fatalf("create instructor: %v", err)
	}
	return ins.Id
}

// createTerm 创建学期，返回学期 ID。
func createTerm(t *testing.T, conn *gorm.DB, code, name string, startsOn *time.Time) uint64 {
	t.Helper()
	term := TermEntity{Code: code, Name: name, StartsOn: startsOn}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	return term.Id
}

// createOffering 为课程创建指定学期/校区的可见开课实例，返回 offering ID。
func createOffering(t *testing.T, conn *gorm.DB, courseId, termId uint64, campus string) uint64 {
	t.Helper()
	offering := OfferingEntity{CourseId: courseId, TermId: termId, Campus: campus, Status: OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	return offering.Id
}

// parseTermDate 解析 "2006-01-02" 日期（starts_on 列用）。
func parseTermDate(t *testing.T, s string) *time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse term date %q: %v", s, err)
	}
	return &d
}

// setCourseStats 写入课程级评价统计。
func setCourseStats(t *testing.T, conn *gorm.DB, courseId uint64, ratingCount, ratingSum, reviewCount int) {
	t.Helper()
	st := CourseStatsEntity{CourseId: courseId, RatingCount: ratingCount, RatingSum: ratingSum, ReviewCount: reviewCount}
	if err := conn.Create(&st).Error; err != nil {
		t.Fatalf("create course stats: %v", err)
	}
}

// courseIDs 提取实体的 ID 序列，便于断言排序结果。
func courseIDs(entities []Entity) []uint64 {
	ids := make([]uint64, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.Id)
	}
	return ids
}

// TestListCoursesHasReview HasReview 过滤：仅 review_count > 0 的课程进入结果，且 COUNT 同步收窄。
func TestListCoursesHasReview(t *testing.T) {
	conn := setupCourseRepTest(t)
	withReviews := createCourse(t, conn, "100001", "CS")
	zeroReviews := createCourse(t, conn, "100002", "CS")
	_ = createCourse(t, conn, "100003", "CS")     // 无 stats 行，HasReview 同样应排除
	setCourseStats(t, conn, withReviews, 2, 8, 3) // review_count > 0
	setCourseStats(t, conn, zeroReviews, 0, 0, 0) // review_count == 0（有行但零评价）

	t.Run("only_with_reviews", func(t *testing.T) {
		got, total, err := ListCourses(ListCourseQuery{HasReview: true, Size: 50})
		if err != nil {
			t.Fatalf("ListCourses err = %v", err)
		}
		if total != 1 || !slices.Equal(courseIDs(got), []uint64{withReviews}) {
			t.Fatalf("HasReview=true: total=%d ids=%v, want total=1 ids=[%d]", total, courseIDs(got), withReviews)
		}
	})

	t.Run("all_courses", func(t *testing.T) {
		got, total, err := ListCourses(ListCourseQuery{Size: 50})
		if err != nil {
			t.Fatalf("ListCourses err = %v", err)
		}
		if total != 3 || len(got) != 3 {
			t.Fatalf("HasReview unset: total=%d len=%d, want 3/3", total, len(got))
		}
	})
}

// TestListCoursesInstructor Instructor 过滤：%v% LIKE 命中教师四列中的任一列即可，且独立于 keyword。
func TestListCoursesInstructor(t *testing.T) {
	conn := setupCourseRepTest(t)
	zhang := createTestInstructor(t, conn, "张三", "zhangsan", "zhangsan", "zs")
	li := createTestInstructor(t, conn, "李四", "lisi", "lisi", "ls")
	cA := createCourse(t, conn, "100010", "CS")
	cB := createCourse(t, conn, "100011", "CS")
	_ = createCourse(t, conn, "100012", "CS") // 无 offering，不应被教师筛选命中
	linkCourseInstructor(t, conn, cA, zhang)
	linkCourseInstructor(t, conn, cB, li)

	cases := []struct {
		name   string
		input  string
		wantID uint64
	}{
		{"chinese_partial", "张", cA},
		{"chinese_exact", "张三", cA},
		{"normalized_name", "zhangsan", cA},
		{"pinyin_partial", "zhang", cA},
		{"initials", "zs", cA},
		{"other_teacher", "李", cB},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, total, err := ListCourses(ListCourseQuery{Instructor: []string{tc.input}, Size: 50})
			if err != nil {
				t.Fatalf("ListCourses(instructor=%q) err = %v", tc.input, err)
			}
			if total != 1 || !slices.Equal(courseIDs(got), []uint64{tc.wantID}) {
				t.Fatalf("instructor=%q: total=%d ids=%v, want total=1 ids=[%d]", tc.input, total, courseIDs(got), tc.wantID)
			}
		})
	}

	t.Run("no_match", func(t *testing.T) {
		got, total, err := ListCourses(ListCourseQuery{Instructor: []string{"王"}, Size: 50})
		if err != nil {
			t.Fatalf("ListCourses err = %v", err)
		}
		if total != 0 || len(got) != 0 {
			t.Fatalf("instructor=王: total=%d len=%d, want 0/0", total, len(got))
		}
	})
}

// TestListCoursesSortByRating SortBy=rating 按平均分降序，零/无评分课程排末尾（id 倒序兜底）；
// COUNT 与排序无关（LEFT JOIN 不放大计数）。同数据下同时锁定前台默认排序（有评价优先）
// 与管理端排序（id 倒序）两个新分支。
func TestListCoursesSortByRating(t *testing.T) {
	conn := setupCourseRepTest(t)
	// 创建顺序即 id 升序：a(avg4.0) b(avg5.0) c(avg3.0) d(无评分行) e(rating_count=0)。
	a := createCourse(t, conn, "100020", "CS")
	b := createCourse(t, conn, "100021", "CS")
	c := createCourse(t, conn, "100022", "CS")
	d := createCourse(t, conn, "100023", "CS")
	e := createCourse(t, conn, "100024", "CS")
	setCourseStats(t, conn, a, 2, 8, 2)
	setCourseStats(t, conn, b, 1, 5, 1)
	setCourseStats(t, conn, c, 1, 3, 1)
	setCourseStats(t, conn, e, 0, 0, 0)

	got, total, err := ListCourses(ListCourseQuery{SortBy: "rating", Size: 50})
	if err != nil {
		t.Fatalf("ListCourses(sortBy=rating) err = %v", err)
	}
	// 有评分者按平均分降序（b 5.0 > a 4.0 > c 3.0），零评分者排末尾按 id 降序（e > d）。
	want := []uint64{b, a, c, e, d}
	if total != 5 || !slices.Equal(courseIDs(got), want) {
		t.Fatalf("sortBy=rating: total=%d ids=%v, want total=5 ids=%v", total, courseIDs(got), want)
	}

	// 前台默认排序：有评价（review_count > 0）的课程优先——仅排序不筛选，
	// 无评论课程仍排在后面；组内保持 id 倒序。
	// 有评价组：a/b/c（id 倒序 c > b > a）；无评价组：d（无统计行）与 e（review_count=0），id 倒序 e > d。
	gotDefault, totalDefault, err := ListCourses(ListCourseQuery{Size: 50})
	if err != nil {
		t.Fatalf("ListCourses(default) err = %v", err)
	}
	wantDefault := []uint64{c, b, a, e, d}
	if totalDefault != 5 || !slices.Equal(courseIDs(gotDefault), wantDefault) {
		t.Fatalf("default sort: total=%d ids=%v, want total=5 ids=%v", totalDefault, courseIDs(gotDefault), wantDefault)
	}

	// 管理端（IncludeHidden=true）：保持 id 倒序，不受评价影响。
	gotAdmin, totalAdmin, err := ListCourses(ListCourseQuery{Size: 50, IncludeHidden: true})
	if err != nil {
		t.Fatalf("ListCourses(includeHidden) err = %v", err)
	}
	wantAdmin := []uint64{e, d, c, b, a}
	if totalAdmin != 5 || !slices.Equal(courseIDs(gotAdmin), wantAdmin) {
		t.Fatalf("admin sort: total=%d ids=%v, want total=5 ids=%v", totalAdmin, courseIDs(gotAdmin), wantAdmin)
	}
}

// TestListDistinctDepartments 院系列表去重与排序：排除空值、隐藏课程与软删课程。
func TestListDistinctDepartments(t *testing.T) {
	conn := setupCourseRepTest(t)
	_ = createCourse(t, conn, "100030", "Math")
	_ = createCourse(t, conn, "100031", "CS")
	_ = createCourse(t, conn, "100032", "Physics")
	_ = createCourse(t, conn, "100033", "CS") // 重复院系应去重
	_ = createCourse(t, conn, "100034", "")   // 空院系应排除
	hidden := createCourse(t, conn, "100035", "HiddenDept")
	if err := conn.Model(&Entity{}).Where("id = ?", hidden).Update("status", StatusHidden).Error; err != nil {
		t.Fatalf("hide course: %v", err)
	}
	ghost := createCourse(t, conn, "100036", "GhostDept")
	if err := conn.Delete(&Entity{Id: ghost}).Error; err != nil {
		t.Fatalf("soft-delete course: %v", err)
	}

	got, err := ListDistinctDepartments()
	if err != nil {
		t.Fatalf("ListDistinctDepartments err = %v", err)
	}
	want := []string{"CS", "Math", "Physics"}
	if len(got) != len(want) {
		t.Fatalf("ListDistinctDepartments = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListDistinctDepartments = %v, want %v", got, want)
		}
	}
}

// TestListDistinctTerms 学期列表去重与排序：仅可见课程的可见 offering 关联学期，
// 排除隐藏课程/隐藏 offering/软删数据与空 code；按 starts_on 倒序（回退 code 字典序）。
func TestListDistinctTerms(t *testing.T) {
	conn := setupCourseRepTest(t)
	c := createCourse(t, conn, "100080", "CS")
	// t1 2024 秋冬（starts_on 最早）→ 应排最后；t2 2025 春 → 应排最前；t3 无 starts_on 按 code 回退。
	t1 := createTerm(t, conn, "2024-2025-1", "2024 秋", parseTermDate(t, "2024-09-01"))
	t2 := createTerm(t, conn, "2025-2026-2", "2026 春", parseTermDate(t, "2026-03-01"))
	t3 := createTerm(t, conn, "2025-2026-1", "2025 秋", nil)
	_ = createOffering(t, conn, c, t1, "四平路校区")
	_ = createOffering(t, conn, c, t2, "嘉定校区")
	_ = createOffering(t, conn, c, t3, "四平路校区")
	// t4 被隐藏 offering 引用 → 不应出现；t5 被隐藏课程引用 → 不应出现；t6 空 code → 不应出现。
	hiddenOffering := OfferingEntity{CourseId: c, TermId: t2, Status: OfferingStatusHidden}
	if err := conn.Create(&hiddenOffering).Error; err != nil {
		t.Fatalf("create hidden offering: %v", err)
	}
	_ = createOffering(t, conn, c, t2, "四平路校区") // 与 t2 重复（t2 已有可见 offering）→ 去重
	t4 := createTerm(t, conn, "2023-2024-1", "2023 秋", parseTermDate(t, "2023-09-01"))
	hiddenCourse := createCourse(t, conn, "100081", "CS")
	if err := conn.Model(&Entity{}).Where("id = ?", hiddenCourse).Update("status", StatusHidden).Error; err != nil {
		t.Fatalf("hide course: %v", err)
	}
	_ = createOffering(t, conn, hiddenCourse, t4, "嘉定校区")
	ghost := createCourse(t, conn, "100082", "CS")
	if err := conn.Delete(&Entity{Id: ghost}).Error; err != nil {
		t.Fatalf("soft-delete course: %v", err)
	}
	_ = createOffering(t, conn, ghost, t4, "沪西校区")
	t5 := createTerm(t, conn, "2022-2023-1", "", parseTermDate(t, "2022-09-01"))
	_ = createOffering(t, conn, c, t5, "嘉定校区")
	t6 := createTerm(t, conn, "", "空 code", parseTermDate(t, "2021-09-01"))
	_ = createOffering(t, conn, c, t6, "四平路校区")
	if err := conn.Delete(&TermEntity{Id: t5}).Error; err != nil {
		t.Fatalf("soft-delete term: %v", err)
	}

	got, err := ListDistinctTerms()
	if err != nil {
		t.Fatalf("ListDistinctTerms err = %v", err)
	}
	want := []string{"2025-2026-2", "2025-2026-1", "2024-2025-1"} // starts_on 倒序，t3 无日期回退 code
	if len(got) != len(want) {
		t.Fatalf("ListDistinctTerms codes = %v, want %v", termCodes(got), want)
	}
	for i, w := range want {
		if got[i].Code != w {
			t.Fatalf("ListDistinctTerms codes = %v, want %v", termCodes(got), want)
		}
	}
}

// TestListDistinctCampuses 校区列表去重与排序：仅可见课程的可见 offering 校区，
// 排除空值、隐藏课程、隐藏 offering 与软删 offering；按字典序。
func TestListDistinctCampuses(t *testing.T) {
	conn := setupCourseRepTest(t)
	c := createCourse(t, conn, "100090", "CS")
	_ = createOffering(t, conn, c, 0, "四平路校区")
	_ = createOffering(t, conn, c, 0, "嘉定校区")
	_ = createOffering(t, conn, c, 0, "四平路校区") // 重复校区 → 去重
	_ = createOffering(t, conn, c, 0, "")      // 空校区 → 排除
	hiddenCourse := createCourse(t, conn, "100091", "CS")
	if err := conn.Model(&Entity{}).Where("id = ?", hiddenCourse).Update("status", StatusHidden).Error; err != nil {
		t.Fatalf("hide course: %v", err)
	}
	_ = createOffering(t, conn, hiddenCourse, 0, "沪西校区")
	hiddenOffering := OfferingEntity{CourseId: c, Campus: "临港校区", Status: OfferingStatusHidden}
	if err := conn.Create(&hiddenOffering).Error; err != nil {
		t.Fatalf("create hidden offering: %v", err)
	}
	ghostOffering := OfferingEntity{CourseId: c, Campus: "虹口校区", Status: OfferingStatusVisible}
	if err := conn.Create(&ghostOffering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	if err := conn.Delete(&OfferingEntity{Id: ghostOffering.Id}).Error; err != nil {
		t.Fatalf("soft-delete offering: %v", err)
	}

	got, err := ListDistinctCampuses()
	if err != nil {
		t.Fatalf("ListDistinctCampuses err = %v", err)
	}
	want := []string{"嘉定校区", "四平路校区"}
	if len(got) != len(want) {
		t.Fatalf("ListDistinctCampuses = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListDistinctCampuses = %v, want %v", got, want)
		}
	}
}

// termCodes 提取学期的 code 序列，便于断言排序结果。
func termCodes(terms []TermEntity) []string {
	codes := make([]string, 0, len(terms))
	for _, t := range terms {
		codes = append(codes, t.Code)
	}
	return codes
}

// TestListCoursesLikeEscaping Instructor 的 LIKE 通配符（%/_）被转义：输入 % 只按字面匹配，不会命中全部课程。
func TestListCoursesLikeEscaping(t *testing.T) {
	conn := setupCourseRepTest(t)
	zhang := createTestInstructor(t, conn, "张三", "zhangsan", "zhangsan", "zs")
	li := createTestInstructor(t, conn, "李四", "lisi", "lisi", "ls")
	cA := createCourse(t, conn, "100040", "CS")
	cB := createCourse(t, conn, "100041", "CS")
	linkCourseInstructor(t, conn, cA, zhang)
	linkCourseInstructor(t, conn, cB, li)

	// "%" 转义后按字面匹配，两个教师名均不含 %，应返回 0；未转义则 %% 会命中全部。
	got, total, err := ListCourses(ListCourseQuery{Instructor: []string{"%"}, Size: 50})
	if err != nil {
		t.Fatalf("ListCourses(instructor=%%) err = %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Fatalf("instructor=%%: total=%d len=%d, want 0/0", total, len(got))
	}
}

// TestListCoursesSoftDeletedCourseExcluded 软删课程既不出现在列表，也不计入 total（Count 与 Find 口径一致）。
func TestListCoursesSoftDeletedCourseExcluded(t *testing.T) {
	conn := setupCourseRepTest(t)
	_ = createCourse(t, conn, "100050", "CS")
	ghost := createCourse(t, conn, "100051", "CS")
	if err := conn.Delete(&Entity{Id: ghost}).Error; err != nil {
		t.Fatalf("soft-delete course: %v", err)
	}

	got, total, err := ListCourses(ListCourseQuery{Size: 50})
	if err != nil {
		t.Fatalf("ListCourses err = %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Fatalf("ListCourses total=%d len=%d, want 1/1 (soft-deleted excluded)", total, len(got))
	}
}

// TestListCoursesInstructorHiddenOfferingExcluded 隐藏开课（offering.status=hidden）的教师不被 instructor 搜出。
func TestListCoursesInstructorHiddenOfferingExcluded(t *testing.T) {
	conn := setupCourseRepTest(t)
	ins := createTestInstructor(t, conn, "王五", "wangwu", "wangwu", "ww")
	c := createCourse(t, conn, "100060", "CS")
	hiddenOffering := OfferingEntity{CourseId: c, Status: OfferingStatusHidden}
	if err := conn.Create(&hiddenOffering).Error; err != nil {
		t.Fatalf("create hidden offering: %v", err)
	}
	if err := conn.Create(&OfferingInstructorEntity{OfferingId: hiddenOffering.Id, InstructorId: ins}).Error; err != nil {
		t.Fatalf("link instructor: %v", err)
	}

	got, total, err := ListCourses(ListCourseQuery{Instructor: []string{"王五"}, Size: 50})
	if err != nil {
		t.Fatalf("ListCourses(instructor=王五) err = %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Fatalf("hidden offering: total=%d len=%d, want 0/0", total, len(got))
	}
}

// TestListCoursesHasReviewSoftDeletedStatsExcluded HasReview 忽略软删的统计行。
func TestListCoursesHasReviewSoftDeletedStatsExcluded(t *testing.T) {
	conn := setupCourseRepTest(t)
	c := createCourse(t, conn, "100070", "CS")
	setCourseStats(t, conn, c, 2, 8, 3)
	if err := conn.Delete(&CourseStatsEntity{CourseId: c}).Error; err != nil {
		t.Fatalf("soft-delete stats: %v", err)
	}

	got, total, err := ListCourses(ListCourseQuery{HasReview: true, Size: 50})
	if err != nil {
		t.Fatalf("ListCourses(HasReview) err = %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Fatalf("soft-deleted stats: total=%d len=%d, want 0/0", total, len(got))
	}
}

// TestListCoursesByPrimaryCodesExcludesHidden 回归 PR #197 P13：
// ListCoursesByPrimaryCodes 是 P13 课评摘要匹配的底层查询，必须与 ListCourses 一致
// 只返回 StatusVisible 的课程，隐藏课程即使主课号命中也不得返回。
func TestListCoursesByPrimaryCodesExcludesHidden(t *testing.T) {
	conn := setupCourseRepTest(t)

	visible := Entity{PrimaryCode: "100001", Name: "高等数学(A)上", Department: "数学科学学院", Status: StatusVisible}
	if err := conn.Create(&visible).Error; err != nil {
		t.Fatalf("create visible course: %v", err)
	}
	hidden := Entity{PrimaryCode: "100002", Name: "被隐藏的课程", Department: "某学院", Status: StatusHidden}
	if err := conn.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden course: %v", err)
	}

	got, err := ListCoursesByPrimaryCodes([]string{"100001", "100002", "999999"})
	if err != nil {
		t.Fatalf("ListCoursesByPrimaryCodes: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("result size = %d, want 1 (only visible course); got %+v", len(got), got)
	}
	if got[0].PrimaryCode != "100001" {
		t.Fatalf("primary_code = %q, want 100001", got[0].PrimaryCode)
	}
}

// TestListVisibleOfferingsByClassCodes P13 回归（review P1/P2/P3）：
// - 入参班号带点（122004.01）去点后匹配 offering.class_code（12200401）
// - JOIN course.status=Visible：隐藏课程的 offering 不得出现在公开响应
// - termId > 0 时只返回该学期（跨学期班号复用不串学期）
func TestListVisibleOfferingsByClassCodes(t *testing.T) {
	conn := setupCourseRepTest(t)
	visible := createCourse(t, conn, "122004", "CS")
	hidden := createCourse(t, conn, "199999", "CS")
	if err := conn.Model(&Entity{}).Where("id = ?", hidden).Update("status", StatusHidden).Error; err != nil {
		t.Fatalf("hide course: %v", err)
	}
	term1 := createTerm(t, conn, "2025-2026-1", "学期1", parseTermDate(t, "2025-09-01"))
	term2 := createTerm(t, conn, "2025-2026-2", "学期2", parseTermDate(t, "2026-02-23"))

	offerings := []OfferingEntity{
		{CourseId: visible, TermId: term1, ClassCode: "12200401", Status: OfferingStatusVisible},
		{CourseId: visible, TermId: term2, ClassCode: "12200401", Status: OfferingStatusVisible}, // 跨学期班号复用
		{CourseId: hidden, TermId: term1, ClassCode: "12200401", Status: OfferingStatusVisible},  // 隐藏课程的可见 offering
		{CourseId: visible, TermId: term1, ClassCode: "12200401", Status: OfferingStatusHidden},  // 隐藏 offering
	}
	for _, o := range offerings {
		if err := conn.Create(&o).Error; err != nil {
			t.Fatalf("create offering: %v", err)
		}
	}

	// 带点入参 + 不限定学期：只返回可见课程的两个学期 offering（隐藏课程/隐藏 offering 排除）。
	got, err := ListVisibleOfferingsByClassCodes([]string{"122004.01"}, 0)
	if err != nil {
		t.Fatalf("ListVisibleOfferingsByClassCodes err = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("all-terms result size = %d, want 2; got %+v", len(got), got)
	}

	// 限定 term1：只剩该学期的 offering。
	got1, err := ListVisibleOfferingsByClassCodes([]string{"122004.01"}, term1)
	if err != nil {
		t.Fatalf("ListVisibleOfferingsByClassCodes(term1) err = %v", err)
	}
	if len(got1) != 1 || got1[0].TermId != term1 {
		t.Fatalf("term-scoped result = %+v, want single offering of term %d", got1, term1)
	}

	// 空入参：空数组而非 nil。
	gotEmpty, err := ListVisibleOfferingsByClassCodes([]string{"  "}, 0)
	if err != nil {
		t.Fatalf("ListVisibleOfferingsByClassCodes(empty) err = %v", err)
	}
	if gotEmpty == nil || len(gotEmpty) != 0 {
		t.Fatalf("empty result = %v, want non-nil empty slice", gotEmpty)
	}
}
