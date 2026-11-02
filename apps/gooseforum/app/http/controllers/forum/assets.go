package forum

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"maps"
	"path/filepath"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/resource"
)

type manifestItem struct {
	File    string   `json:"file"`
	IsEntry bool     `json:"isEntry"`
	Css     []string `json:"css"`
	Imports []string `json:"imports"`
}

var manifest = loadManifest()

func resourceEntry(origin string, page ...PageComponent) template.HTML {
	if devServer := viteDevServer(); devServer != "" {
		devServer = strings.TrimRight(devServer, "/")
		origin = strings.TrimPrefix(origin, "/")
		devBase := strings.Trim(viteDevBase(), "/")
		if devBase != "" {
			devBase += "/"
		}
		// dev 模式输出同源相对路径，由 Go 服务器把 /assets/* 反向代理到 Vite，
		// 这样本机、局域网设备访问都不会受开发服务器地址影响
		return template.HTML(fmt.Sprintf(`<script type="module" src="/%s@vite/client"></script>
<script type="module" src="/%s%s"></script>`, devBase, devBase, origin))
	}

	item, ok := manifest[origin]
	if !ok {
		return template.HTML(fmt.Sprintf(`<script type="module" src="%s"></script>`, resourceAsset(origin)))
	}

	var sb strings.Builder
	for _, css := range collectCSS(origin, map[string]bool{}) {
		fmt.Fprintf(&sb, `<link rel="stylesheet" href="%s" crossorigin>`, resourceAsset(css))
		sb.WriteByte('\n')
	}
	fmt.Fprintf(&sb, `<script type="module" src="%s" crossorigin></script>`, resourceAsset(item.File))
	sb.WriteByte('\n')
	roots := []string{origin}
	var modules []string
	if len(page) > 0 && page[0] == PageComponentHome {
		// The initial page is already known from the server payload. Discover its
		// chunk alongside the entry instead of after downloading/executing it.
		const home = "src/site/pages/HomePage.vue"
		if chunk, exists := manifest[home]; exists && isModuleFile(chunk.File) {
			// Keep the known page ahead of its many shared dependencies in the
			// browser's preload queue, including HTTP/1 connection-limited links.
			modules = append(modules, chunk.File)
		}
		roots = append(roots, home)
	}
	visited := map[string]bool{origin: true}
	for _, root := range roots {
		modules = append(modules, collectModules(root, visited)...)
	}
	for _, file := range dedupeStrings(modules) {
		fmt.Fprintf(&sb, `<link rel="modulepreload" href="%s" crossorigin>`, resourceAsset(file))
		sb.WriteByte('\n')
	}
	return template.HTML(sb.String())
}

func viteDevServer() string {
	return viteDevServerFor(preferences.GetString("resource.devServer", ""), setting.IsProduction())
}

func viteDevServerFor(devServer string, production bool) string {
	if devServer != "" {
		return devServer
	}
	if !production {
		return "http://localhost:3010"
	}
	return ""
}

func viteDevBase() string {
	return preferences.GetString("resource.devBase", "/assets/")
}

func resourceAsset(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/assets/" + strings.TrimPrefix(path, "/")
}

func loadManifest() map[string]manifestItem {
	const manifestPath = "static/dist/.vite/manifest.json"
	content, err := fs.ReadFile(resource.GetTemplateFS(), manifestPath)
	if err != nil {
		// 包初始化时只执行一次：dist 缺失会让页面 200 但引用不存在的源文件路径
		// （/assets/src/site/main.ts → 404），过去完全无日志（issue #1284 B8）。
		slog.Error("forum assets: vite manifest missing, pages will reference unbundled source paths",
			"path", manifestPath, "err", err,
			"fix", "cd apps/gooseforum/resource && pnpm build")
		return map[string]manifestItem{}
	}
	items := jsonopt.Decode[map[string]manifestItem](content)
	if len(items) == 0 {
		slog.Error("forum assets: vite manifest is empty, pages will reference unbundled source paths",
			"path", manifestPath, "fix", "cd apps/gooseforum/resource && pnpm build")
	}
	return items
}

func collectCSS(entry string, visited map[string]bool) []string {
	if visited[entry] {
		return nil
	}
	visited[entry] = true
	item, ok := manifest[entry]
	if !ok {
		return nil
	}

	files := append([]string{}, item.Css...)
	for _, importKey := range item.Imports {
		files = append(files, collectCSS(importKey, visited)...)
	}
	return dedupeStrings(files)
}

// collectModules follows only static imports: unrelated dynamic pages and
// editors remain lazy. Visited keys bound cycles; output paths are deduplicated
// too because manifest entries can share an emitted chunk.
func collectModules(entry string, visited map[string]bool) []string {
	item, ok := manifest[entry]
	if !ok {
		return nil
	}
	var files []string
	for _, key := range item.Imports {
		if visited[key] {
			continue
		}
		visited[key] = true
		files = append(files, collectModules(key, visited)...)
		if imported, ok := manifest[key]; ok && isModuleFile(imported.File) {
			files = append(files, imported.File)
		}
	}
	if !visited[entry] {
		visited[entry] = true
		if isModuleFile(item.File) {
			files = append(files, item.File)
		}
	}
	return files
}

func isModuleFile(file string) bool {
	return strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".mjs")
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	res := make([]string, 0, len(values))
	for _, value := range values {
		normalized := filepath.ToSlash(value)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		res = append(res, normalized)
	}
	return res
}

func templateFuncs() template.FuncMap {
	funcs := template.FuncMap{}
	maps.Copy(funcs, templateFuncMap)
	funcs["ResourceEntry"] = resourceEntry
	funcs["ResourceAsset"] = resourceAsset
	return funcs
}
