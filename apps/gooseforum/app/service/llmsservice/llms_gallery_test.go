package llmsservice

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestAbsoluteImageURL(t *testing.T) {
	for _, tc := range []struct{ base, raw, want string }{
		{"https://f.example.test", "/file/img/2026/09/a.webp", "https://f.example.test/file/img/2026/09/a.webp"},
		{"https://f.example.test/", "//file/img/a.webp", ""},
		{"https://f.example.test/", "file/img/a.webp", "https://f.example.test/file/img/a.webp"},
		{"https://f.example.test", "https://cdn.example.test/x/b.webp", "https://cdn.example.test/x/b.webp"},
		{"https://f.example.test", "HTTP://cdn.example.test/c.png", "HTTP://cdn.example.test/c.png"},
		{"https://f.example.test", "javascript:alert(1)", ""},
		{"https://f.example.test", "/file/img/a b.webp", ""},
		{"https://f.example.test", "/file/img/a>.webp", ""},
		{"https://f.example.test", "  ", ""},
		{"", "/file/img/a.webp", ""},
	} {
		if got := absoluteImageURL(tc.base, tc.raw); got != tc.want {
			t.Errorf("absoluteImageURL(%q, %q) = %q, want %q", tc.base, tc.raw, got, tc.want)
		}
	}
	if gallerySection("###", "https://f.example.test", []string{"javascript:x"}) != "" {
		t.Fatal("gallery with no usable URL must render nothing")
	}
}

func TestBuildTopicListsGalleryOutsideFence(t *testing.T) {
	fixture := setupLLMSProjectionFixture(t)
	host := "https://cap.example.test"
	setLLMSSettings(t, fixture.conn, pageConfig.LLMSConfig{Enabled: true, Files: true})
	t.Cleanup(ClearCache)
	gallery := []string{"/file/img/2026/09/09/a.webp", "https://cdn.example.test/b.webp"}
	if err := fixture.conn.Model(&topics.Entity{}).Where("id = ?", fixture.publicTopicID).Update("image_urls", `["/file/img/2026/09/09/a.webp","https://cdn.example.test/b.webp"]`).Error; err != nil {
		t.Fatal(err)
	}
	ClearCache()
	doc, err := BuildTopic(host, fixture.publicTopicID)
	if err != nil {
		t.Fatalf("BuildTopic err=%v", err)
	}
	section := "### Images\n\n- ![image 1](<https://cap.example.test/file/img/2026/09/09/a.webp>)\n- ![image 2](<https://cdn.example.test/b.webp>)\n\n"
	assertContains(t, doc, section)
	original := strings.Index(doc, "## Original post")
	images := strings.Index(doc, "### Images")
	fenceClose := strings.LastIndex(doc[:images], "```\n")
	if original < 0 || images < original || fenceClose < original {
		t.Fatalf("gallery must follow the closed first-post fence:\n%s", doc)
	}
	if replies := strings.Index(doc, "## Replies"); replies >= 0 && replies < images {
		t.Fatalf("gallery must belong to the original post, before replies:\n%s", doc)
	}
	_ = gallery
}

func TestAppendTopicDocumentCountsGalleryInByteBudget(t *testing.T) {
	fixture := setupLLMSProjectionFixture(t)
	t.Cleanup(ClearCache)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_400_000_000
	topicID, postID := base+1, base+2
	t.Cleanup(func() {
		fixture.conn.Unscoped().Where("id = ?", postID).Delete(&posts.Entity{})
		fixture.conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
	})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := fixture.conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, Content: "gallery body", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{Id: topicID, Title: "Gallery cap", FirstPostId: postID, Status: 1, ProcessStatus: topics.ProcessStatusNormal, ImageUrls: []string{"/file/img/a.webp", "/file/img/b.webp"}, CreatedAt: now, UpdatedAt: now}
	if err := fixture.conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	var full strings.Builder
	if err := appendTopicDocument(&full, "https://cap.example.test", &topic, 1, 0); err != nil {
		t.Fatal(err)
	}
	assertContains(t, full.String(), "### Images")
	sectionStart := strings.Index(full.String(), "### Images")
	// Budget fits the post body but not the gallery section: must truncate, not overflow.
	var capped strings.Builder
	err := appendTopicDocument(&capped, "https://cap.example.test", &topic, 1, int64(sectionStart+10))
	if !errors.Is(err, errOutputLimit) {
		t.Fatalf("err = %v, want errOutputLimit", err)
	}
	if int64(capped.Len()) > int64(sectionStart+10) || strings.Contains(capped.String(), "### Images") {
		t.Fatalf("gallery exceeded byte budget: %d bytes\n%s", capped.Len(), capped.String())
	}
}
