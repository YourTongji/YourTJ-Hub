package moderationservice

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// AIConnectionCheck 管理端「测试连接」单项结果（issue #975）。只含分类、状态码
// 与耗时，不回带 provider 响应原文。
type AIConnectionCheck struct {
	OK         bool   `json:"ok"`
	Kind       string `json:"kind"` // ok | uncertain | not_configured | refused | invalid | timeout | auth | http_<status> | jev_* | error
	HTTPStatus int    `json:"httpStatus,omitempty"`
	Model      string `json:"model,omitempty"`
	LatencyMs  int64  `json:"latencyMs"`
}

// TestAIVisionConnection 用内置测试图走一遍真实证据提取：返回合法证据 JSON
// （含 uncertain，纯色图被判“不确定”属正常）即视为连通且遵守了证据格式。
func TestAIVisionConnection(ctx context.Context, cfg pageConfig.AiModerationConfig) AIConnectionCheck {
	if cfg.VisionBaseURL == "" || cfg.VisionModel == "" {
		return AIConnectionCheck{Kind: "not_configured"}
	}
	started := time.Now()
	_, err := callVisionEvidence(ctx, cfg.VisionBaseURL, cfg.VisionAPIKey, cfg.VisionModel,
		time.Duration(cfg.VisionTimeoutMs)*time.Millisecond, "image/png", connectionTestImage())
	check := AIConnectionCheck{OK: true, Kind: "ok", Model: cfg.VisionModel, LatencyMs: time.Since(started).Milliseconds()}
	var failure visionFailure
	if errors.As(err, &failure) {
		check.Kind, check.HTTPStatus = failure.Kind, failure.HTTPStatus
		check.OK = failure.Kind == "uncertain"
	} else if err != nil {
		check.Kind, check.OK = "error", false
	}
	return check
}

// TestAIJevConnection 以一个最小 Noul 问题验证 Decisions API 可用且返回合法概率。
func TestAIJevConnection(ctx context.Context, cfg pageConfig.AiModerationConfig) AIConnectionCheck {
	if cfg.JevEndpoint == "" || cfg.JevModel == "" {
		return AIConnectionCheck{Kind: "not_configured"}
	}
	started := time.Now()
	response, err := callJev(ctx, cfg.JevEndpoint, cfg.JevAPIKey, time.Duration(cfg.JevTimeoutMs)*time.Millisecond, 0, jevRequest{
		Model: cfg.JevModel,
		State: map[string]string{"message": "This is a connectivity test from the forum admin panel."},
		Questions: map[string]jevQuestion{
			"is_test": {Type: "noul", Instructions: "Is `message` a connectivity test?"},
		},
	})
	check := AIConnectionCheck{Model: cfg.JevModel, LatencyMs: time.Since(started).Milliseconds()}
	var failure jevFailure
	switch {
	case errors.As(err, &failure):
		check.Kind, check.HTTPStatus = "jev_"+failure.Kind, failure.HTTPStatus
	case err != nil:
		check.Kind = "error"
	default:
		answer, ok := response.Answers["is_test"]
		if !ok || answer.Noul == nil || !validProbability(*answer.Noul) {
			check.Kind = "jev_malformed"
			return check
		}
		check.OK, check.Kind = true, "ok"
		if response.Model != "" {
			check.Model = response.Model
		}
	}
	return check
}

// connectionTestImage 生成一张 64×64 的几何测试图（不含任何用户内容）。
func connectionTestImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for x := range 64 {
		for y := range 64 {
			c := color.RGBA{R: 240, G: 240, B: 240, A: 255}
			switch {
			case x > 8 && x < 30 && y > 8 && y < 56:
				c = color.RGBA{R: 30, G: 90, B: 200, A: 255}
			case (x-46)*(x-46)+(y-32)*(y-32) < 120:
				c = color.RGBA{R: 220, G: 60, B: 40, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
