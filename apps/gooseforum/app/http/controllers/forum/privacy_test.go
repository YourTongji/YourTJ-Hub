package forum

import (
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func TestBuildPrivacyPagePropsAddsInsightFlareDisclosureInProduction(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })

	config := pageConfig.PrivacyPolicyConfig{Enabled: true, Content: "# Existing policy"}
	preferences.Set("app.env", "production")
	production := buildPrivacyPageProps(config)
	if !strings.Contains(production.ContentHTML, "https://ana.yourtj.de") {
		t.Fatalf("production privacy policy missing InsightFlare disclosure: %s", production.ContentHTML)
	}

	preferences.Set("app.env", "local")
	local := buildPrivacyPageProps(config)
	if strings.Contains(local.ContentHTML, "https://ana.yourtj.de") {
		t.Fatalf("local privacy policy must not include production InsightFlare disclosure: %s", local.ContentHTML)
	}
}

func TestBuildPrivacyPagePropsAppendsDisclosureWhenOnlyURLMentioned(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	preferences.Set("app.env", "production")

	config := pageConfig.PrivacyPolicyConfig{
		Enabled: true,
		Content: "# Existing policy\n\n观测服务：https://ana.yourtj.de",
	}
	props := buildPrivacyPageProps(config)
	if strings.Count(props.ContentHTML, `<h2 id="事件观测与性能数据`) != 1 ||
		!strings.Contains(props.ContentHTML, "URL query string") {
		t.Fatalf("URL-only policy did not receive full InsightFlare disclosure: %s", props.ContentHTML)
	}
}

func TestBuildPrivacyPagePropsDoesNotDuplicateInsightFlareDisclosure(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	preferences.Set("app.env", "production")

	config := pageConfig.PrivacyPolicyConfig{
		Enabled: true,
		Content: "# Existing policy\n\n" + insightFlarePrivacyDisclosure,
	}
	props := buildPrivacyPageProps(config)
	if got := strings.Count(props.ContentHTML, `<h2 id="事件观测与性能数据`); got != 1 {
		t.Fatalf("InsightFlare disclosure marker count = %d, want 1: %s", got, props.ContentHTML)
	}
}
