package api

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"gorm.io/gorm"
)

// injectSeq 为注入回调生成唯一名称，避免同一次测试运行内重复注册。
var injectSeq atomic.Uint64

// injectWriteFailure 注册一个 GORM 回调，在指定表（table）的第 n 次写操作前注入错误，
// 并返回 fired 标志。op 取 "create" 或 "update"。回调继承到事务连接，因此可在
// WriteTopic 的单事务内精确命中某一步写入（如首帖插入、指针回写、分类索引写入），
// 用于逐步失败原子性测试。回调随 t.Cleanup 移除，不影响同包其他测试。
// 测试必须在 WriteTopic 后断言 fired：若注入未命中（Statement.Table 不匹配或
// 前置守卫提前返回 FAIL），回滚断言会因"什么都没写"而假通过，fired 可区分二者。
func injectWriteFailure(t *testing.T, conn *gorm.DB, op, table string, n int) *atomic.Bool {
	t.Helper()
	fired := &atomic.Bool{}
	name := fmt.Sprintf("inject_%s_%s_%d", op, table, injectSeq.Add(1))
	count := 0
	fn := func(db *gorm.DB) {
		if db.Statement == nil || db.Statement.Table != table {
			return
		}
		count++
		if count == n {
			fired.Store(true)
			db.AddError(errors.New("injected write failure: " + op + " " + table))
		}
	}
	var registerErr error
	if op == "create" {
		registerErr = conn.Callback().Create().Before("gorm:create").Register(name, fn)
	} else {
		registerErr = conn.Callback().Update().Before("gorm:update").Register(name, fn)
	}
	if registerErr != nil {
		t.Fatalf("register %s injection: %v", op, registerErr)
	}
	t.Cleanup(func() {
		if op == "create" {
			_ = conn.Callback().Create().Remove(name)
		} else {
			_ = conn.Callback().Update().Remove(name)
		}
	})
	return fired
}

// createPublishedTopicWithPost 直接插入一条已发布话题及其首帖（PostNo=1），
// 供编辑分支（TopicId>0）的原子回滚测试使用。
func createPublishedTopicWithPost(t *testing.T, conn *gorm.DB, authorID, topicID, postID uint64, title, content string) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: topicID, Title: title, UserId: authorID, Status: 1, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	firstPost := posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, UserId: authorID, Content: content, ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("first_post_id", postID).Error; err != nil {
		t.Fatalf("set first_post_id: %v", err)
	}
}

// assertTopicWriteAbsent 断言指定标题的话题及其首帖都不存在（整体回滚）。
func assertTopicWriteAbsent(t *testing.T, conn *gorm.DB, title, content string) {
	t.Helper()
	var topic topics.Entity
	if err := conn.Where("title = ?", title).First(&topic).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("topic %q should be rolled back, got err=%v topic=%+v", title, err, topic)
	}
	var post posts.Entity
	if err := conn.Where("content = ?", content).First(&post).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("first post %q should be rolled back, got err=%v post=%+v", content, err, post)
	}
}

func TestWriteTopicCreateRollsBackWhenPostInsertFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2001, "author")
	if err := conn.Create(&category.Entity{Id: 3001, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	fired := injectWriteFailure(t, conn, "create", "posts", 1)

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2001,
		Params: WriteTopicReq{
			Title:       "Atomic create post-fail title",
			Content:     "Atomic create post-fail content with enough words",
			CategoryId:  []uint64{3001},
			TopicStatus: 1,
		},
	})
	if !fired.Load() {
		t.Fatalf("injection did not fire: create posts #1")
	}
	if res.Data.Code != component.FAIL {
		t.Fatalf("WriteTopic() = code=%v, want FAIL (injected post insert failure)", res.Data.Code)
	}
	// 首帖插入失败 ⇒ 话题、首帖、分类索引应全部回滚，不留孤立数据。
	assertTopicWriteAbsent(t, conn, "Atomic create post-fail title", "Atomic create post-fail content with enough words")
}

func TestWriteTopicCreateRollsBackWhenPointerSaveFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2002, "author")
	if err := conn.Create(&category.Entity{Id: 3002, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	// 创建分支中 topics 的 UPDATE 仅出现在指针回写（FirstPostId/LastPostId/LastPostedAt）。
	fired := injectWriteFailure(t, conn, "update", "topics", 1)

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2002,
		Params: WriteTopicReq{
			Title:       "Atomic create pointer-fail title",
			Content:     "Atomic create pointer-fail content with enough words",
			CategoryId:  []uint64{3002},
			TopicStatus: 1,
		},
	})
	if !fired.Load() {
		t.Fatalf("injection did not fire: update topics #1 (pointer write)")
	}
	if res.Data.Code != component.FAIL {
		t.Fatalf("WriteTopic() = code=%v, want FAIL (injected pointer save failure)", res.Data.Code)
	}
	assertTopicWriteAbsent(t, conn, "Atomic create pointer-fail title", "Atomic create pointer-fail content with enough words")
}

func TestWriteTopicCreateRollsBackWhenCategoryIndexFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2003, "author")
	if err := conn.Create(&category.Entity{Id: 3003, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	// 新话题无既有分类索引，ReplaceTopicCategories 会走 Create 新增。
	fired := injectWriteFailure(t, conn, "create", "topic_category_index", 1)

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2003,
		Params: WriteTopicReq{
			Title:       "Atomic create category-fail title",
			Content:     "Atomic create category-fail content with enough words",
			CategoryId:  []uint64{3003},
			TopicStatus: 1,
		},
	})
	if !fired.Load() {
		t.Fatalf("injection did not fire: create topic_category_index #1")
	}
	if res.Data.Code != component.FAIL {
		t.Fatalf("WriteTopic() = code=%v, want FAIL (injected category index failure)", res.Data.Code)
	}
	assertTopicWriteAbsent(t, conn, "Atomic create category-fail title", "Atomic create category-fail content with enough words")
}

// 编辑分支首笔写入（topics.SaveTx）失败：此处注入的是事务内第一笔写，失败时
// 尚无先前写入可回滚，断言的是"错误被传播、请求返回 FAIL"（非回滚语义——
// 真正的整体回滚由后续 PostSaveFails/CategoryIndexSaveFails 等第二笔起失败的
// 测试覆盖）。保留此测试用于验证事务入口能正确将首笔错误上抛，避免静默吞错。
func TestWriteTopicEditRollsBackWhenTopicSaveFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2004, "author")
	createPublishedTopicWithPost(t, conn, 2004, 4004, 5004, "Original title", "Original content")
	if err := conn.Create(&category.Entity{Id: 3004, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	fired := injectWriteFailure(t, conn, "update", "topics", 1)

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2004,
		Params: WriteTopicReq{
			TopicId:     4004,
			Title:       "Edited title",
			Content:     "Edited content with enough words",
			CategoryId:  []uint64{3004},
			TopicStatus: 1,
		},
	})
	if !fired.Load() {
		t.Fatalf("injection did not fire: update topics #1 (edit branch first write)")
	}
	if res.Data.Code != component.FAIL {
		t.Fatalf("WriteTopic() edit = code=%v, want FAIL (injected topic save failure)", res.Data.Code)
	}
	if got := topics.Get(4004); got.Title != "Original title" {
		t.Fatalf("topic title after rollback = %q, want original %q", got.Title, "Original title")
	}
	if got := posts.Get(5004); got.Content != "Original content" {
		t.Fatalf("post content after rollback = %q, want original %q", got.Content, "Original content")
	}
}

func TestWriteTopicEditRollsBackWhenPostSaveFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2005, "author")
	createPublishedTopicWithPost(t, conn, 2005, 4005, 5005, "Original title", "Original content")
	if err := conn.Create(&category.Entity{Id: 3005, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	// 编辑分支 topics.Save 在前，posts.Save 在后；注入 posts UPDATE 命中首帖保存。
	fired := injectWriteFailure(t, conn, "update", "posts", 1)

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2005,
		Params: WriteTopicReq{
			TopicId:     4005,
			Title:       "Edited title",
			Content:     "Edited content with enough words",
			CategoryId:  []uint64{3005},
			TopicStatus: 1,
		},
	})
	if !fired.Load() {
		t.Fatalf("injection did not fire: update posts #1 (edit branch first-post save)")
	}
	if res.Data.Code != component.FAIL {
		t.Fatalf("WriteTopic() edit = code=%v, want FAIL (injected post save failure)", res.Data.Code)
	}
	if got := topics.Get(4005); got.Title != "Original title" {
		t.Fatalf("topic title after rollback = %q, want original %q", got.Title, "Original title")
	}
	if got := posts.Get(5005); got.Content != "Original content" {
		t.Fatalf("post content after rollback = %q, want original %q", got.Content, "Original content")
	}
}

// 成功路径：提交后统计副作用（userStatistics.WriteTopic）应执行，
// 与失败路径"整体回滚、无副作用"形成闭环验证。
func TestWriteTopicCreateIncrementsUserStatsAfterCommit(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2007, "author")
	if err := conn.Create(&category.Entity{Id: 3007, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2007,
		Params: WriteTopicReq{
			Title:       "Atomic success stats title",
			Content:     "Atomic success stats content with enough words",
			CategoryId:  []uint64{3007},
			TopicStatus: 1,
		},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("WriteTopic() = code=%v, want SUCCESS", res.Data.Code)
	}
	var stats userStatistics.Entity
	if err := conn.Where("user_id = ?", 2007).First(&stats).Error; err != nil {
		t.Fatalf("load user stats: %v", err)
	}
	if stats.TopicCount != 1 {
		t.Fatalf("topic_count after publish = %d, want 1 (after-commit stat side effect ran)", stats.TopicCount)
	}
}

// 编辑成功路径（isEdit 快照回归点）：编辑已发布话题成功时不得走创建分支副作用——
// topic_count 不得递增、不得发 TopicPublishedEvent（若有人误将提交后分支判断
// 复用被 CreateTx 回填的 topic.Id，新建话题会误走编辑分支、编辑会误走创建分支，
// 统计会静默错乱）。此处断言 topic_count 保持 0，即可捕获该回归。
func TestWriteTopicEditDoesNotIncrementUserStats(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2009, "author")
	createPublishedTopicWithPost(t, conn, 2009, 4009, 5009, "Original title", "Original content")
	if err := conn.Create(&category.Entity{Id: 3010, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2009,
		Params: WriteTopicReq{
			TopicId:     4009,
			Title:       "Edited title",
			Content:     "Edited content with enough words",
			CategoryId:  []uint64{3010},
			TopicStatus: 1,
		},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("WriteTopic() edit = code=%v, want SUCCESS", res.Data.Code)
	}
	var stats userStatistics.Entity
	if err := conn.Where("user_id = ?", 2009).First(&stats).Error; err != nil {
		t.Fatalf("load user stats: %v", err)
	}
	if stats.TopicCount != 0 {
		t.Fatalf("topic_count after edit = %d, want 0 (edit must not trigger create-branch WriteTopic)", stats.TopicCount)
	}
	// 编辑确实生效：标题已更新（避免"什么都没写"导致的假通过）。
	if got := topics.Get(4009); got.Title != "Edited title" {
		t.Fatalf("topic title after edit = %q, want %q (edit should have persisted)", got.Title, "Edited title")
	}
}

func TestWriteTopicEditRollsBackWhenCategoryIndexSaveFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2006, "author")
	createPublishedTopicWithPost(t, conn, 2006, 4010, 5010, "Original title", "Original content")
	if err := conn.Create(&category.Entity{Id: 3006, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	// 话题已有分类索引，编辑时 ReplaceTopicCategories 先对既有行 Save（update）。
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: 4010, CategoryId: 3006, Effective: 1}).Error; err != nil {
		t.Fatalf("create existing category index: %v", err)
	}
	fired := injectWriteFailure(t, conn, "update", "topic_category_index", 1)

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2006,
		Params: WriteTopicReq{
			TopicId:     4010,
			Title:       "Edited title",
			Content:     "Edited content with enough words",
			CategoryId:  []uint64{3006},
			TopicStatus: 1,
		},
	})
	if !fired.Load() {
		t.Fatalf("injection did not fire: update topic_category_index #1 (existing index save)")
	}
	if res.Data.Code != component.FAIL {
		t.Fatalf("WriteTopic() edit = code=%v, want FAIL (injected category index save failure)", res.Data.Code)
	}
	if got := topics.Get(4010); got.Title != "Original title" {
		t.Fatalf("topic title after rollback = %q, want original %q", got.Title, "Original title")
	}
	if got := posts.Get(5010); got.Content != "Original content" {
		t.Fatalf("post content after rollback = %q, want original %q", got.Content, "Original content")
	}
}

// 编辑分支「新增分类索引」的失败注入：话题+首帖已保存后，新分类索引 INSERT 失败
// 同样必须整体回滚（这是与既有行 Save 不同的独立部分写入状态）。
func TestWriteTopicEditRollsBackWhenNewCategoryIndexInsertFails(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 2008, "author")
	createPublishedTopicWithPost(t, conn, 2008, 4008, 5008, "Original title", "Original content")
	if err := conn.Create(&category.Entity{Id: 3008, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := conn.Create(&category.Entity{Id: 3009, Name: "Second", Slug: "second"}).Error; err != nil {
		t.Fatalf("create second category: %v", err)
	}
	// 话题已有分类 3008 的索引；编辑新增 3009 时 ReplaceTopicCategories 走 Create 插入新行。
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: 4008, CategoryId: 3008, Effective: 1}).Error; err != nil {
		t.Fatalf("create existing category index: %v", err)
	}
	fired := injectWriteFailure(t, conn, "create", "topic_category_index", 1)

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 2008,
		Params: WriteTopicReq{
			TopicId:     4008,
			Title:       "Edited title",
			Content:     "Edited content with enough words",
			CategoryId:  []uint64{3008, 3009},
			TopicStatus: 1,
		},
	})
	if !fired.Load() {
		t.Fatalf("injection did not fire: create topic_category_index #1 (new index insert on edit)")
	}
	if res.Data.Code != component.FAIL {
		t.Fatalf("WriteTopic() edit = code=%v, want FAIL (injected new category index insert failure)", res.Data.Code)
	}
	if got := topics.Get(4008); got.Title != "Original title" {
		t.Fatalf("topic title after rollback = %q, want original %q", got.Title, "Original title")
	}
	if got := posts.Get(5008); got.Content != "Original content" {
		t.Fatalf("post content after rollback = %q, want original %q", got.Content, "Original content")
	}
}
