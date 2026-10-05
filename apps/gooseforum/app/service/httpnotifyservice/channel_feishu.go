package httpnotifyservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

// feishuChannel 飞书群自定义机器人（issue #1049）：无需注册飞书应用，直接向群 webhook
// 发送 schema 2.0 interactive 卡片。自定义机器人卡片只支持 open_url 按钮，因此快捷
// 动作是指向 Hub 确认页的签名链接，最终处理仍在 Hub 内完成登录与权限复核。
type feishuChannel struct{}

func (feishuChannel) supportsEvent(eventName string) bool { return IsApprovalEvent(eventName) }

func (feishuChannel) requiresApproval() bool { return true }

func (feishuChannel) encode(endpoint pageConfig.HttpNotifyEndpoint, _ Alternative, approval *ApprovalPayload, now time.Time) ([]byte, error) {
	if approval == nil {
		return nil, errors.New("feishu channel requires an approval payload")
	}
	return feishuBody(endpoint, feishuCard(*approval, false), now)
}

// testBody 发送一张示例举报卡片：样式与真实审批卡片一致，按钮只打开版主工作台。
func (feishuChannel) testBody(endpoint pageConfig.HttpNotifyEndpoint, baseURI string, now time.Time) ([]byte, error) {
	sample := ApprovalPayload{BaseURI: baseURI, Approval: Approval{
		ID: "test", Kind: ApprovalKindReport, TargetType: "post", TargetID: 1, ReportID: 1,
		Reason: "spam", Title: "示例话题：期中复习资料汇总",
		Excerpt:       "这是一条示例回复，用来预览审批卡片在群里的样子，不对应站内的真实内容。",
		Author:        &ApprovalAuthor{DisplayName: "示例用户"},
		Categories:    []string{"学习交流"},
		Note:          "示例举报说明：疑似广告引流。",
		CreatedAt:     now.Format(time.RFC3339),
		ModerationURL: "/moderation?tab=reports",
	}}
	sample.Approval.Actions = []ApprovalAction{
		{Action: ApprovalActionBan, URL: sample.Approval.ModerationURL},
		{Action: ApprovalActionDismiss, URL: sample.Approval.ModerationURL},
	}
	return feishuBody(endpoint, feishuCard(sample, true), now)
}

func feishuBody(endpoint pageConfig.HttpNotifyEndpoint, card map[string]any, now time.Time) ([]byte, error) {
	body := map[string]any{
		"msg_type": "interactive",
		"card":     card,
	}
	// 开启飞书“签名校验”时，请求体携带 timestamp 与 sign（飞书要求一小时内）。
	if endpoint.Secret != "" {
		timestamp := strconv.FormatInt(now.Unix(), 10)
		body["timestamp"] = timestamp
		body["sign"] = feishuSign(timestamp, endpoint.Secret)
	}
	return json.Marshal(body)
}

func (feishuChannel) buildRequest(endpoint pageConfig.HttpNotifyEndpoint, _ string, _ string, _ int64, body []byte) (*http.Request, error) {
	return newJSONPost(endpoint, body)
}

// checkResponse 飞书常以 HTTP 200 返回业务失败（签名错误、频控、关键词不匹配等），
// 必须解析 body：code（或旧版 StatusCode）非 0、或返回体无法识别都计为失败。
func (feishuChannel) checkResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("feishu response read failed: %w", err)
	}
	var result struct {
		Code          *int   `json:"code"`
		Msg           string `json:"msg"`
		StatusCode    *int   `json:"StatusCode"`
		StatusMessage string `json:"StatusMessage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return errors.New("feishu response is not JSON")
	}
	switch {
	case result.Code != nil && *result.Code != 0:
		return fmt.Errorf("feishu code %d: %s", *result.Code, SafeText(result.Msg, 120))
	case result.Code != nil:
		return nil
	case result.StatusCode != nil && *result.StatusCode != 0:
		return fmt.Errorf("feishu code %d: %s", *result.StatusCode, SafeText(result.StatusMessage, 120))
	case result.StatusCode != nil:
		return nil
	default:
		return errors.New("feishu response missing code")
	}
}

// feishuSign 飞书自定义机器人签名：以 timestamp+"\n"+secret 为 HMAC-SHA256 密钥、
// 空串为消息，结果 base64。
func feishuSign(timestamp string, secret string) string {
	mac := hmac.New(sha256.New, []byte(timestamp+"\n"+secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

var feishuTargetLabels = map[string]string{
	"topic":         "话题",
	"post":          "回复",
	"chat_message":  "私信",
	"course_review": "课程评价",
}

var feishuReasonLabels = map[string]string{
	"spam":           "垃圾广告",
	"abuse":          "辱骂/人身攻击",
	"illegal":        "违法违规",
	"irrelevant":     "无关内容",
	"other":          "其他",
	"sensitive_word": "命中敏感词",
	"ai":             "AI 审核存疑",
}

// feishuActionStyle 快捷按钮的文案、样式与图标。图标 token 均取自飞书卡片图标库
// （https://open.feishu.cn/document/feishu-cards/enumerations-for-icons），
// 写错的 token 在客户端静默不显示，新增时须逐个核对。
type feishuActionStyle struct {
	label      string
	buttonType string
	icon       string
}

var feishuActionStyles = map[string]feishuActionStyle{
	ApprovalActionApprove: {"通过并公开", "primary_filled", "yes_outlined"},
	ApprovalActionReject:  {"拒绝", "danger", "no_outlined"},
	ApprovalActionBan:     {"封禁并结案", "danger", "ban_outlined"},
	ApprovalActionHide:    {"隐藏评价并结案", "danger", "invisible_outlined"},
	ApprovalActionDismiss: {"驳回举报", "default", "close_outlined"},
}

// feishuCard 渲染 schema 2.0 卡片：标题栏（图标、编号、最多 3 个标签）→ 内容标题 →
// 灰底摘要 → 两栏元信息 → 操作按钮 → 灰色说明。用户可控文本一律走 plain_text
// （不解析 Markdown、链接与 <at> 提及），避免恶意内容破坏卡片或 @ 全员。test 为管理端
// 「测试发送」的示例卡片：样式与真实卡片一致，按钮只打开版主工作台。
func feishuCard(payload ApprovalPayload, test bool) map[string]any {
	a := payload.Approval
	target := feishuTargetLabels[a.TargetType]
	if target == "" {
		target = SafeText(a.TargetType, 20)
	}
	title, template, headerIcon, reasonColor := "待人工审核 · "+target, "orange", "approval_outlined", "orange"
	if a.Kind == ApprovalKindReport {
		title, template, headerIcon, reasonColor = "新举报 · "+target, "red", "report_outlined", "red"
	}
	reason := feishuReasonLabels[a.Reason]
	if reason == "" {
		reason = SafeText(a.Reason, 20)
	}

	tags := make([]map[string]any, 0, 3)
	addTag := func(text, color string) {
		if text != "" && len(tags) < 3 {
			tags = append(tags, map[string]any{"tag": "text_tag", "text": feishuPlain(text), "color": color})
		}
	}
	if test {
		addTag("测试", "blue")
	}
	addTag(reason, reasonColor)
	if a.Edited {
		addTag("编辑后", "wathet")
	}
	if a.Anonymous {
		addTag("匿名", "neutral")
	}

	subtitle := target + " #" + strconv.FormatUint(a.TargetID, 10)
	if a.ReportID > 0 {
		subtitle = "举报 #" + strconv.FormatUint(a.ReportID, 10) + " · " + subtitle
	}
	header := map[string]any{
		"template": template,
		"title":    feishuPlain(title),
		"subtitle": feishuPlain(subtitle),
		"icon":     feishuIcon(headerIcon, template),
	}
	if len(tags) > 0 {
		header["text_tag_list"] = tags
	}

	elements := make([]map[string]any, 0, 8)
	if heading := SafeText(a.Title, 100); heading != "" {
		elements = append(elements, map[string]any{
			"tag":  "div",
			"text": map[string]any{"tag": "plain_text", "content": heading, "text_size": "heading", "lines": 2},
		})
	}
	if excerpt := SafeText(a.Excerpt, 200); excerpt != "" {
		elements = append(elements, map[string]any{
			"tag":              "column_set",
			"background_style": "grey-50",
			"columns": []map[string]any{{
				"tag": "column", "width": "weighted", "weight": 1, "padding": "8px 12px 8px 12px",
				"elements": []map[string]any{{
					"tag":  "div",
					"text": map[string]any{"tag": "plain_text", "content": excerpt, "lines": 4},
				}},
			}},
		})
	}

	author := ""
	switch {
	case a.Anonymous:
		author = "匿名（不披露身份）"
	case a.Author != nil:
		author = SafeText(a.Author.DisplayName, 40)
	}
	fields := make([]map[string]any, 0, 3)
	if author != "" {
		fields = append(fields, feishuField("member_outlined", "作者", author))
	}
	if len(a.Categories) > 0 {
		fields = append(fields, feishuField("tag_outlined", "分类", SafeText(strings.Join(a.Categories, "、"), 40)))
	}
	if created := feishuTime(a.CreatedAt); created != "" {
		fields = append(fields, feishuField("time_outlined", "时间", created))
	}
	if len(fields) == 1 {
		elements = append(elements, fields[0])
	} else if len(fields) > 1 {
		// 两栏交替排布元信息；bisect 在窄屏仍保持两栏等分。
		var left, right []map[string]any
		for i, field := range fields {
			if i%2 == 0 {
				left = append(left, field)
			} else {
				right = append(right, field)
			}
		}
		elements = append(elements, map[string]any{
			"tag":                "column_set",
			"flex_mode":          "bisect",
			"horizontal_spacing": "12px",
			"columns": []map[string]any{
				{"tag": "column", "width": "weighted", "weight": 1, "elements": left},
				{"tag": "column", "width": "weighted", "weight": 1, "elements": right},
			},
		})
	}
	if note := SafeText(a.Note, 300); note != "" {
		elements = append(elements, feishuField("chat_outlined", "举报说明", note))
	}
	if a.TargetType == "chat_message" {
		elements = append(elements, feishuNote("lock_outlined", "私信内容不会发到外部群，请在版主工作台查看证据并处理。"))
	}

	base := strings.TrimRight(payload.BaseURI, "/")
	if base == "" {
		elements = append(elements, feishuNote("warning_outlined", "站点地址未配置，无法生成处理链接，请在版主工作台处理。"))
	} else {
		buttons := make([]map[string]any, 0, len(a.Actions)+1)
		for _, action := range a.Actions {
			style, ok := feishuActionStyles[action.Action]
			if !ok || action.URL == "" {
				continue
			}
			buttons = append(buttons, feishuButton(style.label, style.buttonType, style.icon, base+action.URL))
		}
		if a.ModerationURL != "" {
			buttons = append(buttons, feishuButton("打开版主工作台", "default", "admin_outlined", base+a.ModerationURL))
		}
		if len(buttons) > 0 {
			columns := make([]map[string]any, 0, len(buttons))
			for _, button := range buttons {
				columns = append(columns, map[string]any{"tag": "column", "width": "auto", "elements": []map[string]any{button}})
			}
			// flow：窄屏（手机）放不下时按钮自动换行，而不是被压缩。
			elements = append(elements, map[string]any{
				"tag": "column_set", "flex_mode": "flow", "horizontal_spacing": "8px", "columns": columns,
			})
		}
		switch {
		case test:
			elements = append(elements, feishuNote("info_outlined", "这是一条测试消息，内容仅为示例；按钮只会打开版主工作台。"))
		case len(a.Actions) > 0:
			elements = append(elements, feishuNote("safe_outlined", "按钮会先打开本站确认页，登录且有审核权限才能处理。"))
		}
	}

	summary := title
	if a.Title != "" {
		summary += "：" + SafeText(a.Title, 40)
	}
	return map[string]any{
		"schema": "2.0",
		"config": map[string]any{
			"update_multi": true,
			"summary":      map[string]any{"content": summary},
		},
		"header": header,
		"body": map[string]any{
			"direction": "vertical",
			"elements":  elements,
		},
	}
}

// feishuTime 把 RFC3339 时间转为服务器本地时区的「2006-01-02 15:04」，无法解析时原样截断。
func feishuTime(value string) string {
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.In(time.Local).Format("2006-01-02 15:04")
	}
	return SafeText(value, 40)
}

func feishuPlain(content string) map[string]any {
	return map[string]any{"tag": "plain_text", "content": content}
}

// feishuIcon 图标库图标；color 为空时继承所在组件（如按钮文字）的颜色。
func feishuIcon(token, color string) map[string]any {
	icon := map[string]any{"tag": "standard_icon", "token": token}
	if color != "" {
		icon["color"] = color
	}
	return icon
}

// feishuField 带图标的「标签：值」一行。
func feishuField(icon, label, value string) map[string]any {
	return map[string]any{
		"tag":  "div",
		"icon": feishuIcon(icon, "grey"),
		"text": map[string]any{"tag": "plain_text", "content": label + "：" + value, "lines": 3},
	}
}

// feishuNote 灰色小字说明（2.0 移除了 note 组件，官方建议用 notation 字号的文本代替）。
func feishuNote(icon, content string) map[string]any {
	return map[string]any{
		"tag":  "div",
		"icon": feishuIcon(icon, "grey"),
		"text": map[string]any{"tag": "plain_text", "content": content, "text_size": "notation", "text_color": "grey"},
	}
}

func feishuButton(label, buttonType, icon, url string) map[string]any {
	return map[string]any{
		"tag":       "button",
		"type":      buttonType,
		"size":      "medium",
		"text":      feishuPlain(label),
		"icon":      feishuIcon(icon, ""),
		"behaviors": []map[string]any{{"type": "open_url", "default_url": url}},
	}
}
