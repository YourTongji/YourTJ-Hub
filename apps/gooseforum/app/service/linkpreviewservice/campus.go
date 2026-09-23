package linkpreviewservice

import (
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

// 校园网链接一律按一方资源本地渲染：服务端不对其发起任何 HTTP 请求。
//
// 校园服务通常只在校内可达——DNS 落在 RFC1918 私网段，safefetch 的公网校验会
// 判为 blocked；而放行这类地址去抓取，就要在 dialVerified 上开一条绕过「只拨
// 已验证公网 IP」的旁路，并且 URL 里的端口/路径由发帖人控制，等于给出一个端口
// 探测面。抓回来的「名字」也没有用：实测 6 个常见校园主机里，software 的
// <title> 是「登录」（登录页）、agent / yunpan 是微前端与 SPA 壳（无 title）、
// 1 与 www 只有无区分度的「同济大学」，且全部没有 og 元数据。所以这里只做本地
// 渲染，SSRF 面保持为零。
//
// 展示名只来自部署配置，取值优先级（三级回退）：
//
//	[link_preview]
//	domains = ["tongji.edu.cn"]                  # 后缀匹配，含子域
//	[link_preview.domain_names]
//	"tongji.edu.cn" = "同济大学"                  # 可选，域名级默认名
//	[link_preview.host_names]
//	"1.tongji.edu.cn" = "同济大学教学管理系统"      # 可选，精确主机，优先级最高
//
// 两级都没配就返回空名：兜底文案（中文「校园网」、描述「需校园网络访问」等）
// 由客户端按自身语言渲染，服务端不再硬编码任何中文。domains 留空 = 关闭校园网卡片。
type CampusPolicy struct {
	// Domains 为后缀匹配的校园域名（如 tongji.edu.cn）。
	Domains []string
	// DomainNames 为可选的域名级默认名，键同 Domains。
	DomainNames map[string]string
	// HostNames 为可选的精确主机展示名。
	HostNames map[string]string
}

// campusPolicyFromConfig 每次解析都重新读配置：链接预览是无状态热路径，
// 配置改动应当立即生效，与 configuredOrigins 的做法一致。
func campusPolicyFromConfig() CampusPolicy {
	return CampusPolicy{
		Domains:     preferences.GetStringSlice("link_preview.domains"),
		DomainNames: normalizeCampusNames(preferences.GetStringMapString("link_preview.domain_names")),
		HostNames:   normalizeCampusNames(preferences.GetStringMapString("link_preview.host_names")),
	}
}

// normalizeCampusNames 统一小写去尾点、丢弃空名，避免配置里的大小写或空格让
// 精确匹配悄悄失效。
func normalizeCampusNames(names map[string]string) map[string]string {
	if len(names) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(names))
	for key, name := range names {
		key = normalizeCampusHost(key)
		name = strings.TrimSpace(name)
		if key == "" || name == "" {
			continue
		}
		normalized[key] = name
	}
	return normalized
}

// Card 返回校园网卡片的展示名。第二个返回值为 false 表示该 host 不属于校园网，
// 调用方应回退到外链解析；名字为空表示配置没给名字，客户端应使用自身语言的兜底
// 文案。
func (policy CampusPolicy) Card(host string) (string, bool) {
	host = normalizeCampusHost(host)
	if host == "" {
		return "", false
	}
	domain := policy.matchedDomain(host)
	if domain == "" {
		return "", false
	}
	if name := policy.HostNames[host]; name != "" {
		return name, true
	}
	return policy.DomainNames[domain], true
}

// matchedDomain 只认「等于配置域」或「配置域前恰好多一个点」的后缀，避免
// tongji.edu.cn.evil.com、not-tongji.edu.cn 这类仿冒域名命中校园网。返回规范化
// 后的命中域名，供域名级默认名取用。
func (policy CampusPolicy) matchedDomain(host string) string {
	for _, domain := range policy.Domains {
		domain = normalizeCampusHost(domain)
		if domain == "" {
			continue
		}
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return domain
		}
	}
	return ""
}

func normalizeCampusHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}
