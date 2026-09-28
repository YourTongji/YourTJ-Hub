package forum

import (
	"html/template"
	"strings"
	"testing"
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
			ID:   895,
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
