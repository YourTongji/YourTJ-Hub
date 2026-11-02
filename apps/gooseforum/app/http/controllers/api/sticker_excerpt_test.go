package api

import (
	"strings"
	"testing"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

const stickerExcerptToken = "[:sticker:Atari哭哭:]"

func setupStickerExcerptTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{},
		&postRevisions.Entity{}, &moderationDecision.Entity{},
	); err != nil {
		t.Fatalf("migrate sticker excerpt tables: %v", err)
	}
	return conn
}

func seedStickerExcerptRows(t *testing.T, conn *gorm.DB, rows ...any) {
	t.Helper()
	for _, row := range rows {
		if err := conn.Create(row).Error; err != nil {
			t.Fatalf("create sticker excerpt row: %v", err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(row) })
	}
}

// 我的内容 / 已删内容列表是纯文本投影：摘要去掉贴纸 token，话题与回复两路一致。
func TestMyContentAndDeletedContentListsStripStickerTokens(t *testing.T) {
	conn := setupStickerExcerptTestDB(t)
	owner := users.EntityComplete{Id: 978901, Username: "sticker-content-owner"}
	topic := topics.Entity{Id: 978901, UserId: owner.Id, Title: "贴纸话题", Excerpt: "开头 " + stickerExcerptToken + " 结尾", Status: 1, FirstPostId: 978905}
	first := posts.Entity{Id: 978905, TopicId: topic.Id, PostNo: 1, UserId: owner.Id, Content: "正文 " + stickerExcerptToken}
	reply := posts.Entity{Id: 978906, TopicId: topic.Id, PostNo: 2, UserId: owner.Id, Content: "回复 " + stickerExcerptToken + " 正文"}
	removedTopic := topics.Entity{Id: 978902, UserId: owner.Id, Title: "已删贴纸话题", Excerpt: "已删 " + stickerExcerptToken, Status: 1, FirstPostId: 978907, VisibilityStatus: topics.VisibilityUserDeleted, RetentionStatus: topics.RetentionRecoverable}
	removedFirst := posts.Entity{Id: 978907, TopicId: removedTopic.Id, PostNo: 1, UserId: owner.Id, Content: "已删正文"}
	removedReply := posts.Entity{Id: 978908, TopicId: topic.Id, PostNo: 3, UserId: owner.Id, Content: "已删回复 " + stickerExcerptToken, VisibilityStatus: posts.VisibilityUserDeleted, RetentionStatus: posts.RetentionRecoverable}
	seedStickerExcerptRows(t, conn, &owner, &topic, &first, &reply, &removedTopic, &removedFirst, &removedReply)

	request := func(contentType string) map[string]any {
		t.Helper()
		resp := MyContentList(component.BetterRequest[MyContentListReq]{UserId: owner.Id, Params: MyContentListReq{ContentType: contentType, Limit: 10}})
		payload, ok := resp.Data.Result.(map[string]any)
		if !ok {
			t.Fatalf("my content %s payload = %#v", contentType, resp.Data)
		}
		return payload
	}
	topicItems, ok := request("topic")["items"].([]MyContentItem)
	if !ok || len(topicItems) != 1 || topicItems[0].Excerpt != "开头 结尾" {
		t.Fatalf("my content topic items = %+v", topicItems)
	}
	postItems, ok := request("post")["items"].([]MyContentItem)
	if !ok {
		t.Fatalf("my content post items = %+v", postItems)
	}
	for _, item := range postItems {
		if strings.Contains(item.Excerpt, "[:sticker:") {
			t.Fatalf("my content post excerpt kept a token: %q", item.Excerpt)
		}
		if item.PostNo == reply.PostNo && item.Excerpt != "回复 正文" {
			t.Fatalf("my content reply excerpt = %q", item.Excerpt)
		}
	}

	deleted := func(contentType string) map[string]any {
		t.Helper()
		resp := DeletedContentList(component.BetterRequest[DeletedContentListReq]{UserId: owner.Id, Params: DeletedContentListReq{ContentType: contentType, Limit: 10}})
		payload, ok := resp.Data.Result.(map[string]any)
		if !ok {
			t.Fatalf("deleted content %s payload = %#v", contentType, resp.Data)
		}
		return payload
	}
	deletedTopics, ok := deleted("topic")["items"].([]DeletedContentItem)
	if !ok || len(deletedTopics) != 1 || deletedTopics[0].Excerpt != "已删" {
		t.Fatalf("deleted topic items = %+v", deletedTopics)
	}
	deletedPosts, ok := deleted("post")["items"].([]DeletedContentItem)
	if !ok || len(deletedPosts) != 1 || deletedPosts[0].Excerpt != "已删回复" {
		t.Fatalf("deleted post items = %+v", deletedPosts)
	}
}

// 后台审核队列：excerpt 是文本预览，Content 保留完整 Markdown 供审核阅读。
func TestReviewQueueExcerptsStripStickerTokens(t *testing.T) {
	conn := setupStickerExcerptTestDB(t)
	author := users.EntityComplete{Id: 978951, Username: "sticker-queue-author"}
	topic := topics.Entity{Id: 978951, UserId: author.Id, Title: "待审贴纸话题", Excerpt: "待审 " + stickerExcerptToken, Status: 1, ProcessStatus: topics.ProcessStatusPending, FirstPostId: 978955}
	first := posts.Entity{Id: 978955, TopicId: topic.Id, PostNo: 1, UserId: author.Id, Content: "待审正文 " + stickerExcerptToken}
	pendingReply := posts.Entity{Id: 978956, TopicId: topic.Id, PostNo: 2, UserId: author.Id, Content: "待审回复 " + stickerExcerptToken, ProcessStatus: posts.ProcessStatusPending}
	// 全 token 摘要剥除后为空，队列回退标题，不退化成空白行。
	tokenOnly := topics.Entity{Id: 978952, UserId: author.Id, Title: "只有贴纸的话题", Excerpt: stickerExcerptToken, Status: 1, ProcessStatus: topics.ProcessStatusPending, FirstPostId: 978957}
	tokenOnlyFirst := posts.Entity{Id: 978957, TopicId: tokenOnly.Id, PostNo: 1, UserId: author.Id, Content: "正文"}
	// 多字节摘要：按 rune 截断，不能把汉字切成半个。
	longReply := posts.Entity{Id: 978958, TopicId: topic.Id, PostNo: 3, UserId: author.Id, Content: "a" + strings.Repeat("字", 130), ProcessStatus: posts.ProcessStatusPending}
	seedStickerExcerptRows(t, conn, &author, &topic, &first, &pendingReply, &tokenOnly, &tokenOnlyFirst, &longReply)

	queue := func(kind string) func(uint64) *ReviewQueueItem {
		t.Helper()
		// 共享内存库里同包其它用例也留下待审内容；仓储层 BoundPageSize 把每页夹到 30 条，
		// 这里取满一页，避免条目落第 2 页。
		resp := ReviewQueue(component.BetterRequest[ReviewQueueReq]{Params: ReviewQueueReq{Kind: kind, PageSize: 30}})
		payload, ok := resp.Data.Result.(map[string]any)
		if !ok {
			t.Fatalf("review queue %s payload = %#v", kind, resp.Data)
		}
		items, ok := payload["items"].([]ReviewQueueItem)
		if !ok {
			t.Fatalf("review queue %s items = %#v", kind, payload["items"])
		}
		// 同包的其它用例也在共享内存库里留下待审内容，按 ID 取本条。
		return func(id uint64) *ReviewQueueItem {
			t.Helper()
			for i := range items {
				if items[i].Id == id {
					return &items[i]
				}
			}
			t.Fatalf("review queue %s missing id %d in %+v", kind, id, items)
			return nil
		}
	}
	topicItem := queue("topic")(topic.Id)
	if topicItem.Excerpt != "待审" {
		t.Fatalf("topic review excerpt = %q", topicItem.Excerpt)
	}
	if topicItem.Content != first.Content {
		t.Fatalf("topic review content changed: %q", topicItem.Content)
	}
	postItem := queue("post")(pendingReply.Id)
	if postItem.Excerpt != "待审回复" {
		t.Fatalf("post review excerpt = %q", postItem.Excerpt)
	}
	if postItem.Content != pendingReply.Content {
		t.Fatalf("post review content changed: %q", postItem.Content)
	}
	if item := queue("topic")(tokenOnly.Id); item.Excerpt != tokenOnly.Title {
		t.Fatalf("token-only excerpt should fall back to title: %q", item.Excerpt)
	}
	longItem := queue("post")(longReply.Id)
	if !utf8.ValidString(longItem.Excerpt) || len([]rune(longItem.Excerpt)) != 120 {
		t.Fatalf("multi-byte excerpt truncated badly: %q", longItem.Excerpt)
	}
}

// 后台话题管理列表的 description 是纯文本摘要；同一个 TopicAdminBaseVo 在
// TopicSource（编辑器回填源）里必须保留原摘要，否则会静默丢掉 token。
func TestAdminTopicListStripsStickerTokensButEditorSourceKeepsThem(t *testing.T) {
	conn := setupStickerExcerptTestDB(t)
	author := users.EntityComplete{Id: 978971, Username: "sticker-admin-author"}
	topic := topics.Entity{Id: 978971, UserId: author.Id, Title: "后台贴纸话题", Excerpt: "摘要 " + stickerExcerptToken, Status: 1, FirstPostId: 978975}
	first := posts.Entity{Id: 978975, TopicId: topic.Id, PostNo: 1, UserId: author.Id, Content: "正文"}
	seedStickerExcerptRows(t, conn, &author, &topic, &first)

	res := TopicsList(component.BetterRequest[TopicsListReq]{Params: TopicsListReq{Page: 1, PageSize: 10, Search: topic.Title}})
	page, ok := res.Data.Result.(component.Page[TopicInfoAdminVo])
	if !ok || len(page.List) != 1 || page.List[0].Description != "摘要" {
		t.Fatalf("admin topic list = %#v", res.Data.Result)
	}
	source := TopicSource(component.BetterRequest[TopicSourceReq]{Params: TopicSourceReq{TopicId: topic.Id}})
	sourceVo, ok := source.Data.Result.(TopicSourceVo)
	if !ok || sourceVo.Description != topic.Excerpt {
		t.Fatalf("editor source description = %#v", source.Data.Result)
	}
}

// 已删内容与我的内容的回复摘要共用 excerptOf：先剥 token 再截断，避免半个 token。
func TestExcerptOfStripsStickerTokensBeforeTruncation(t *testing.T) {
	if got := excerptOf("前缀 " + stickerExcerptToken + " 尾巴"); got != "前缀 尾巴" {
		t.Fatalf("excerptOf = %q", got)
	}
	if kept := excerptOf("没有贴纸的摘要"); kept != "没有贴纸的摘要" {
		t.Fatalf("token-free excerpt changed: %q", kept)
	}
	long := strings.Repeat("字", 99) + " " + stickerExcerptToken
	if got := excerptOf(long); got != strings.Repeat("字", 99) {
		t.Fatalf("truncation kept a partial token: %q", got)
	}
}

// Agent 话题列表同属纯文本摘要投影。
func TestAgentTopicItemStripsStickerTokensFromExcerpt(t *testing.T) {
	item := toAgentTopicItem(topics.Entity{Id: 1, Title: "话题", Excerpt: "摘要 " + stickerExcerptToken})
	if item.Excerpt != "摘要" {
		t.Fatalf("agent topic excerpt = %q", item.Excerpt)
	}
}
