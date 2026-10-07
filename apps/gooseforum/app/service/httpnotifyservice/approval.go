package httpnotifyservice

import (
	"strings"
	"unicode"
)

// 审批种类与快捷动作（issue #1049）。
const (
	ApprovalKindReview = "review" // 内容真正进入人工审核队列
	ApprovalKindReport = "report" // 新举报

	ApprovalActionApprove = "approve" // 待审内容：通过并公开
	ApprovalActionReject  = "reject"  // 待审内容：拒绝
	ApprovalActionBan     = "ban"     // 举报：封禁目标并结案
	ApprovalActionHide    = "hide"    // 课评举报：隐藏评价并结案
	ApprovalActionDismiss = "dismiss" // 举报：驳回
)

// ApprovalPayload 审批事件的 generic 负载（data 字段）与卡片通道的渲染输入。
// BaseURI 用于把站内路径拼成绝对地址。
type ApprovalPayload struct {
	BaseURI  string   `json:"baseUri"`
	Approval Approval `json:"approval"`
}

// Approval 待人工审批的安全摘要：生产者已完成匿名/隐私裁剪，渲染器不得再回库
// 拼接作者等身份信息。所有 URL 字段均为站内路径。
type Approval struct {
	ID         string `json:"id"`         // 稳定审批标识（待审内容含送审版本 ID），亦为投递去重键
	Kind       string `json:"kind"`       // review | report
	TargetType string `json:"targetType"` // topic | post | chat_message | course_review
	TargetID   uint64 `json:"targetId"`
	ReportID   uint64 `json:"reportId,omitempty"`
	TopicID    uint64 `json:"topicId,omitempty"`
	// Reason 举报为原因代码（spam/abuse/...）；人工审核为触发来源（sensitive_word/ai）。
	Reason     string          `json:"reason,omitempty"`
	Note       string          `json:"note,omitempty"`
	Title      string          `json:"title,omitempty"`
	Excerpt    string          `json:"excerpt,omitempty"`
	Anonymous  bool            `json:"anonymous"`
	Author     *ApprovalAuthor `json:"author,omitempty"`
	Categories []string        `json:"categories,omitempty"`
	Edited     bool            `json:"edited,omitempty"`
	// Version 送审版本 ID；快捷动作绑定该版本，作者改稿后旧卡片失效。
	Version       string           `json:"version,omitempty"`
	CreatedAt     string           `json:"createdAt"`
	TargetURL     string           `json:"targetUrl,omitempty"`
	ModerationURL string           `json:"moderationUrl"`
	Actions       []ApprovalAction `json:"actions"`
}

// ApprovalAuthor 可公开披露的作者信息（匿名内容不输出）。
type ApprovalAuthor struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"displayName"`
	URL         string `json:"url"`
}

// ApprovalAction 快捷动作：URL 指向 Hub 确认页（GET 无副作用），登录并通过权限
// 复核后由用户显式提交。
type ApprovalAction struct {
	Action string `json:"action"`
	URL    string `json:"url"`
}

// SafeText 规范化外部展示文本：去控制字符、折叠空白并按 rune 截断，避免用户输入
// 破坏卡片结构或撑爆消息体。
func SafeText(value string, maxRunes int) string {
	var b strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if unicode.IsControl(r) || r == '\uFEFF' {
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	text := b.String()
	if runes := []rune(text); maxRunes > 0 && len(runes) > maxRunes {
		text = string(runes[:maxRunes-1]) + "…"
	}
	return text
}
