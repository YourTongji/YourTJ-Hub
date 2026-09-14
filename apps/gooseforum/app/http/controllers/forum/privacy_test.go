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

func TestBuildPrivacyPagePropsReplacesLegacyInsightFlareDisclosure(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	preferences.Set("app.env", "production")

	config := pageConfig.PrivacyPolicyConfig{
		Enabled: true,
		Content: "# Existing policy\n\n" + legacyInsightFlarePrivacyDisclosure,
	}
	props := buildPrivacyPageProps(config)
	if strings.Contains(props.ContentHTML, "InsightFlare") || strings.Contains(props.ContentHTML, "https://ana.yourtj.de") {
		t.Fatalf("legacy InsightFlare disclosure remained after migration: %s", props.ContentHTML)
	}
	if got := strings.Count(props.ContentHTML, `<h2 id="访问统计与会话回放`); got != 1 {
		t.Fatalf("Umami disclosure marker count = %d, want 1: %s", got, props.ContentHTML)
	}
}
