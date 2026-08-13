package courseservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// manageTestModels 课程/评价管理用到的全部 course 域 + taskQueue 表。
var manageTestModels = []any{
	&course.Entity{},
	&course.TermEntity{},
	&course.OfferingEntity{},
	&course.ReviewEntity{},
	&course.HelpfulEntity{},
	&course.CourseStatsEntity{},
	&course.OfferingStatsEntity{},
	&course.AliasEntity{},
	&course.InstructorEntity{},
	&course.OfferingInstructorEntity{},
	&course.SourceRefEntity{},
	&taskQueue.Entity{},
}

// setupManageTest 迁移并清空课程管理相关表。
func setupManageTest(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(manageTestModels...); err != nil {
		t.Fatalf("migrate manage tables: %v", err)
	}
	for _, model := range manageTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean manage table: %v", err)
		}
	}
}

func countTasksByType(t *testing.T, prefix string) int64 {
	t.Helper()
	var count int64
	if err := dbconnect.Connect().Table((&taskQueue.Entity{}).TableName()).
		Where("type LIKE ?", prefix+"%").
		Count(&count).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	return count
}

// TestCreateCourse 新增课程写入 normalized/pinyin/initials、别名、教师并入队搜索任务。
func TestCreateCourse(t *testing.T) {
	setupManageTest(t)
	item, err := CreateCourse(CourseCreateInput{
		PrimaryCode: "CS101",
		Name:        "数据结构",
		Department:  "计算机科学与技术系",
		CreditX10:   30,
		Aliases:     []string{"数据结构A"},
		Instructors: []string{"张三"},
	})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	if item.Id == 0 {
		t.Fatal("expected non-zero course id")
	}
	entity := course.GetCourse(item.Id)
	if entity.NormalizedName != Normalize("数据结构") {
		t.Fatalf("normalized name mismatch: %q", entity.NormalizedName)
	}
	if entity.NamePinyin == "" || entity.NameInitials == "" {
		t.Fatalf("expected pinyin/initials populated, got pinyin=%q initials=%q", entity.NamePinyin, entity.NameInitials)
	}
	aliases, err := course.ListAliasesByCourse(item.Id)
	if err != nil || len(aliases) != 1 || aliases[0].Value != "数据结构A" {
		t.Fatalf("alias mismatch: %v %v", aliases, err)
	}
	if countTasksByType(t, "course-search.") == 0 {
		t.Fatal("expected course search task enqueued")
	}
}

// TestUpdateCourseNameSyncsSearch 改名同步 normalized/pinyin/initials 并入队搜索任务。
func TestUpdateCourseNameSyncsSearch(t *testing.T) {
	setupManageTest(t)
	item, err := CreateCourse(CourseCreateInput{PrimaryCode: "CS101", Name: "数据结构"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	_ = countTasksByType(t, "course-search.") // 建立基线（首次入队任务）
	newName := "算法设计与分析"
	updated, err := UpdateCourse(item.Id, CourseUpdateInput{Name: &newName})
	if err != nil {
		t.Fatalf("update course name: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}
	entity := course.GetCourse(item.Id)
	if entity.NormalizedName != Normalize(newName) {
		t.Fatalf("normalized name not synced: %q", entity.NormalizedName)
	}
	if entity.NamePinyin == "" || entity.NameInitials == "" {
		t.Fatal("expected pinyin/initials recomputed on rename")
	}
	if countTasksByType(t, "course-search.") == 0 {
		t.Fatal("expected search task enqueued on rename")
	}
}

// TestDeleteCourseCascades 级联删除课程 + offering + 评价 + helpful + 统计 + 别名 + 来源映射。
func TestDeleteCourseCascades(t *testing.T) {
	setupManageTest(t)
	conn := dbconnect.Connect()

	// 课程 + offering + 教师关联
	c := course.Entity{PrimaryCode: "CS101", Name: "数据结构", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "2025-2026 第一学期", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	ins := course.InstructorEntity{Name: "张三", NormalizedName: "张三", Department: "计算机"}
	if err := conn.Create(&ins).Error; err != nil {
		t.Fatalf("create instructor: %v", err)
	}
	link := course.OfferingInstructorEntity{OfferingId: offering.Id, InstructorId: ins.Id, Role: "lecturer"}
	if err := conn.Create(&link).Error; err != nil {
		t.Fatalf("create instructor link: %v", err)
	}
	// 评价 + helpful + 统计
	rating := 5
	review := course.ReviewEntity{OfferingId: offering.Id, AuthorUserId: uint64Ptr(1001), Rating: &rating, Content: "很好", Status: course.ReviewStatusVisible}
	if err := conn.Create(&review).Error; err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := conn.Create(&course.HelpfulEntity{ReviewId: review.Id, UserId: 1002}).Error; err != nil {
		t.Fatalf("create helpful: %v", err)
	}
	if err := course.UpsertCourseStatsTx(conn, c.Id, 1, rating, 1); err != nil {
		t.Fatalf("upsert course stats: %v", err)
	}
	if err := course.UpsertOfferingStatsTx(conn, offering.Id, 1, rating, 1); err != nil {
		t.Fatalf("upsert offering stats: %v", err)
	}
	// 别名 + 来源映射
	if err := conn.Create(&course.AliasEntity{CourseId: c.Id, Kind: course.AliasKindName, Value: "数据结构A", NormalizedValue: "数据结构a"}).Error; err != nil {
		t.Fatalf("create alias: %v", err)
	}
	if err := conn.Create(&course.SourceRefEntity{Source: "s", EntityType: course.EntityTypeCourse, ExternalId: "e1", LocalId: c.Id}).Error; err != nil {
		t.Fatalf("create source ref: %v", err)
	}
	if err := conn.Create(&course.SourceRefEntity{Source: "s", EntityType: course.EntityTypeReview, ExternalId: "r1", LocalId: review.Id}).Error; err != nil {
		t.Fatalf("create review source ref: %v", err)
	}

	info, err := DeleteCourse(c.Id)
	if err != nil {
		t.Fatalf("delete course: %v", err)
	}
	if info.ReviewCount != 1 || info.OfferingCount != 1 {
		t.Fatalf("unexpected delete info: %+v", info)
	}
	// 课程、offering、评价、helpful、统计、别名、来源映射均被物理移除
	if got := course.GetCourseByIdTx(conn, c.Id); got.Id != 0 {
		t.Fatalf("course still present: %+v", got)
	}
	var offeringCount, reviewCount, helpfulCount, aliasCount, sourceRefCount, reviewSourceRefCount, courseStatsCount, offeringStatsCount int64
	conn.Unscoped().Table("course_offering").Where("id = ?", offering.Id).Count(&offeringCount)
	conn.Unscoped().Table("course_review").Where("id = ?", review.Id).Count(&reviewCount)
	conn.Unscoped().Table("course_review_helpful").Where("review_id = ?", review.Id).Count(&helpfulCount)
	conn.Unscoped().Table("course_alias").Where("course_id = ?", c.Id).Count(&aliasCount)
	conn.Unscoped().Table("course_source_ref").Where("local_id = ?", c.Id).Count(&sourceRefCount)
	conn.Unscoped().Table("course_source_ref").Where("entity_type = ? AND local_id = ?", course.EntityTypeReview, review.Id).Count(&reviewSourceRefCount)
	conn.Unscoped().Table("course_review_stats").Where("course_id = ?", c.Id).Count(&courseStatsCount)
	conn.Unscoped().Table("offering_review_stats").Where("offering_id = ?", offering.Id).Count(&offeringStatsCount)
	if offeringCount+reviewCount+helpfulCount+aliasCount+sourceRefCount+reviewSourceRefCount+courseStatsCount+offeringStatsCount != 0 {
		t.Fatalf("cascade delete left rows: offering=%d review=%d helpful=%d alias=%d sourceRef=%d reviewSourceRef=%d courseStats=%d offeringStats=%d",
			offeringCount, reviewCount, helpfulCount, aliasCount, sourceRefCount, reviewSourceRefCount, courseStatsCount, offeringStatsCount)
	}
	if countTasksByType(t, "course-search.") == 0 {
		t.Fatal("expected search delete task enqueued")
	}
}

// TestAdminReviewListSearch 按课程名/课号/正文搜索评价。
func TestAdminReviewListSearch(t *testing.T) {
	setupManageTest(t)
	conn := dbconnect.Connect()
	c := course.Entity{PrimaryCode: "CS101", Name: "数据结构", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "t", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	rating := 5
	if err := conn.Create(&course.ReviewEntity{OfferingId: offering.Id, AuthorUserId: uint64Ptr(1001), Rating: &rating, Content: "讲得很好", Status: course.ReviewStatusVisible}).Error; err != nil {
		t.Fatalf("create review: %v", err)
	}

	page, err := AdminReviewList(AdminReviewQuery{Keyword: "数据结构", Status: -1, PageSize: 20})
	if err != nil {
		t.Fatalf("list reviews by name: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].CourseName != "数据结构" {
		t.Fatalf("unexpected items: %+v", page.Items)
	}
	// 按正文搜索
	page, err = AdminReviewList(AdminReviewQuery{Keyword: "讲得很好", Status: -1, PageSize: 20})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list reviews by body: %v %+v", err, page.Items)
	}
}

// TestAdminUpdateReviewRatingSyncsStats 管理端改评分仅对可见评价同步 stats delta。
func TestAdminUpdateReviewRatingSyncsStats(t *testing.T) {
	setupManageTest(t)
	conn := dbconnect.Connect()
	c := course.Entity{PrimaryCode: "CS101", Name: "数据结构", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "t", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	rating := 3
	review := course.ReviewEntity{OfferingId: offering.Id, AuthorUserId: uint64Ptr(1001), Rating: &rating, Content: "内容", Status: course.ReviewStatusVisible}
	if err := conn.Create(&review).Error; err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := course.UpsertCourseStatsTx(conn, c.Id, 1, 3, 1); err != nil {
		t.Fatalf("upsert stats: %v", err)
	}
	if err := course.UpsertOfferingStatsTx(conn, offering.Id, 1, 3, 1); err != nil {
		t.Fatalf("upsert offering stats: %v", err)
	}

	newRating := 5
	if _, err := AdminUpdateReview(review.Id, AdminReviewUpdateInput{Rating: &newRating}); err != nil {
		t.Fatalf("update rating: %v", err)
	}
	stats, err := course.GetCourseStats(c.Id)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.RatingSum != 5 || stats.RatingCount != 1 || stats.ReviewCount != 1 {
		t.Fatalf("expected stats sum=5 count=1 reviews=1, got sum=%d count=%d reviews=%d",
			stats.RatingSum, stats.RatingCount, stats.ReviewCount)
	}
}

// intPtr 返回 int 指针，便于构造可空 rating。
func intPtr(v int) *int { return &v }

// TestAdminUpdateReviewSetsRatingOnUnratedReview 对未评分（rating NULL）的 legacy 评价补评分时
// rating_count 需 +1：否则平均分按偏小计数计算导致 ratingAvg 虚高（回归 #204 review）。
func TestAdminUpdateReviewSetsRatingOnUnratedReview(t *testing.T) {
	setupManageTest(t)
	conn := dbconnect.Connect()
	c := course.Entity{PrimaryCode: "CS101", Name: "数据结构", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "t", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	// legacy 导入：author_user_id=0 且 rating NULL（历史 0 转 NULL）。
	review := course.ReviewEntity{OfferingId: offering.Id, AuthorUserId: uint64Ptr(0), Rating: nil, Content: "历史评价", Status: course.ReviewStatusVisible, Source: course.ReviewSourceLegacyImport}
	if err := conn.Create(&review).Error; err != nil {
		t.Fatalf("create review: %v", err)
	}
	// 初始统计：可见未评分评价只计入 review_count。
	if err := course.UpsertCourseStatsTx(conn, c.Id, 0, 0, 1); err != nil {
		t.Fatalf("upsert course stats: %v", err)
	}
	if err := course.UpsertOfferingStatsTx(conn, offering.Id, 0, 0, 1); err != nil {
		t.Fatalf("upsert offering stats: %v", err)
	}

	newRating := 5
	if _, err := AdminUpdateReview(review.Id, AdminReviewUpdateInput{Rating: &newRating}); err != nil {
		t.Fatalf("update rating: %v", err)
	}
	stats, err := course.GetCourseStats(c.Id)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if stats.RatingSum != 5 || stats.RatingCount != 1 || stats.ReviewCount != 1 {
		t.Fatalf("expected course stats sum=5 count=1 reviews=1, got sum=%d count=%d reviews=%d",
			stats.RatingSum, stats.RatingCount, stats.ReviewCount)
	}
	offeringStats, err := course.GetOfferingStats(offering.Id)
	if err != nil {
		t.Fatalf("get offering stats: %v", err)
	}
	if offeringStats.RatingSum != 5 || offeringStats.RatingCount != 1 || offeringStats.ReviewCount != 1 {
		t.Fatalf("expected offering stats sum=5 count=1 reviews=1, got sum=%d count=%d reviews=%d",
			offeringStats.RatingSum, offeringStats.RatingCount, offeringStats.ReviewCount)
	}
}

// TestAdminDeleteReviewSyncsStats 管理端硬删除评价的 stats 扣减分支：
// 可见评分评价扣 (-1,-rating,-1)，可见未评分评价扣 (0,0,-1)，且评价行被物理删除。
func TestAdminDeleteReviewSyncsStats(t *testing.T) {
	for _, tc := range []struct {
		name      string
		rating    *int
		initCount int
		initSum   int
	}{
		{"rated", intPtr(5), 1, 5},
		{"unrated", nil, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupManageTest(t)
			conn := dbconnect.Connect()
			c := course.Entity{PrimaryCode: "CS101", Name: "数据结构", Status: course.StatusVisible}
			if err := conn.Create(&c).Error; err != nil {
				t.Fatalf("create course: %v", err)
			}
			term := course.TermEntity{Code: "2025-2026-1", Name: "t", Status: 0}
			if err := conn.Create(&term).Error; err != nil {
				t.Fatalf("create term: %v", err)
			}
			offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
			if err := conn.Create(&offering).Error; err != nil {
				t.Fatalf("create offering: %v", err)
			}
			review := course.ReviewEntity{OfferingId: offering.Id, AuthorUserId: uint64Ptr(1001), Rating: tc.rating, Content: "内容", Status: course.ReviewStatusVisible}
			if err := conn.Create(&review).Error; err != nil {
				t.Fatalf("create review: %v", err)
			}
			if err := conn.Create(&course.SourceRefEntity{Source: "s", EntityType: course.EntityTypeReview, ExternalId: "r1", LocalId: review.Id}).Error; err != nil {
				t.Fatalf("create review source ref: %v", err)
			}
			if err := course.UpsertCourseStatsTx(conn, c.Id, tc.initCount, tc.initSum, 1); err != nil {
				t.Fatalf("upsert course stats: %v", err)
			}
			if err := course.UpsertOfferingStatsTx(conn, offering.Id, tc.initCount, tc.initSum, 1); err != nil {
				t.Fatalf("upsert offering stats: %v", err)
			}

			if _, err := AdminDeleteReview(review.Id); err != nil {
				t.Fatalf("delete review: %v", err)
			}
			stats, err := course.GetCourseStats(c.Id)
			if err != nil {
				t.Fatalf("get course stats: %v", err)
			}
			if stats.RatingCount != 0 || stats.RatingSum != 0 || stats.ReviewCount != 0 {
				t.Fatalf("expected course stats zeroed, got count=%d sum=%d reviews=%d",
					stats.RatingCount, stats.RatingSum, stats.ReviewCount)
			}
			offeringStats, err := course.GetOfferingStats(offering.Id)
			if err != nil {
				t.Fatalf("get offering stats: %v", err)
			}
			if offeringStats.RatingCount != 0 || offeringStats.RatingSum != 0 || offeringStats.ReviewCount != 0 {
				t.Fatalf("expected offering stats zeroed, got count=%d sum=%d reviews=%d",
					offeringStats.RatingCount, offeringStats.RatingSum, offeringStats.ReviewCount)
			}
			var remaining int64
			if err := conn.Unscoped().Table("course_review").Where("id = ?", review.Id).Count(&remaining).Error; err != nil {
				t.Fatalf("count reviews: %v", err)
			}
			if remaining != 0 {
				t.Fatalf("review not physically deleted")
			}
			var refRemaining int64
			if err := conn.Unscoped().Table("course_source_ref").Where("entity_type = ? AND local_id = ?", course.EntityTypeReview, review.Id).Count(&refRemaining).Error; err != nil {
				t.Fatalf("count review source refs: %v", err)
			}
			if refRemaining != 0 {
				t.Fatalf("review source ref not cleaned up")
			}
		})
	}
}

// TestEnqueueCourseStatsRebuildTaskDedup 统计重建任务入队去重。
func TestEnqueueCourseStatsRebuildTaskDedup(t *testing.T) {
	setupManageTest(t)
	if err := EnqueueCourseStatsRebuildTask(); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := EnqueueCourseStatsRebuildTask(); err != nil {
		t.Fatalf("enqueue again: %v", err)
	}
	if got := countTasksByType(t, TaskTypeCourseStatsRebuild); got != 1 {
		t.Fatalf("expected 1 deduped task, got %d", got)
	}
}
