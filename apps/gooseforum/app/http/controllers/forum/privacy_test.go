package forum

import (
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func TestBuildPrivacyPagePropsAddsUmamiDisclosureInProduction(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })

	config := pageConfig.PrivacyPolicyConfig{Enabled: true, Content: "# Existing policy"}
	preferences.Set("app.env", "production")
	production := buildPrivacyPageProps(config)
	if !strings.Contains(production.ContentHTML, "https://umi.yourtj.de") {
		t.Fatalf("production privacy policy missing Umami disclosure: %s", production.ContentHTML)
	}

	preferences.Set("app.env", "local")
	local := buildPrivacyPageProps(config)
	if strings.Contains(local.ContentHTML, "https://umi.yourtj.de") {
		t.Fatalf("local privacy policy must not include production Umami disclosure: %s", local.ContentHTML)
	}
}

func TestBuildPrivacyPagePropsAppendsDisclosureWhenOnlyURLMentioned(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	preferences.Set("app.env", "production")

	config := pageConfig.PrivacyPolicyConfig{
		Enabled: true,
		Content: "# Existing policy\n\n观测服务：https://umi.yourtj.de",
	}
	props := buildPrivacyPageProps(config)
	if strings.Count(props.ContentHTML, `<h2 id="访问统计与会话回放`) != 1 ||
		!strings.Contains(props.ContentHTML, "会话回放") {
		t.Fatalf("URL-only policy did not receive full Umami disclosure: %s", props.ContentHTML)
	}
}

func TestBuildPrivacyPagePropsDoesNotDuplicateUmamiDisclosure(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	preferences.Set("app.env", "production")

	config := pageConfig.PrivacyPolicyConfig{
		Enabled: true,
		Content: "# Existing policy\n\n" + umamiPrivacyDisclosure,
	}
	props := buildPrivacyPageProps(config)
	if got := strings.Count(props.ContentHTML, `<h2 id="访问统计与会话回放`); got != 1 {
		t.Fatalf("Umami disclosure marker count = %d, want 1: %s", got, props.ContentHTML)
	}
}

func TestBuildPrivacyPagePropsReplacesPersistedDefaultInsightFlareDisclosure(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	preferences.Set("app.env", "production")

	const legacyDefault = `# Existing policy

## 事件观测与性能数据

- **公共页面会加载自建 InsightFlare 事件观测服务**：用于统计页面访问、站内路由切换、出站链接和页面性能。当前统计站点为 ` + "`https://f.yourtj.de`" + `，观测服务为 ` + "`https://ana.yourtj.de`" + `。
- **采集字段**：事件可能包含访问页面的 hostname、pathname、URL query string、URL hash、页面标题、来源页面、语言、时区、屏幕尺寸、匿名访问者/会话标识，以及浏览器提供的有限设备信息；性能观测可能包含 TTFB、FCP、LCP、CLS、INP 等 Web Vitals。请勿把密码、Token、身份证件号或其他敏感信息放入 URL 的 query 或 hash。
- **浏览器隐私信号**：当前统计站点配置为不遵循浏览器 Do Not Track（DNT）信号；站点未提供应用内统计开关。如不希望参与统计，请阻止统计脚本或其采集请求。
- **保存与用途**：上述观测数据仅用于站点运行分析、性能改进与安全运营，由自建 InsightFlare 服务处理；具体保存期限以该服务的站点设置和归档策略为准。

## 联系

保留这段。`
	config := pageConfig.PrivacyPolicyConfig{Enabled: true, Content: legacyDefault}
	props := buildPrivacyPageProps(config)
	if strings.Contains(props.ContentHTML, "InsightFlare") || strings.Contains(props.ContentHTML, "https://ana.yourtj.de") {
		t.Fatalf("legacy InsightFlare disclosure remained after migration: %s", props.ContentHTML)
	}
	if !strings.Contains(props.ContentHTML, "保留这段") {
		t.Fatalf("content after legacy disclosure was not preserved: %s", props.ContentHTML)
	}
	if got := strings.Count(props.ContentHTML, `<h2 id="访问统计与会话回放`); got != 1 {
		t.Fatalf("Umami disclosure marker count = %d, want 1: %s", got, props.ContentHTML)
	}
}
