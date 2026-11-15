// Package aiservice shares the configured AI summary provider across explicit AI actions.
package aiservice

import (
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/llmprovider"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// 共享生成配额（issue #1284 #35）：课程 AI 总结与校历 AI 解析等显式 AI 动作
// 消耗同一个全局每分钟窗口。key/窗口/默认上限只在这里定义一次，
// 避免各 service 复制粘贴后上限漂移。
const (
	// GlobalGenerationKey 全局生成配额的限流 key。
	GlobalGenerationKey = "ai.summary.global"
	// GlobalGenerationWindow 全局生成配额窗口（每分钟）。
	GlobalGenerationWindow = time.Minute
	// defaultGlobalPerMinute 全局每分钟生成上限的默认值（成本护栏，
	// 实际值可经 pageConfig aiSummarySettings.globalPerMinute 热调整）。
	defaultGlobalPerMinute = 5
)

// ResolveGlobalPerMinute 解析全局每分钟生成上限：管理端热配置优先，
// 未配置（<=0）时用默认值。所有共享该配额的调用方必须经此取值。
func ResolveGlobalPerMinute(configured int) int {
	if configured <= 0 {
		return defaultGlobalPerMinute
	}
	return configured
}

// SummaryConfig resolves the admin configuration (including decrypted credentials),
// falling back to deployment configuration. Secrets never leave the server.
func SummaryConfig() llmprovider.Config {
	aiCfg := hotdataserve.GetAiSummarySettingsConfigCache()
	if strings.TrimSpace(aiCfg.BaseURL) != "" || strings.TrimSpace(aiCfg.Model) != "" {
		return llmprovider.Config{
			BaseURL:     strings.TrimRight(aiCfg.BaseURL, "/"),
			APIKey:      aiCfg.APIKey,
			Model:       aiCfg.Model,
			Temperature: float64Or(aiCfg.Temperature, 0.3),
			MaxTokens:   intOr(aiCfg.MaxTokens, 1024),
		}
	}
	return llmprovider.LoadConfig()
}

func float64Or(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}

func intOr(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}
