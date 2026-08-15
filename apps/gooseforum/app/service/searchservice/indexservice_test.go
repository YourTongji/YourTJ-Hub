package searchservice

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

func TestConvertTopicToSearchDocument(t *testing.T) {
	createdAt := time.Unix(1700000000, 0)
	updatedAt := time.Unix(1700000300, 0)
	topic := &topics.Entity{
		Id:            42,
		Title:         "Searchable title",
		CategoryIds:   []uint64{3, 5},
		Status:        1,
		ProcessStatus: 0,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	firstPost := &posts.Entity{Content: "# Heading\n\nVisible text with [link](https://example.com).\n\n```go\nhidden()\n```"}

	got := convertTopicToSearchDocument(topic, firstPost)

	if got.ID != topic.Id || got.Title != topic.Title {
		t.Fatalf("unexpected identity fields: %#v", got)
	}
	if got.TopicStatus != topic.Status || got.ProcessStatus != topic.ProcessStatus {
		t.Fatalf("unexpected status fields: %#v", got)
	}
	if got.CreatedAt != createdAt.Unix() || got.UpdatedAt != updatedAt.Unix() {
		t.Fatalf("unexpected timestamps: %#v", got)
	}
	if len(got.Category) != 2 || got.Category[0] != 3 || got.Category[1] != 5 {
		t.Fatalf("Category = %#v, want [3 5]", got.Category)
	}
	if !strings.Contains(got.SearchContent, "Visible text") {
		t.Fatalf("SearchContent should include readable text, got %q", got.SearchContent)
	}
	if strings.Contains(got.SearchContent, "hidden") {
		t.Fatalf("SearchContent should skip fenced code, got %q", got.SearchContent)
	}
}

func TestConvertTopicToSearchDocumentStripsWikiFrontmatter(t *testing.T) {
	// issue #258：wiki 页面可携带 YAML frontmatter，搜索索引只索引剥离后的正文，
	// 元数据行（title/description/tags）不得进入 SearchContent。
	topic := &topics.Entity{
		Id:            7,
		Title:         "Wiki page",
		TopicType:     topics.TopicTypeWiki,
		Status:        1,
		ProcessStatus: topics.ProcessStatusNormal,
	}
	firstPost := &posts.Entity{Content: "---\ntitle: 元数据标题\ndescription: 元数据描述\ntags:\n  - 标签\n---\n\n# 标题\n\n正文内容"}

	got := convertTopicToSearchDocument(topic, firstPost)

	for _, forbidden := range []string{"元数据标题", "元数据描述", "标签", "title:", "description:", "---"} {
		if strings.Contains(got.SearchContent, forbidden) {
			t.Fatalf("SearchContent must not contain frontmatter %q, got %q", forbidden, got.SearchContent)
		}
	}
	if !strings.Contains(got.SearchContent, "正文内容") {
		t.Fatalf("SearchContent should contain body text, got %q", got.SearchContent)
	}

	// 非 wiki 话题不走剥离（原有行为不变）。
	forumTopic := &topics.Entity{Id: 8, Title: "Forum", TopicType: 0}
	forumPost := &posts.Entity{Content: "---\ntitle: 元数据\n---\n\n论坛正文"}
	forumGot := convertTopicToSearchDocument(forumTopic, forumPost)
	if !strings.Contains(forumGot.SearchContent, "元数据") {
		t.Fatalf("non-wiki content should stay unstripped, got %q", forumGot.SearchContent)
	}
}

func TestTopicIndexUsesTopicName(t *testing.T) {
	if TopicIndex != "topics" {
		t.Fatalf("TopicIndex = %q, want topics", TopicIndex)
	}
}

func TestTopicSearchDocumentDoesNotExposeLegacyType(t *testing.T) {
	if _, ok := reflect.TypeOf(TopicSearchDocument{}).FieldByName("Type"); ok {
		t.Fatalf("TopicSearchDocument should not expose legacy Type field")
	}
}

func TestGetTaskUIDNil(t *testing.T) {
	if got := getTaskUID(nil); got != nil {
		t.Fatalf("getTaskUID(nil) = %v, want nil", got)
	}
}

func TestIsTopicPubliclySearchable(t *testing.T) {
	base := topics.Entity{Id: 1, Title: "t", Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive}
	cases := []struct {
		name  string
		mut   func(*topics.Entity)
		want  bool
	}{
		{"published normal", func(e *topics.Entity) {}, true},
		{"unpublished (status 0)", func(e *topics.Entity) { e.Status = 0 }, false},
		{"pending review", func(e *topics.Entity) { e.ProcessStatus = topics.ProcessStatusPending }, false},
		{"blocked", func(e *topics.Entity) { e.ProcessStatus = topics.ProcessStatusBlocked }, false},
		{"user deleted", func(e *topics.Entity) { e.VisibilityStatus = topics.VisibilityUserDeleted }, false},
		{"moderator removed", func(e *topics.Entity) { e.VisibilityStatus = topics.VisibilityModeratorRemoved }, false},
		{"soft deleted", func(e *topics.Entity) { e.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true} }, false},
	}
	for _, tc := range cases {
		e := base
		tc.mut(&e)
		if got := isTopicPubliclySearchable(&e); got != tc.want {
			t.Fatalf("isTopicPubliclySearchable(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
	if isTopicPubliclySearchable(nil) {
		t.Fatal("isTopicPubliclySearchable(nil) = true, want false")
	}
}
