package api

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// ExternalLinkPolicyResult GET /api/external-link-policy 响应 data
// （Flutter 外链确认策略同步：公开下发管理员配置的免确认白名单）。
type ExternalLinkPolicyResult struct {
	// TrustedDomains 小写主机名清单，按提交顺序；条目覆盖其全部子域。
	// 空数组表示不豁免任何域名。
	TrustedDomains []string `json:"trustedDomains"`
}

// GetExternalLinkPolicy 返回外链免确认白名单（公开只读，无鉴权）。
// 命中的主机名（含其子域）且风险为 normal_external 的站外链接跳过确认弹层；
// 标签与目标不符（suspicious）或预览解析被拦截（blocked）的链接永不豁免，
// 该边界由客户端在执行豁免时保证。数据源与 SSR payload 的
// layout.site.trustedExternalDomains 一致（page_config SiteSettings）。
func GetExternalLinkPolicy() component.Response {
	return component.SuccessResponse(ExternalLinkPolicyResult{
		TrustedDomains: hotdataserve.GetSiteSettingsConfigCache().TrustedExternalDomains,
	})
}
