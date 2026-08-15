package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
	"github.com/gin-gonic/gin"
)

// WikiSyncStatus 返回 wiki 同步面板状态（PageManager/Admin）。
func WikiSyncStatus(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(wikiservice.BuildSyncStatus())
}

// WikiSyncRun 手动触发一次 wiki 同步（PageManager/Admin）。
// 同步异步执行：git clone/fetch + 全量投影可能超过 HTTP WriteTimeout（10s），
// 同步执行会让管理端看到失败响应并可能重复触发。端点立即返回 accepted，
// 进度由 /sync/status 与 /sync/runs 轮询（run 行 status=running 开始）。
func WikiSyncRun(req component.BetterRequest[component.Null]) component.Response {
	if !wikiservice.LoadGitConfig().Enabled() {
		return component.FailResponseCode(component.MessageWikiSyncFailed, nil)
	}
	go func() {
		if _, err := wikiservice.Sync("manual"); err != nil {
			if errors.Is(err, wikiservice.ErrSyncAlreadyRunning) {
				return // 已合并：运行中同步完成后会自动补跑（syncPending）
			}
			slog.Error("wiki sync failed", "error", err)
		}
	}()
	return component.SuccessResponse(wikiservice.SyncAccepted{Accepted: true})
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

// GetWikiWebhookSecret 返回 webhook 验签密钥配置状态（PageManager/Admin）。
// 仅回显是否已配置（securestore 密文不回显、不解密）；兼容旧
// config.toml [wiki.git].webhook_secret 明文配置（已配置也算 configured）。
func GetWikiWebhookSecret(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(map[string]any{
		"configured": wikiservice.LoadWebhookSecret() != "",
	})
}

// SaveWikiWebhookSecretReq 保存 webhook 验签密钥请求。
type SaveWikiWebhookSecretReq struct {
	// Secret GitHub webhook 验签密钥（明文，仅在保存瞬间存在）；留空表示清除已存密钥。
	Secret string `json:"secret" validate:"max=1024"`
}

// SaveWikiWebhookSecret 保存 GitHub webhook 验签密钥（PageManager/Admin）：
// securestore 加密后落库（密文经 WikiSyncSettingsStorage 持久化，领域结构
// json:"-" 防导出泄露），明文不持久化。清除时传空字符串。
func SaveWikiWebhookSecret(req component.BetterRequest[SaveWikiWebhookSecretReq]) component.Response {
	encrypted := ""
	if secret := strings.TrimSpace(req.Params.Secret); secret != "" {
		sealed, err := securestore.EncryptPurpose(secret, securestore.WikiWebhookSecretPurpose)
		if err != nil {
			return component.FailResponseError(fmt.Errorf("加密 wiki webhook secret 失败（请确认 app.signingKey 已配置）：%w", err))
		}
		encrypted = sealed
	}
	entity := pageConfig.GetByPageType(pageConfig.WikiSyncSettings)
	entity.PageType = pageConfig.WikiSyncSettings
	entity.Config = jsonopt.Encode(pageConfig.WikiSyncSettingsStorage{WebhookSecretEncrypted: encrypted})
	pageConfig.CreateOrSave(&entity)
	hotdataserve.ClearWikiSyncSettingsConfigCache()
	return component.SuccessResponse(wikiservice.ActionResult{Ok: true})
}

// WikiWebhookReq GitHub webhook 请求体（仅校验事件，不解析内容）。
type WikiWebhookReq struct{}

// maxWebhookBodyBytes 限制 webhook 请求体大小：该端点无需 JWT（公开），
// 服务器无全局 body 上限，不设限可被未认证调用方以任意大 body 耗尽内存。
// 同步只读取 body 中的 ref，5MB 远大于正常 push 负载且足以解析分支。
const maxWebhookBodyBytes = 5 << 20

// WikiWebhook GitHub webhook 端点：PR merge 后触发即时同步。
// 安全：X-Hub-Signature-256 = HMAC-SHA256(webhook_secret, rawBody)，
// 与 GitHub 文档一致；secret 未配置时拒绝（fail-closed）。
// secret 来源：管理端设置（securestore 加密落库）优先，
// 兼容旧 config.toml [wiki.git].webhook_secret 明文配置。
func WikiWebhook(c *gin.Context) {
	secret := wikiservice.LoadWebhookSecret()
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
	// 限流请求体（验签前也受限：签名前缀错误的超大 body 同样不能进入内存）。
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodyBytes)
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
	// 只同步默认分支的 push（PR merge = push；其他分支/删除分支的 push 忽略）。
	if event == "push" {
		var payload struct {
			Ref string `json:"ref"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		branch := wikiservice.LoadGitConfig().Branch
		if payload.Ref != "" && payload.Ref != "refs/heads/"+branch {
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
		go func() {
			if _, err := wikiservice.Sync("webhook"); err != nil {
				if errors.Is(err, wikiservice.ErrSyncAlreadyRunning) {
					return // 已合并：运行中同步完成后自动补跑
				}
				slog.Warn("wiki webhook sync failed", "error", err)
			}
		}()
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
