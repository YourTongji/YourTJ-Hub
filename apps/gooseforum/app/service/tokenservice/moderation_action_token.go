package tokenservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// 版主快捷审批链接（issue #1049）：飞书等通知卡片里的按钮只携带签名 token，
// 打开 Hub 确认页后由登录用户显式提交。token 只证明“链接由本站签发且未被篡改、
// 仍在有效期内”，绝不代表点击者有审核权限——执行时必须按真实会话重新授权。

// ModerationActionTTL 快捷审批链接有效期。
const ModerationActionTTL = 24 * time.Hour

// 快捷审批的目标种类。
const (
	ModerationActionSubjectReviewTopic = "review.topic" // 待人工审核的话题（ID = topic id）
	ModerationActionSubjectReviewPost  = "review.post"  // 待人工审核的回复（ID = post id）
	ModerationActionSubjectReport      = "report"       // 待处理举报（ID = report id）
)

// moderationActionPurpose 派生签名密钥的用途标签：与重置密码/激活 JWT（直接用
// app.signingKey）隔离，任何一方的 token 都不能被另一方接受。
const moderationActionPurpose = "yourtj-moderation-action-token"

const moderationActionPrefix = "v1."

var (
	// ErrModerationActionInvalid token 结构、签名或字段非法。
	ErrModerationActionInvalid = errors.New("moderation action token invalid")
	// ErrModerationActionExpired token 签名有效但已过期（返回的 claims 仍可用于引导到工作台）。
	ErrModerationActionExpired = errors.New("moderation action token expired")
)

// ModerationActionClaims 快捷审批链接绑定的目标、动作与版本。
// Version 对待审内容是送审版本 ID（post_revisions.id），作者改稿后旧链接失效；
// 举报为空（举报是否仍待处理由 status 复核）。
type ModerationActionClaims struct {
	Subject string `json:"s"`
	ID      uint64 `json:"i"`
	Action  string `json:"a"`
	Version string `json:"v,omitempty"`
	Expires int64  `json:"e"`
}

// IssueModerationAction 签发快捷审批 token：v1.<base64url(payload)>.<base64url(hmac)>。
func IssueModerationAction(claims ModerationActionClaims, now time.Time) (string, error) {
	if claims.Subject == "" || claims.ID == 0 || claims.Action == "" {
		return "", ErrModerationActionInvalid
	}
	key, err := moderationActionKey()
	if err != nil {
		return "", err
	}
	claims.Expires = now.Add(ModerationActionTTL).Unix()
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return moderationActionPrefix + body + "." + base64.RawURLEncoding.EncodeToString(moderationActionMAC(key, body)), nil
}

// ParseModerationAction 校验签名（常量时间比较）与有效期。签名有效但过期时返回
// claims 与 ErrModerationActionExpired。
func ParseModerationAction(token string, now time.Time) (ModerationActionClaims, error) {
	var claims ModerationActionClaims
	rest, ok := strings.CutPrefix(strings.TrimSpace(token), moderationActionPrefix)
	if !ok {
		return claims, ErrModerationActionInvalid
	}
	body, sig, ok := strings.Cut(rest, ".")
	if !ok || body == "" || sig == "" {
		return claims, ErrModerationActionInvalid
	}
	key, err := moderationActionKey()
	if err != nil {
		return claims, err
	}
	gotMAC, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(gotMAC, moderationActionMAC(key, body)) {
		return claims, ErrModerationActionInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil || json.Unmarshal(payload, &claims) != nil || claims.Subject == "" || claims.ID == 0 || claims.Action == "" {
		return ModerationActionClaims{}, ErrModerationActionInvalid
	}
	if now.Unix() > claims.Expires {
		return claims, ErrModerationActionExpired
	}
	return claims, nil
}

func moderationActionKey() ([]byte, error) {
	base, err := signingKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, base)
	mac.Write([]byte(moderationActionPurpose))
	return mac.Sum(nil), nil
}

func moderationActionMAC(key []byte, body string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(moderationActionPrefix))
	mac.Write([]byte(body))
	return mac.Sum(nil)
}
