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
	body := map[string]any{
		"msg_type": "interactive",
		"card":     feishuCard(*approval),
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

var feishuActionLabels = map[string]string{
	ApprovalActionApprove: "✅ 通过并公开",
	ApprovalActionReject:  "❌ 拒绝",
	ApprovalActionBan:     "⛔ 封禁并结案",
	ApprovalActionHide:    "🙈 隐藏评价并结案",
	ApprovalActionDismiss: "↩️ 驳回举报",
}

// feishuCard 渲染 schema 2.0 卡片。用户可控文本一律走 plain_text（不解析 Markdown、
// 链接与 <at> 提及），避免恶意内容破坏卡片或 @ 全员。
func feishuCard(payload ApprovalPayload) map[string]any {
	a := payload.Approval
	target := feishuTargetLabels[a.TargetType]
	if target == "" {
		target = a.TargetType
	}
	title := "待人工审核 · " + target
	template := "orange"
	if a.Kind == ApprovalKindReport {
		title = "新举报 · " + target
		template = "red"
	}
	if a.Edited {
		title += "（编辑后）"
	}
	reason := feishuReasonLabels[a.Reason]
	if reason == "" {
		reason = SafeText(a.Reason, 40)
	}

	elements := make([]map[string]any, 0, 16)
	addLine := func(label, value string) {
		if value == "" {
			return
		}
		elements = append(elements, feishuText(label+"："+value))
	}
	addLine("标题", SafeText(a.Title, 100))
	addLine("内容摘要", SafeText(a.Excerpt, 200))
	switch {
	case a.Anonymous:
		addLine("作者", "匿名（身份不在通知中披露）")
	case a.Author != nil:
		addLine("作者", SafeText(a.Author.DisplayName, 60))
	}
	if len(a.Categories) > 0 {
		addLine("分类", SafeText(strings.Join(a.Categories, "、"), 80))
	}
	addLine("举报说明", SafeText(a.Note, 300))
	if a.TargetType == "chat_message" {
		elements = append(elements, feishuText("私信内容不会发送到外部群，请在版主工作台查看证据并处理。"))
	}
	meta := target + " #" + strconv.FormatUint(a.TargetID, 10)
	if a.ReportID > 0 {
		meta = "举报 #" + strconv.FormatUint(a.ReportID, 10) + " · " + meta
	}
	if a.CreatedAt != "" {
		meta += " · " + a.CreatedAt
	}
	elements = append(elements, feishuText(meta))

	base := strings.TrimRight(payload.BaseURI, "/")
	if base != "" {
		elements = append(elements, map[string]any{"tag": "hr"})
		for _, action := range a.Actions {
			label := feishuActionLabels[action.Action]
			if label == "" || action.URL == "" {
				continue
			}
			buttonType := "default"
			switch action.Action {
			case ApprovalActionApprove:
				buttonType = "primary"
			case ApprovalActionBan, ApprovalActionHide, ApprovalActionReject:
				buttonType = "danger"
			}
			elements = append(elements, feishuButton(label, buttonType, base+action.URL))
		}
		if a.ModerationURL != "" {
			elements = append(elements, feishuButton("打开版主工作台", "default", base+a.ModerationURL))
		}
		if len(a.Actions) > 0 {
			elements = append(elements, feishuText("快捷按钮会先打开 Hub 确认页：需登录且具备该内容的审核权限，提交后才会执行。"))
		}
	} else {
		elements = append(elements, feishuText("站点地址未配置，无法生成处理链接，请在版主工作台处理。"))
	}

	header := map[string]any{
		"template": template,
		"title":    map[string]any{"tag": "plain_text", "content": title},
	}
	if reason != "" {
		header["subtitle"] = map[string]any{"tag": "plain_text", "content": reason}
	}
	return map[string]any{
		"schema": "2.0",
		"config": map[string]any{
			"update_multi": true,
			"summary":      map[string]any{"content": title},
		},
		"header": header,
		"body": map[string]any{
			"direction": "vertical",
			"elements":  elements,
		},
	}
}

func feishuText(content string) map[string]any {
	return map[string]any{"tag": "div", "text": map[string]any{"tag": "plain_text", "content": content}}
}

func feishuButton(label, buttonType, url string) map[string]any {
	return map[string]any{
		"tag":       "button",
		"type":      buttonType,
		"text":      map[string]any{"tag": "plain_text", "content": label},
		"behaviors": []map[string]any{{"type": "open_url", "default_url": url}},
	}
}
