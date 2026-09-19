package campusservice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type messageProvider struct {
	*fakeProvider
	detailCalls int
}

func (p *messageProvider) Data(_ context.Context, key, _ string) (any, error) {
	if key == "profile" {
		return []any{map[string]any{"userId": p.id, "name": "测试同学", "sexName": "private"}}, nil
	}
	return map[string]any{"list": []any{map[string]any{"id": "123", "title": "通知", "publishTime": "2026-09-19 10:00:00"}}}, nil
}
func (p *messageProvider) Message(context.Context, string, string) (any, error) {
	p.detailCalls++
	return map[string]any{"id": "123", "content": "<p>通知正文</p>"}, nil
}

func TestMessageRequiresMembershipInOwnList(t *testing.T) {
	s, p := setup(t)
	bind(t, s, 1)
	m := &messageProvider{fakeProvider: p}
	s.provider = m
	for _, id := range []string{"124", "../123", "0", "123&other=1"} {
		if _, e := s.Message(context.Background(), 1, id); !errors.Is(e, ErrMessageNotFound) {
			t.Fatalf("unavailable ID accepted: %v", e)
		}
	}
	if m.detailCalls != 0 {
		t.Fatal("unlisted detail requested upstream")
	}
	v, e := s.Message(context.Background(), 1, "123")
	if e != nil || v.Content != "通知正文" || m.detailCalls != 1 {
		t.Fatalf("listed message failed: %v", e)
	}
	if _, e = s.Message(context.Background(), 2, "123"); !errors.Is(e, ErrAuthorization) {
		t.Fatal("unbound account read message")
	}
}

func TestPrivateProfileExposesOnlyOwnName(t *testing.T) {
	s, p := setup(t)
	bind(t, s, 1)
	s.provider = &messageProvider{fakeProvider: p}
	v, e := s.Dataset(context.Background(), 1, "profile")
	if e != nil || len(v.Metrics) != 1 || v.Metrics[0].Value != "测试同学" {
		t.Fatalf("profile failed: %v", e)
	}
	b, marshalErr := json.Marshal(v)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if strings.Contains(string(b), "private") || strings.Contains(string(b), p.id) {
		t.Fatal("private profile fields leaked")
	}
	p.id = "different-student"
	if _, e = s.Dataset(context.Background(), 1, "profile"); !errors.Is(e, ErrAuthorization) {
		t.Fatal("foreign profile accepted")
	}
}

func TestMessageContentDoesNotExecuteOrLoadRemoteHTML(t *testing.T) {
	content, links := messageContent(`<p>第一段 <b>内容</b></p><script>secret()</script><iframe>hidden</iframe><p>第二段</p><a href="javascript:alert(1)">不可执行</a><a href="https://example.edu/notice">学校链接</a><img src="https://example.edu/image.png" onerror="secret()">`)
	if !strings.Contains(content, "第一段 内容\n\n第二段") || strings.Contains(content, "secret") || strings.Contains(content, "hidden") || strings.Contains(content, "<") {
		t.Fatalf("unsafe or unreadable content: %q", content)
	}
	if len(links) != 2 || links[0].URL != "https://example.edu/notice" {
		t.Fatalf("unsafe link projection: %#v", links)
	}
}

func TestMessagesSortByPublishTimeAndKeepNumericIDs(t *testing.T) {
	d := normalize("messages", map[string]any{"list": []any{
		map[string]any{"id": json.Number("1234567890123456789"), "publishTime": "2026-09-18 12:00:00"},
		map[string]any{"id": float64(12345678), "publishTime": "2026-09-19 10:00:00"},
	}})
	if len(d.Messages) != 2 || d.Messages[0].ID != "12345678" || d.Messages[1].ID != "1234567890123456789" {
		t.Fatal("message order or identifier precision changed")
	}
}

func TestProviderMessageQueryAndPermission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/rt/onetongji/msg_detail" || r.URL.Query().Get("id") != "123" || r.Header.Get("Authorization") != "Bearer test" {
			t.Error("invalid detail request")
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	p := NewProvider(Config{})
	p.Base = server.URL
	if _, e := p.Message(context.Background(), "123", "test"); !errors.Is(e, ErrPermission) {
		t.Fatal("missing detail scope should request updated authorization")
	}
}
