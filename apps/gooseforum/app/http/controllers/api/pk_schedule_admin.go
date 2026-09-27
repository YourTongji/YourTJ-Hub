package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/console/job"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
	"github.com/robfig/cron/v3"
)

// PkSyncScheduleSettingsView 排课数据定时同步配置的管理端回显形状（issue #569）。
// 字段均非敏感（无密钥/凭证），原样回显；与「一系统 Cookie」配置共用同一
// 管理面板但互不包含对方数据。
type PkSyncScheduleSettingsView struct {
	Enabled  bool   `json:"enabled"`
	Schedule string `json:"schedule"`
	Term     string `json:"term"`
	Depth    int    `json:"depth"`
	Audience string `json:"audience"`
}

// GetPkSyncScheduleSettings 获取排课数据定时同步配置（issue #569）。
// 返回当前配置或内置默认（默认关闭、cron=30 2 * * *、目标学期留空=最近学期）。
func GetPkSyncScheduleSettings(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(toPkSyncScheduleView(hotdataserve.GetPkSyncScheduleConfigCache()))
}

// SavePkSyncScheduleSettingsReq 保存排课数据定时同步配置请求。请求提供完整配置
// （全部字段）——与凭据类设置不同，这里没有「保留旧值」语义，整块替换。
type SavePkSyncScheduleSettingsReq struct {
	Enabled bool `json:"enabled"`
	// Schedule 5 段标准 cron 表达式（分 时 日 月 周）；启用时必须有值且能被
	// robfig/cron 标准解析器解析（管理端 + 服务端双重校验）。
	Schedule string `json:"schedule" validate:"omitempty,max=128"`
	// Term 目标学期：数字 calendarId / 学期名；留空 = 同步最近已同步学期。
	Term string `json:"term" validate:"omitempty,max=128"`
	// Depth 回溯学期数（1..8，越界 clamp）。
	Depth int `json:"depth"`
	// Audience 数据来源（undergraduate / graduate；空按本科处理）。
	Audience string `json:"audience"`
}

// SavePkSyncScheduleSettings 保存排课数据定时同步配置（issue #569）：
// 原子写回 pageConfig，清理热缓存后热刷新 console/job 的 cron 注册（无需重启）。
// 启用时必须提供合法 cron 表达式（ParseStandard，5 段）；禁用时仅需 enabled=false。
func SavePkSyncScheduleSettings(req component.BetterRequest[SavePkSyncScheduleSettingsReq]) component.Response {
	schedule := strings.TrimSpace(req.Params.Schedule)
	audience := strings.TrimSpace(req.Params.Audience)
	if req.Params.Enabled {
		if schedule == "" {
			return component.FailResponseError(errors.New("启用定时同步必须提供 cron 表达式（如 30 2 * * *）"))
		}
		if _, err := cron.ParseStandard(schedule); err != nil {
			return component.FailResponseError(fmt.Errorf("cron 表达式无效（请使用 5 段标准格式，如 \"30 2 * * *\"）：%w", err))
		}
		if audience == "" {
			audience = string(pkservice.AudienceUndergraduate)
		}
	}
	if _, err := pkservice.ParseAudience(audience); err != nil {
		return component.FailResponseError(err)
	}

	depth := req.Params.Depth
	if depth < 1 {
		depth = 1
	}
	if depth > maxPkSyncDepth {
		depth = maxPkSyncDepth
	}

	config := pageConfig.PkSyncScheduleConfig{
		Enabled:  req.Params.Enabled,
		Schedule: schedule,
		Term:     strings.TrimSpace(req.Params.Term),
		Depth:    depth,
		Audience: audience,
	}
	pageConfig.UpdatePkSyncScheduleConfig(func(_ pageConfig.PkSyncScheduleConfig) pageConfig.PkSyncScheduleConfig {
		return config
	})
	hotdataserve.ClearPkSyncScheduleConfigCache()
	job.RefreshPkSyncCron()
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

func toPkSyncScheduleView(cfg pageConfig.PkSyncScheduleConfig) PkSyncScheduleSettingsView {
	audience := strings.TrimSpace(cfg.Audience)
	if audience == "" {
		audience = string(pkservice.AudienceUndergraduate)
	}
	return PkSyncScheduleSettingsView{
		Enabled:  cfg.Enabled,
		Schedule: cfg.Schedule,
		Term:     cfg.Term,
		Depth:    cfg.Depth,
		Audience: audience,
	}
}
