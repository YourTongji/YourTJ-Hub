package forum

import (
	"html/template"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
)

func TestTopicListLabelNeverEmpty(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		want        string
	}{
		{name: "title wins", title: "标题", description: "摘要", want: "标题"},
		{name: "description fallback", title: "", description: "摘要", want: "摘要"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := topicListLabel(tt.title, tt.description); got != tt.want {
				t.Fatalf("topicListLabel() = %q, want %q", got, tt.want)
			}
		})
	}

	// 纯图瞬间：标题与摘要都为空时必须有非空回退文案。
	if fallback := topicListLabel("", ""); fallback == "" {
		t.Fatal("topicListLabel() returned an empty label for an untitled, excerpt-less topic")
	}

	// 纯贴纸摘要：链接文案用去 token 的纯文本，不能露出 token。
	if label := topicListLabel("", "[:sticker:crawler_label:]"); strings.Contains(label, "[:sticker:") || label == "" {
		t.Fatalf("topicListLabel() = %q, want token-free text", label)
	}
}

// issue #1304：列表卡片把贴纸渲染成图片，爬虫/无 JS 的 SSR 列表同样展开；
// 公开口径与列表 API 一致（个人 u_* 与停用/审核中名字不解析），其余文字转义。
func TestTopicListDescriptionRendersPublicStickersOnly(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&sticker.Entity{}, &sticker.Alias{}); err != nil {
		t.Fatal(err)
	}
	rows := []sticker.Entity{
		{Name: "crawler_ok", FileName: "stickers/crawler_ok.png", ReviewStatus: sticker.ReviewStatusApproved},
		{Name: "crawler_off", FileName: "stickers/crawler_off.png", ReviewStatus: sticker.ReviewStatusApproved},
		{Name: "u_crawler_private", FileName: "stickers/crawler_private.png", ReviewStatus: sticker.ReviewStatusApproved},
	}
	for i := range rows {
		// gorm 的 default 标签会跳过零值字段，先落行再用 Save 写入 false 的启用/官方标记。
		if err := conn.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows[1].IsEnabled = false
	rows[2].IsOfficial = false
	for i := 1; i < len(rows); i++ {
		if err := sticker.Save(&rows[i]); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		conn.Where("name IN ?", []string{"crawler_ok", "crawler_off", "u_crawler_private"}).Delete(&sticker.Entity{})
	})

	description := `A & <b>"q"</b> [:sticker:crawler_ok:]|[:sticker:crawler_off:]|[:sticker:u_crawler_private:]|[:sticker:crawler_missing:]`
	// 小字视图（爬虫列表）不展开图片：可解析的公开贴纸显示 [名字]，其余省略。
	label := topicListDescription(description)

	if !strings.Contains(label, "[crawler_ok]") {
		t.Fatalf("public sticker not rendered as a text label: %q", label)
	}
	if !strings.Contains(label, `A & <b>"q"</b>`) {
		t.Fatalf("text segment lost: %q", label)
	}
	for _, omitted := range []string{"crawler_off", "crawler_private", "crawler_missing", "[:sticker:"} {
		if strings.Contains(label, omitted) {
			t.Fatalf("disabled/personal/unknown sticker %q leaked: %q", omitted, label)
		}
	}
	if untouched := topicListDescription("纯文字"); untouched != "纯文字" {
		t.Fatalf("token-free description changed: %q", untouched)
	}

	// 模板接线：列表摘要用同一渲染，且模板不二次转义 <img>。
	tmpl, err := template.New("partials/topic_list.gohtml").
		Funcs(templateFuncs()).
		ParseFiles("../../../../resource/templates/partials/topic_list.gohtml")
	if err != nil {
		t.Fatalf("parse topic_list partial: %v", err)
	}
	var out strings.Builder
	data := map[string]any{"Lang": "en", "Topics": []TopicPayload{{ID: 1304, Title: "标题", Description: description}}}
	if err := tmpl.ExecuteTemplate(&out, "partials/topic_list.gohtml", data); err != nil {
		t.Fatalf("execute topic_list partial: %v", err)
	}
	html := out.String()
	if !strings.Contains(html, `class="gf-crawler-topic-excerpt"`) || !strings.Contains(html, "[crawler_ok]") {
		t.Fatalf("crawler list excerpt did not render the sticker label:\n%s", html)
	}
	if strings.Contains(html, "<img") || strings.Contains(html, "&lt;img") || strings.Contains(html, "[:sticker:") {
		t.Fatalf("crawler list excerpt rendered an image or raw token:\n%s", html)
	}
}

// issue #895：爬虫/无 JS 的 SSR 话题列表不能出现空链接文案。
func TestTopicListPartialRendersUntitledMomentFallback(t *testing.T) {
	tmpl, err := template.New("partials/topic_list.gohtml").
		Funcs(templateFuncs()).
		ParseFiles("../../../../resource/templates/partials/topic_list.gohtml")
	if err != nil {
		t.Fatalf("parse topic_list partial: %v", err)
	}
	data := map[string]any{
		"Lang": "en",
		"Topics": []TopicPayload{{
			ID:    895,
			Title: "",
			URL:   "/p/post/895",
		}},
	}
	var out strings.Builder
	if err := tmpl.ExecuteTemplate(&out, "partials/topic_list.gohtml", data); err != nil {
		t.Fatalf("execute topic_list partial: %v", err)
	}
	html := out.String()
	if strings.Contains(html, "rel=\"bookmark\"></a>") {
		t.Fatalf("crawler list rendered an empty link label:\n%s", html)
	}
	if !strings.Contains(html, topicListLabel("", "")) {
		t.Fatalf("crawler list is missing the fallback label %q:\n%s", topicListLabel("", ""), html)
	}
}
