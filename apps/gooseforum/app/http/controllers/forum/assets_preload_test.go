package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceEntryPreloadsStaticGraph(t *testing.T) {
	oldManifest := manifest
	oldEnv := preferences.GetString("app.env", "production")
	oldDev := preferences.GetString("resource.devServer", "")
	t.Cleanup(func() {
		manifest = oldManifest
		preferences.Set("app.env", oldEnv)
		preferences.Set("resource.devServer", oldDev)
	})
	preferences.Set("app.env", "production")
	preferences.Set("resource.devServer", "")
	manifest = map[string]manifestItem{
		"src/site/main.ts": {File: "assets/site.js", Css: []string{"assets/site.css"}, Imports: []string{"vue", "shared", "missing"}},
		"vue":              {File: "assets/vue.js", Imports: []string{"shared"}},
		"shared":           {File: "assets/shared.js", Imports: []string{"vue"}},
		"other":            {File: "assets/editor.js"},
	}
	html := string(resourceEntry("src/site/main.ts"))
	for _, file := range []string{"vue", "shared"} {
		expected := `<link rel="modulepreload" href="/assets/assets/` + file + `.js" crossorigin>`
		if strings.Count(html, expected) != 1 {
			t.Fatalf("expected exactly one %s, got %s", expected, html)
		}
	}
	if strings.Contains(html, "editor.js") || strings.Contains(html, `modulepreload" href="/assets/assets/site.js`) {
		t.Fatalf("unrelated/entry preload: %s", html)
	}
	if !strings.Contains(html, `rel="stylesheet" href="/assets/assets/site.css"`) || !strings.Contains(html, `type="module" src="/assets/assets/site.js"`) {
		t.Fatalf("entry CSS/script lost: %s", html)
	}
}

func TestResourceEntryHomeChunkScope(t *testing.T) {
	oldManifest := manifest
	oldEnv := preferences.GetString("app.env", "production")
	oldDev := preferences.GetString("resource.devServer", "")
	t.Cleanup(func() {
		manifest = oldManifest
		preferences.Set("app.env", oldEnv)
		preferences.Set("resource.devServer", oldDev)
	})
	preferences.Set("app.env", "production")
	preferences.Set("resource.devServer", "")
	manifest = map[string]manifestItem{
		"src/site/main.ts":            {File: "assets/site.js", Imports: []string{"shared"}},
		"shared":                      {File: "assets/shared.js"},
		"src/site/pages/HomePage.vue": {File: "assets/home.js", Imports: []string{"shared", "home-only"}},
		"home-only":                   {File: "assets/home-only.js"},
		"unused":                      {File: "assets/editor.js"},
	}
	home := string(resourceEntry("src/site/main.ts", PageComponentHome))
	for _, file := range []string{"home.js", "home-only.js", "shared.js"} {
		if strings.Count(home, `href="/assets/assets/`+file+`"`) != 1 {
			t.Fatalf("home graph missing or duplicated %s: %s", file, home)
		}
	}
	other := string(resourceEntry("src/site/main.ts", PageComponentTopic))
	if strings.Contains(other, "home.js") || strings.Contains(other, "home-only.js") || strings.Contains(home, "editor.js") {
		t.Fatalf("page scope leaked: %s / %s", home, other)
	}
	delete(manifest, "src/site/pages/HomePage.vue")
	if got := string(resourceEntry("src/site/main.ts", PageComponentHome)); got != other {
		t.Fatalf("missing home chunk must retain entry: %s", got)
	}
	preferences.Set("app.env", "local")
	if got := string(resourceEntry("src/site/main.ts", PageComponentHome)); strings.Contains(got, "modulepreload") || !strings.Contains(got, "@vite/client") {
		t.Fatalf("local Vite changed: %s", got)
	}
}

// The browser harness consumes the exact Go-generated head and real Vite build.
// Normal unit runs do not export fixtures or require a browser.
func TestResourceEntryBrowserFixture(t *testing.T) {
	output := os.Getenv("YOURTJ_ASSET_FIXTURE_OUTPUT")
	if output == "" {
		t.Skip("browser fixture export not requested")
	}
	oldEnv := preferences.GetString("app.env", "production")
	oldDev := preferences.GetString("resource.devServer", "")
	t.Cleanup(func() { preferences.Set("app.env", oldEnv); preferences.Set("resource.devServer", oldDev) })
	preferences.Set("app.env", "production")
	preferences.Set("resource.devServer", "")
	if _, ok := manifest["src/site/pages/HomePage.vue"]; !ok {
		t.Fatal("build the current Web sources before exporting this fixture")
	}
	if err := os.WriteFile(filepath.Join(output, "head.html"), []byte(resourceEntry("src/site/main.ts", PageComponentHome)), 0600); err != nil {
		t.Fatal(err)
	}
}
