package httpnotifyservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// astrbotChannel AstrBot 推送插件 astrbot_plugin_push_lite（issue #1049）：向插件的
// POST /send 发送 {content, umo, message_type: "text"}，由 AstrBot 转发到 umo 指定的
// 会话（QQ 群、私聊等，umo 即在目标会话发送 /sid 得到的 SID）。插件无需改动：
// endpoint.Target 为 umo，endpoint.Secret 为插件配置里的 API token。
// 插件只把消息放进队列就返回 queued，真正发到聊天平台失败时这里无法得知。
type astrbotChannel struct{}

func (astrbotChannel) supportsEvent(string) bool { return true }

func (astrbotChannel) requiresApproval() bool { return false }

func (astrbotChannel) encode(endpoint pageConfig.HttpNotifyEndpoint, alt Alternative, approval *ApprovalPayload, _ time.Time) ([]byte, error) {
	if approval != nil {
		return astrbotBody(endpoint, astrbotApprovalText(*approval, false))
	}
	return astrbotBody(endpoint, astrbotEventText(alt))
}

// testBody 发送一条示例举报文本，与真实审批消息同一排版，链接只指向版主工作台。
func (astrbotChannel) testBody(endpoint pageConfig.HttpNotifyEndpoint, baseURI string, now time.Time) ([]byte, error) {
	sample := sampleApproval(baseURI, now)
	return astrbotBody(endpoint, astrbotApprovalText(sample, true))
}

func astrbotBody(endpoint pageConfig.HttpNotifyEndpoint, content string) ([]byte, error) {
	umo := strings.TrimSpace(endpoint.Target)
	if umo == "" {
		return nil, errors.New("target session (SID) is required")
	}
	return json.Marshal(map[string]string{
		"content":      content,
		"umo":          umo,
		"message_type": "text",
	})
}

// buildRequest 插件按 Authorization 头整串比对 "Bearer <token>"。URL 只填到端口
// （没有路径）时补上 /send，省去管理员记接口路径。
func (astrbotChannel) buildRequest(endpoint pageConfig.HttpNotifyEndpoint, _ string, _ string, _ int64, body []byte) (*http.Request, error) {
	req, err := newJSONPost(endpoint, body)
	if err != nil {
		return nil, err
	}
	if req.URL.Path == "" || req.URL.Path == "/" {
		req.URL.Path = "/send"
	}
	if endpoint.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+endpoint.Secret)
	}
	return req, nil
}

// checkResponse 2xx 且返回 {"status":"queued"} 才算成功；插件的 400/403 带
// {"error","details"}，details 是可读原因（如 "403 Forbidden: 无效令牌"）。
func (astrbotChannel) checkResponse(resp *http.Response) error {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("astrbot response read failed: %w", err)
	}
	var result struct {
		Status  string `json:"status"`
		Details string `json:"details"`
	}
	parsed := json.Unmarshal(raw, &result) == nil
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed && result.Details != "" {
			return errors.New(SafeText(result.Details, 160))
		}
		return errors.New(resp.Status)
	}
	if !parsed {
		return errors.New("astrbot response is not JSON (is the URL the push_lite /send endpoint?)")
	}
	if result.Status != "queued" {
		return fmt.Errorf("astrbot response status %q, want \"queued\"", SafeText(result.Status, 40))
	}
	return nil
}

// astrbotApprovalText 审批摘要的纯文本排版：首行【种类 · 目标】+ 标签，随后编号、
// 标题、摘要、元信息、举报说明与处理链接。用户文本已经 SafeText 折叠为单行，
// 不会伪造出额外的行或链接说明。
func astrbotApprovalText(payload ApprovalPayload, test bool) string {
	a := payload.Approval
	target := approvalTargetLabels[a.TargetType]
	if target == "" {
		target = SafeText(a.TargetType, 20)
	}
	kind := "待人工审核"
	if a.Kind == ApprovalKindReport {
		kind = "新举报"
	}
	tags := make([]string, 0, 4)
	if reason := approvalReasonLabels[a.Reason]; reason != "" {
		tags = append(tags, reason)
	} else if a.Reason != "" {
		tags = append(tags, SafeText(a.Reason, 20))
	}
	if a.Edited {
		tags = append(tags, "编辑后")
	}
	if a.Anonymous {
		tags = append(tags, "匿名")
	}

	var b strings.Builder
	if test {
		b.WriteString("【测试】")
	}
	b.WriteString("【" + kind + " · " + target + "】")
	b.WriteString(strings.Join(tags, " · "))
	line := func(label, value string) {
		if value != "" {
			b.WriteString("\n" + label + "：" + value)
		}
	}
	ids := target + " #" + strconv.FormatUint(a.TargetID, 10)
	if a.ReportID > 0 {
		ids = "举报 #" + strconv.FormatUint(a.ReportID, 10) + " · " + ids
	}
	line("编号", ids)
	line("标题", SafeText(a.Title, 100))
	line("摘要", SafeText(a.Excerpt, 200))
	switch {
	case a.Anonymous:
		line("作者", "匿名（不披露身份）")
	case a.Author != nil:
		line("作者", SafeText(a.Author.DisplayName, 40))
	}
	if len(a.Categories) > 0 {
		line("分类", SafeText(strings.Join(a.Categories, "、"), 40))
	}
	line("时间", feishuTime(a.CreatedAt))
	line("举报说明", SafeText(a.Note, 300))
	if a.TargetType == "chat_message" {
		b.WriteString("\n私信内容不会发到外部，请在版主工作台查看证据并处理。")
	}

	base := strings.TrimRight(payload.BaseURI, "/")
	if base == "" {
		b.WriteString("\n\n站点地址未配置，无法生成处理链接，请在版主工作台处理。")
		return b.String()
	}
	// 文本里链接不能像按钮那样区分样式：与工作台同址的动作（如测试消息）只保留工作台一行。
	links := make([]string, 0, len(a.Actions)+1)
	for _, action := range a.Actions {
		if label := approvalActionLabels[action.Action]; label != "" && action.URL != "" && action.URL != a.ModerationURL {
			links = append(links, label+"："+base+action.URL)
		}
	}
	if a.ModerationURL != "" {
		links = append(links, "版主工作台："+base+a.ModerationURL)
	}
	if len(links) > 0 {
		b.WriteString("\n\n" + strings.Join(links, "\n"))
	}
	switch {
	case test:
		b.WriteString("\n（测试消息，内容仅为示例；链接只会打开版主工作台）")
	case len(a.Actions) > 0:
		b.WriteString("\n（链接会先打开本站确认页，登录且有审核权限才能处理）")
	}
	return b.String()
}

var astrbotEventTitles = map[string]string{
	EventTopicPublished: "新话题",
	EventTopicUpdated:   "话题已更新",
	EventCommentCreated: "新回复",
	EventUserSignup:     "新用户注册",
	EventReportCreated:  "新举报",
}

// astrbotEventData 通用事件负载中用于文本排版的字段（与 eventhandlers 的 JSON 形状一致）。
type astrbotEventData struct {
	BaseURI        string `json:"baseUri"`
	ContentPreview string `json:"contentPreview"`
	Topic          *struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
		User        struct {
			DisplayName string `json:"displayName"`
		} `json:"user"`
		Categories []struct {
			Name string `json:"name"`
		} `json:"categories"`
	} `json:"topic"`
	User *struct {
		DisplayName string `json:"displayName"`
		Username    string `json:"username"`
		URL         string `json:"url"`
	} `json:"user"`
	Post *struct {
		URL string `json:"url"`
	} `json:"post"`
}

// astrbotEventText 非审批事件（新话题、新回复、新用户等）的纯文本：标题行 + 关键字段 + 链接。
func astrbotEventText(alt Alternative) string {
	title := astrbotEventTitles[alt.Event]
	if title == "" {
		title = SafeText(alt.Event, 60)
	}
	var data astrbotEventData
	if raw, err := json.Marshal(alt.Data); err == nil {
		_ = json.Unmarshal(raw, &data)
	}
	base := strings.TrimRight(data.BaseURI, "/")
	absolute := func(path string) string {
		if path == "" || base == "" || !strings.HasPrefix(path, "/") {
			return ""
		}
		return base + path
	}

	var b strings.Builder
	b.WriteString("【" + title + "】")
	line := func(label, value string) {
		if value != "" {
			b.WriteString("\n" + label + "：" + value)
		}
	}
	link := ""
	if t := data.Topic; t != nil {
		b.WriteString(SafeText(t.Title, 100))
		if alt.Event != EventCommentCreated {
			line("作者", SafeText(t.User.DisplayName, 40))
			names := make([]string, 0, len(t.Categories))
			for _, c := range t.Categories {
				names = append(names, c.Name)
			}
			line("分类", SafeText(strings.Join(names, "、"), 40))
			line("摘要", SafeText(t.Description, 200))
		}
		link = absolute(t.URL)
	}
	if u := data.User; u != nil {
		name := SafeText(u.DisplayName, 40)
		if username := SafeText(u.Username, 40); username != "" && username != name {
			name += "（@" + username + "）"
		}
		switch alt.Event {
		case EventCommentCreated:
			line("回复者", name)
		case EventUserSignup:
			b.WriteString(name)
			link = absolute(u.URL)
		}
	}
	line("内容", SafeText(data.ContentPreview, 200))
	if p := data.Post; p != nil && absolute(p.URL) != "" {
		link = absolute(p.URL)
	}
	if link != "" {
		b.WriteString("\n" + link)
	}
	return b.String()
}
