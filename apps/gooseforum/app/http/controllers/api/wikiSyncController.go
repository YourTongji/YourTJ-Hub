package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
	"github.com/gin-gonic/gin"
)

// WikiSyncStatus 返回 wiki 同步面板状态（PageManager/Admin）。
func WikiSyncStatus(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(wikiservice.BuildSyncStatus())
}

// WikiSyncRun 手动触发一次 wiki 同步（PageManager/Admin）。
// 幂等：重复触发/并发触发安全（防重入锁；内容 hash 不变零变更）。
func WikiSyncRun(req component.BetterRequest[component.Null]) component.Response {
	result, err := wikiservice.Sync("manual")
	if err != nil {
		if strings.Contains(err.Error(), "already running") {
			return component.FailResponseCode(component.MessageWikiSyncRunning, nil)
		}
		slog.Error("wiki sync failed", "error", err)
		return component.FailResponseCode(component.MessageWikiSyncFailed, nil)
	}
	return component.SuccessResponse(result)
}

// WikiSyncRuns 返回最近同步运行日志（PageManager/Admin）。
func WikiSyncRuns(req component.BetterRequest[component.Null]) component.Response {
	runs := wikiSyncRuns.ListRecent(20)
	views := make([]wikiservice.SyncRunView, 0, len(runs))
	for _, r := range runs {
		views = append(views, wikiservice.ToRunView(r))
	}
	return component.SuccessResponse(views)
}

// WikiWebhookReq GitHub webhook 请求体（仅校验事件，不解析内容）。
type WikiWebhookReq struct{}

// WikiWebhook GitHub webhook 端点：PR merge 后触发即时同步。
// 安全：X-Hub-Signature-256 = HMAC-SHA256(webhook_secret, rawBody)，
// 与 GitHub 文档一致；secret 未配置时拒绝（fail-closed）。
func WikiWebhook(c *gin.Context) {
	secret := preferences.GetString("wiki.git.webhook_secret", "")
	if secret == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "webhook not configured"})
		return
	}
	sig := c.GetHeader("X-Hub-Signature-256")
	event := c.GetHeader("X-GitHub-Event")
	if !strings.HasPrefix(sig, "sha256=") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body failed"})
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature mismatch"})
		return
	}
	// 只在 push 到默认分支时同步（PR merge = push）。
	if event == "push" {
		go func() {
			if _, err := wikiservice.Sync("webhook"); err != nil {
				slog.Warn("wiki webhook sync failed", "error", err)
			}
		}()
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
