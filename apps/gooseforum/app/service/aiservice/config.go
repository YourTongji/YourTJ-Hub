// Package aiservice shares the configured AI summary provider across explicit AI actions.
package aiservice

import (
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/llmprovider"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

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
