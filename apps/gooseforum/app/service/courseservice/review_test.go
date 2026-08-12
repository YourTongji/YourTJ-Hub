package courseservice

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
)

// reviewTestModels 测试用到的 course 域 + taskQueue 表（写评会入队搜索任务）。
var reviewTestModels = []any{
	&course.Entity{},
	&course.TermEntity{},
	&course.OfferingEntity{},
	&course.ReviewEntity{},
	&course.HelpfulEntity{},
	&course.CourseStatsEntity{},
	&course.OfferingStatsEntity{},
	&taskQueue.Entity{},
}

// setupReviewTest 迁移 course 域表并清空，返回一个可见 offering。
func setupReviewTest(t *testing.T) (courseId, offeringId uint64) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(reviewTestModels...); err != nil {
		t.Fatalf("migrate review tables: %v", err)
	}
	for _, model := range reviewTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean review table: %v", err)
		}
	}
	c := course.Entity{PrimaryCode: "100001", Name: "高等数学(A)上", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "2025-2026 第一学期", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Campus: "四平路校区", Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	return c.Id, offering.Id
}

func reviewPayloadJSON(t *testing.T, payload ReviewPayload) string {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal review payload: %v", err)
	}
	return string(data)
}

// TestCreateReviewAnonymousZeroLeak 匿名评价 DTO 不泄漏任何身份字段。
func TestCreateReviewAnonymousZeroLeak(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{
		OfferingId:  offeringId,
		Rating:      5,
		Content:     "老师讲得很好",
		IsAnonymous: true,
	})
	if err != nil {
		t.Fatalf("create anonymous review: %v", err)
	}
	if payload.Author.Kind != "anonymous" {
		t.Fatalf("expected anonymous author kind, got %q", payload.Author.Kind)
	}
	raw := reviewPayloadJSON(t, payload)
	for _, leak := range []string{"authorUserId", "userId", "username", "avatar", "profileUrl", "id\":1001"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("anonymous payload leaks %q: %s", leak, raw)
		}
	}
	// 匿名只对公众匿名：作者本人管理自己的评价（canEdit/canDelete）是正确行为。
	if !payload.Viewer.CanEdit || !payload.Viewer.CanDelete {
		t.Fatalf("author must be able to manage own anonymous review: %+v", payload.Viewer)
	}
	// 他人视角（未登录 viewer=0）必须看到 canEdit/canDelete=false。
	others, err := ListReviewsByOffering(offeringId, 0)
	if err != nil {
		t.Fatalf("list anonymous review as stranger: %v", err)
	}
	if len(others) != 1 {
		t.Fatalf("expected 1 review, got %d", len(others))
	}
	if others[0].Viewer.CanEdit || others[0].Viewer.CanDelete {
		t.Fatalf("stranger must not see ownership of anonymous review: %+v", others[0].Viewer)
	}
	rawOthers := reviewPayloadJSON(t, others[0])
	for _, leak := range []string{"authorUserId", "userId", "username", "avatar"} {
		if strings.Contains(rawOthers, leak) {
			t.Fatalf("stranger anonymous payload leaks %q: %s", leak, rawOthers)
		}
	}
}

// TestCreateReviewDuplicate 同一用户对同一 offering 重复评价返回 409 语义错误。
func TestCreateReviewDuplicate(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	input := CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "第一次评价", IsAnonymous: false}
	if _, err := CreateReview(1001, input); err != nil {
		t.Fatalf("first review: %v", err)
	}
	if _, err := CreateReview(1001, input); err != ErrReviewDuplicate {
		t.Fatalf("expected ErrReviewDuplicate, got %v", err)
	}
}

// TestCreateReviewValidation 越界评分与空正文返回稳定错误。
func TestCreateReviewValidation(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 0, Content: "x"}); err != ErrRatingOutOfRange {
		t.Fatalf("rating 0: expected ErrRatingOutOfRange, got %v", err)
	}
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 6, Content: "x"}); err != ErrRatingOutOfRange {
		t.Fatalf("rating 6: expected ErrRatingOutOfRange, got %v", err)
	}
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 3, Content: ""}); err != ErrReviewContentEmpty {
		t.Fatalf("empty content: expected ErrReviewContentEmpty, got %v", err)
	}
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: 999999, Rating: 3, Content: "x"}); err != ErrOfferingNotFound {
		t.Fatalf("missing offering: expected ErrOfferingNotFound, got %v", err)
	}
}

// TestUpdateReviewNotOwned 用户不能修改/删除他人评价。
func TestUpdateReviewNotOwned(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "作者评价", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	rating := 5
	if _, err := UpdateReview(1002, payload.Id, UpdateReviewInput{Rating: &rating}); err != ErrReviewNotOwned {
		t.Fatalf("update others: expected ErrReviewNotOwned, got %v", err)
	}
	if err := DeleteReview(1002, payload.Id); err != ErrReviewNotOwned {
		t.Fatalf("delete others: expected ErrReviewNotOwned, got %v", err)
	}
}

// TestDeleteReviewRemovesFromList 删除后评价不再出现在公开列表（隔离窗口语义）。
func TestDeleteReviewRemovesFromList(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "将被删除", IsAnonymous: true})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := DeleteReview(1001, payload.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	list, err := ListReviewsByCourse(courseId, 0)
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list after delete, got %d items", len(list))
	}
}

// TestReviewStatsDelta 创建/删除评价时课程与 offering 统计投影同步增减。
func TestReviewStatsDelta(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "五星好评", IsAnonymous: false}); err != nil {
		t.Fatalf("create review: %v", err)
	}
	courseStats, err := course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 1 || courseStats.RatingSum != 5 || courseStats.ReviewCount != 1 {
		t.Fatalf("unexpected course stats after create: %+v", courseStats)
	}
	offeringStats, err := course.GetOfferingStats(offeringId)
	if err != nil {
		t.Fatalf("get offering stats: %v", err)
	}
	if offeringStats.RatingCount != 1 || offeringStats.RatingSum != 5 || offeringStats.ReviewCount != 1 {
		t.Fatalf("unexpected offering stats after create: %+v", offeringStats)
	}
	// 删除后回退
	review, _ := course.FindReviewByOfferingAndUser(offeringId, 1001)
	if err := DeleteReview(1001, review.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	courseStats, _ = course.GetCourseStats(courseId)
	if courseStats.RatingCount != 0 || courseStats.RatingSum != 0 || courseStats.ReviewCount != 0 {
		t.Fatalf("unexpected course stats after delete: %+v", courseStats)
	}
}

// TestSetReviewVisibility 审核隐藏后评价对普通用户不可见，恢复后重新可见。
func TestSetReviewVisibility(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 3, Content: "一般般", IsAnonymous: true})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := SetReviewVisibility(payload.Id, true); err != nil {
		t.Fatalf("hide review: %v", err)
	}
	list, err := ListReviewsByCourse(courseId, 0)
	if err != nil {
		t.Fatalf("list after hide: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list after hide, got %d", len(list))
	}
	if err := SetReviewVisibility(payload.Id, false); err != nil {
		t.Fatalf("show review: %v", err)
	}
	list, err = ListReviewsByCourse(courseId, 0)
	if err != nil {
		t.Fatalf("list after show: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 review after show, got %d", len(list))
	}
}

// TestReviewHelpfulIdempotent 重复标记 helpful 幂等，计数不重复累计。
func TestReviewHelpfulIdempotent(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "有帮助", IsAnonymous: true})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if err := SetReviewHelpful(2001, payload.Id, true); err != nil {
		t.Fatalf("first helpful: %v", err)
	}
	if err := SetReviewHelpful(2001, payload.Id, true); err != nil {
		t.Fatalf("duplicate helpful must be idempotent: %v", err)
	}
	if err := SetReviewHelpful(2002, payload.Id, true); err != nil {
		t.Fatalf("second user helpful: %v", err)
	}
	list, err := ListReviewsByOffering(offeringId, 2001)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].HelpfulCount != 2 {
		t.Fatalf("expected helpfulCount=2, got %+v", list)
	}
	if !list[0].Viewer.IsHelpful {
		t.Fatal("expected viewer 2001 isHelpful=true")
	}
	if err := SetReviewHelpful(2001, payload.Id, false); err != nil {
		t.Fatalf("unhelpful: %v", err)
	}
	list, _ = ListReviewsByOffering(offeringId, 2001)
	if list[0].HelpfulCount != 1 {
		t.Fatalf("expected helpfulCount=1 after unhelpful, got %d", list[0].HelpfulCount)
	}
}

// TestLegacyReviewAnonymized 历史匿名评价（author 0 / legacy-import）公开为 legacy 且零身份泄漏。
func TestLegacyReviewAnonymized(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	conn := dbconnect.Connect()
	rating := 0
	legacy := course.ReviewEntity{
		OfferingId:   offeringId,
		AuthorUserId: 0,
		Rating:       &rating,
		Content:      "历史评价正文",
		IsAnonymous:  true,
		Status:       course.ReviewStatusVisible,
		Source:       "legacy-import",
	}
	// rating=0 的 legacy 行按设计转 NULL 不计平均：这里直接验证公开 DTO 行为。
	legacy.Rating = nil
	if err := conn.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy review: %v", err)
	}
	list, err := ListReviewsByOffering(offeringId, 0)
	if err != nil {
		t.Fatalf("list legacy: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 legacy review, got %d", len(list))
	}
	if list[0].Author.Kind != "legacy" {
		t.Fatalf("expected legacy kind, got %q", list[0].Author.Kind)
	}
	if list[0].Rating != nil {
		t.Fatalf("expected nil rating for rating=0 legacy review, got %d", *list[0].Rating)
	}
	raw := reviewPayloadJSON(t, list[0])
	for _, leak := range []string{"authorUserId", "username", "avatar"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("legacy payload leaks %q: %s", leak, raw)
		}
	}
}

// TestCreateReviewMemberLabelVisibleToOwner 非匿名评价作者本人可编辑，他人不可编辑。
func TestCreateReviewMemberLabelVisibleToOwner(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "署名评价", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if payload.Author.Kind != "member" {
		t.Fatalf("expected member kind, got %q", payload.Author.Kind)
	}
	if !payload.Viewer.CanEdit || !payload.Viewer.CanDelete {
		t.Fatalf("owner should see canEdit/canDelete: %+v", payload.Viewer)
	}
	// 他人视角（未登录）不可编辑
	list, err := ListReviewsByOffering(offeringId, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Viewer.CanEdit || list[0].Viewer.CanDelete {
		t.Fatalf("stranger must not see ownership: %+v", list)
	}
}

// TestDeleteReviewAfterHideDoesNotDoubleDecrement 隐藏后再删除不得重复扣减 stats
// （隐藏时 SetReviewVisibility 已扣减；删除只对可见评价扣减）。
func TestDeleteReviewAfterHideDoesNotDoubleDecrement(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "隐藏后删除", IsAnonymous: false}); err != nil {
		t.Fatalf("create review: %v", err)
	}
	review, err := course.FindReviewByOfferingAndUser(offeringId, 1001)
	if err != nil {
		t.Fatalf("find review: %v", err)
	}
	if err := SetReviewVisibility(review.Id, true); err != nil {
		t.Fatalf("hide review: %v", err)
	}
	courseStats, err := course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 0 || courseStats.RatingSum != 0 || courseStats.ReviewCount != 0 {
		t.Fatalf("expected zero stats after hide, got %+v", courseStats)
	}
	if err := DeleteReview(1001, review.Id); err != nil {
		t.Fatalf("delete hidden review: %v", err)
	}
	courseStats, _ = course.GetCourseStats(courseId)
	if courseStats.RatingCount != 0 || courseStats.RatingSum != 0 || courseStats.ReviewCount != 0 {
		t.Fatalf("stats corrupted after deleting hidden review: %+v", courseStats)
	}
	offeringStats, err := course.GetOfferingStats(offeringId)
	if err != nil {
		t.Fatalf("get offering stats: %v", err)
	}
	if offeringStats.RatingCount != 0 || offeringStats.RatingSum != 0 || offeringStats.ReviewCount != 0 {
		t.Fatalf("offering stats corrupted after deleting hidden review: %+v", offeringStats)
	}
}

// TestDeleteReviewIdempotent 重复删除幂等（隔离窗口内再次删除直接成功，不重复扣减）。
func TestDeleteReviewIdempotent(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "幂等删除", IsAnonymous: false}); err != nil {
		t.Fatalf("create review: %v", err)
	}
	review, err := course.FindReviewByOfferingAndUser(offeringId, 1001)
	if err != nil {
		t.Fatalf("find review: %v", err)
	}
	if err := DeleteReview(1001, review.Id); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := DeleteReview(1001, review.Id); err != nil {
		t.Fatalf("second delete must be idempotent: %v", err)
	}
	courseStats, err := course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 0 || courseStats.RatingSum != 0 || courseStats.ReviewCount != 0 {
		t.Fatalf("expected zero stats after idempotent delete, got %+v", courseStats)
	}
}

// TestReviewUniqueOfferingAuthor 数据库唯一索引兜底：同 offering+用户直接插入第二行被拒。
func TestReviewUniqueOfferingAuthor(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	rating := 5
	first := course.ReviewEntity{OfferingId: offeringId, AuthorUserId: 1001, Rating: &rating, Content: "第一条", Status: course.ReviewStatusVisible}
	if err := dbconnect.Connect().Create(&first).Error; err != nil {
		t.Fatalf("create first review: %v", err)
	}
	second := course.ReviewEntity{OfferingId: offeringId, AuthorUserId: 1001, Rating: &rating, Content: "第二条", Status: course.ReviewStatusVisible}
	if err := dbconnect.Connect().Create(&second).Error; err == nil {
		t.Fatal("expected duplicate key error for same offering+user")
	}
}

// TestUpsertCourseStatsAtomicAccumulates 原子 upsert（INSERT ... ON CONFLICT DO UPDATE + delta）
// 跨多次调用正确累加，首次插入、后续冲突增量不丢。
func TestUpsertCourseStatsAtomicAccumulates(t *testing.T) {
	setupReviewTest(t) // 迁移并清空 course/offering stats 表
	conn := dbconnect.Connect()
	if err := course.UpsertCourseStatsTx(conn, 42, 1, 5, 1); err != nil {
		t.Fatalf("first course upsert: %v", err)
	}
	if err := course.UpsertCourseStatsTx(conn, 42, 1, 3, 1); err != nil {
		t.Fatalf("second course upsert: %v", err)
	}
	if err := course.UpsertOfferingStatsTx(conn, 7, 1, 5, 1); err != nil {
		t.Fatalf("first offering upsert: %v", err)
	}
	courseStats, err := course.GetCourseStats(42)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 2 || courseStats.RatingSum != 8 || courseStats.ReviewCount != 2 {
		t.Fatalf("course stats after two upserts = %+v, want rating_count=2 sum=8 review_count=2", courseStats)
	}
	offeringStats, err := course.GetOfferingStats(7)
	if err != nil {
		t.Fatalf("get offering stats: %v", err)
	}
	if offeringStats.RatingCount != 1 || offeringStats.RatingSum != 5 || offeringStats.ReviewCount != 1 {
		t.Fatalf("offering stats = %+v, want 1/5/1", offeringStats)
	}
}

// TestUpdateReviewContentPresence PATCH 部分更新语义：content 缺省（nil）保留原正文，
// 显式空串才清空；单改 rating/anonymous 不得把正文冲成空。
func TestUpdateReviewContentPresence(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	payload, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "原始正文", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	conn := dbconnect.Connect()
	// 仅改 rating，不传 content → 正文保留。
	rating := 5
	if _, err := UpdateReview(1001, payload.Id, UpdateReviewInput{Rating: &rating}); err != nil {
		t.Fatalf("update without content: %v", err)
	}
	var ent course.ReviewEntity
	if err := conn.Where("id = ?", payload.Id).First(&ent).Error; err != nil {
		t.Fatalf("load review after rating-only update: %v", err)
	}
	if ent.Content != "原始正文" {
		t.Fatalf("omitted content must preserve stored body, got %q", ent.Content)
	}
	// 显式空串 → 清空正文（契约：空串清除 body 而保留评价）。
	empty := ""
	if _, err := UpdateReview(1001, payload.Id, UpdateReviewInput{Content: &empty}); err != nil {
		t.Fatalf("update with explicit empty content: %v", err)
	}
	if err := conn.Where("id = ?", payload.Id).First(&ent).Error; err != nil {
		t.Fatalf("load review after empty-content update: %v", err)
	}
	if ent.Content != "" {
		t.Fatalf("explicit empty content must clear stored body, got %q", ent.Content)
	}
}

// TestLegacyReviewExposesLegacyHelpfulCount 历史导入的 legacy_helpful_count 计入公开 helpfulCount。
func TestLegacyReviewExposesLegacyHelpfulCount(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	conn := dbconnect.Connect()
	legacy := course.ReviewEntity{
		OfferingId:         offeringId,
		AuthorUserId:       0,
		Content:            "历史评价",
		IsAnonymous:        true,
		Status:             course.ReviewStatusVisible,
		Source:             course.ReviewSourceLegacyImport,
		LegacyHelpfulCount: 7,
	}
	if err := conn.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy review: %v", err)
	}
	list, err := ListReviewsByOffering(offeringId, 0)
	if err != nil {
		t.Fatalf("list legacy: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 legacy review, got %d", len(list))
	}
	if list[0].HelpfulCount != 7 {
		t.Fatalf("expected helpfulCount=7 (native 0 + legacy 7), got %d", list[0].HelpfulCount)
	}
}

// TestRecreateReviewAfterDelete 删除（软删）后同一用户可对同一 offering 重新评价：
// deleted 行占用唯一索引 (offering_id, author_user_id)，重新评价必须复用该行
// （ReactivateReviewTx），stats 按新 rating 重新累加；再次创建才返回 ErrReviewDuplicate。
func TestRecreateReviewAfterDelete(t *testing.T) {
	courseId, offeringId := setupReviewTest(t)
	first, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "第一次评价", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create first review: %v", err)
	}
	if err := DeleteReview(1001, first.Id); err != nil {
		t.Fatalf("delete review: %v", err)
	}
	courseStats, err := course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats: %v", err)
	}
	if courseStats.RatingCount != 0 || courseStats.RatingSum != 0 || courseStats.ReviewCount != 0 {
		t.Fatalf("expected zero stats after delete, got %+v", courseStats)
	}
	// 同一 offering 重新评价必须成功（复用 deleted 行，而非唯一键冲突 409）。
	second, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "重新评价", IsAnonymous: true})
	if err != nil {
		t.Fatalf("recreate review after delete must succeed: %v", err)
	}
	if second.Id != first.Id {
		t.Fatalf("recreated review should reuse soft-deleted row id %d, got %d", first.Id, second.Id)
	}
	if second.Rating == nil || *second.Rating != 5 {
		t.Fatalf("expected recreated rating 5, got %+v", second.Rating)
	}
	if !second.Viewer.CanEdit || !second.Viewer.CanDelete {
		t.Fatalf("recreated review owner should see canEdit/canDelete: %+v", second.Viewer)
	}
	courseStats, err = course.GetCourseStats(courseId)
	if err != nil {
		t.Fatalf("get course stats after recreate: %v", err)
	}
	if courseStats.RatingCount != 1 || courseStats.RatingSum != 5 || courseStats.ReviewCount != 1 {
		t.Fatalf("expected stats 1/5/1 after recreate, got %+v", courseStats)
	}
	if _, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 3, Content: "第三次", IsAnonymous: false}); err != ErrReviewDuplicate {
		t.Fatalf("expected ErrReviewDuplicate after recreate, got %v", err)
	}
}

// TestListHelpfulReviewIDsByUserBatch 批量查询当前用户 helpful 标记：
// 多条评价中只返回已标记项；取消（物理删除）后不再返回。
func TestListHelpfulReviewIDsByUserBatch(t *testing.T) {
	_, offeringId := setupReviewTest(t)
	a, err := CreateReview(1001, CreateReviewInput{OfferingId: offeringId, Rating: 4, Content: "甲的评价", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review A: %v", err)
	}
	b, err := CreateReview(1002, CreateReviewInput{OfferingId: offeringId, Rating: 5, Content: "乙的评价", IsAnonymous: false})
	if err != nil {
		t.Fatalf("create review B: %v", err)
	}
	if err := SetReviewHelpful(3001, a.Id, true); err != nil {
		t.Fatalf("mark helpful: %v", err)
	}
	marked, err := course.ListHelpfulReviewIDsByUser(3001, []uint64{a.Id, b.Id})
	if err != nil {
		t.Fatalf("batch list helpful: %v", err)
	}
	if len(marked) != 1 || !marked[a.Id] || marked[b.Id] {
		t.Fatalf("expected only review %d marked, got %+v", a.Id, marked)
	}
	// 取消标记（物理删除）后批量查询不再返回。
	if err := SetReviewHelpful(3001, a.Id, false); err != nil {
		t.Fatalf("unhelpful: %v", err)
	}
	marked, err = course.ListHelpfulReviewIDsByUser(3001, []uint64{a.Id, b.Id})
	if err != nil {
		t.Fatalf("batch list helpful after unhelpful: %v", err)
	}
	if len(marked) != 0 {
		t.Fatalf("expected no marked reviews after unhelpful, got %+v", marked)
	}
}
