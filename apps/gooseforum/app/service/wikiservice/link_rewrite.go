package wikiservice

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	nethtml "golang.org/x/net/html"
)

// ---------- 仓库相对链接/资源重写（issue #284） ----------
//
// GitHub SSOT 仓库内页面用仓库相对路径互相引用（[下一页](other.md)、
// ![图](../assets/a.png)、[附件](../files/x.zip)）。同步时把渲染后 HTML 中的
// 站内相对引用改写为论坛内可解析的目标，再落库渲染快照：
//
//   - `.md` 页面链接 → 站内 wiki 路由 `/wiki/<path>`（去 `.md`，path 首段 = URL key
//     slug，逐段转义，与前端 wikiHref 一致）；
//   - 图片/附件 → GitHub raw 外链（`raw.githubusercontent.com/{repo}/{branch}/{path}`，
//     逐段转义）；
//   - 锚点 / 查询串 / 外部 URL（http(s)/mailto/data/…）/ 协议相对 URL 原样保留；
//   - `/wiki/...` 站内路由原样保留；其他 `/` 开头的根相对 URL 仅在仓库内能解析到
//     文件/页面时改写，否则视为站内路由原样保留（刻意行为，见 deployment.md）；
//   - 相对引用解析越界（`../../` 逃出仓库根）、引用的页面/文件不存在、非法转义 →
//     同步 fail-fast，错误信息含页面路径与引用原文（可操作）。
//
// 重写发生在同步时（渲染快照落库），公开读路径零改动；论坛普通帖子渲染不受影响。

// linkRewriteContext 一次同步的引用解析上下文。
type linkRewriteContext struct {
	cfg          GitConfig
	pageBySource map[string]string   // 仓库相对路径（去 .md）→ wiki_pages.path（URL key）
	repoFiles    map[string]struct{} // 仓库全部文件（含非 .md，仓库相对路径）
}

// scanRepoAllFiles 递归扫描 clone 目录下全部文件（排除 .git、隐藏目录），
// 返回仓库相对路径（/ 分隔，排序）。链接重写校验资源存在性用。
func scanRepoAllFiles(cloneDir string) ([]string, error) {
	var rels []string
	err := filepath.Walk(cloneDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != cloneDir {
				name := info.Name()
				if name == ".git" || strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		rel, err := filepath.Rel(cloneDir, path)
		if err != nil {
			return err
		}
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(rels)
	return rels, nil
}

// RawURL 返回仓库内文件的 GitHub raw 外链（{raw}/{repo}/{branch}/{path}，逐段转义）。
// 仓库未配置时返回空串。
func (c GitConfig) RawURL(relPath string) string {
	repo := c.RepoPath()
	if repo == "" {
		return ""
	}
	return "https://raw.githubusercontent.com/" + repo + "/" + c.Branch + "/" + githubPathEscape(relPath)
}

// renderWikiPageHTML 渲染 wiki 页面 markdown 并重写仓库相对引用（同步落库快照）。
func renderWikiPageHTML(ctx linkRewriteContext, wp wantedPage) (string, error) {
	rendered := markdown2html.PostMarkdownToHTML(wp.body)
	return rewriteRenderedWikiHTML(ctx, rendered, wp.sourcePath)
}

// rewriteRenderedWikiHTML 遍历渲染后 HTML 的 <a href>/<img src>，重写仓库相对引用。
func rewriteRenderedWikiHTML(ctx linkRewriteContext, rawHTML string, sourcePath string) (string, error) {
	root, err := nethtml.Parse(strings.NewReader("<div>" + rawHTML + "</div>"))
	if err != nil {
		return "", fmt.Errorf("wiki page %s: parse rendered html: %w", sourcePath, err)
	}
	var refErrs []string
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode {
			switch node.Data {
			case "a":
				if href := getAttr(node, "href"); href != "" {
					rewritten, err := rewriteWikiRef(ctx, href, sourcePath)
					if err != nil {
						refErrs = append(refErrs, err.Error())
					} else {
						setAttr(node, "href", rewritten)
					}
				}
			case "img":
				if src := getAttr(node, "src"); src != "" {
					rewritten, err := rewriteWikiRef(ctx, src, sourcePath)
					if err != nil {
						refErrs = append(refErrs, err.Error())
					} else {
						setAttr(node, "src", rewritten)
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if len(refErrs) > 0 {
		return "", fmt.Errorf("wiki page %s: %s", sourcePath, strings.Join(refErrs, "; "))
	}
	container := findFirstElementNode(root, "div")
	if container == nil {
		return rawHTML, nil
	}
	var buf bytes.Buffer
	for child := container.FirstChild; child != nil; child = child.NextSibling {
		if err := nethtml.Render(&buf, child); err != nil {
			return "", fmt.Errorf("wiki page %s: render rewritten html: %w", sourcePath, err)
		}
	}
	return buf.String(), nil
}

// rewriteWikiRef 解析单个 href/src 引用，返回改写后的完整引用（含 query/fragment）。
func rewriteWikiRef(ctx linkRewriteContext, raw string, sourcePath string) (string, error) {
	ref := strings.TrimSpace(raw)
	if ref == "" || isAbsoluteURL(ref) || strings.HasPrefix(ref, "//") {
		return raw, nil
	}
	pathPart, query, fragment := splitURLTail(ref)
	if pathPart == "" {
		return raw, nil // 纯锚点/查询串：原样保留
	}
	rootRelative := strings.HasPrefix(pathPart, "/")
	repoRef := strings.TrimPrefix(pathPart, "/")
	if rootRelative && (repoRef == "" || strings.HasPrefix(repoRef, "wiki/")) {
		return raw, nil // / 与 /wiki/... 站内路由：原样保留
	}
	decoded, err := url.PathUnescape(repoRef)
	if err != nil {
		return "", fmt.Errorf("invalid percent-encoding in reference %q", ref)
	}
	// 目录引用（. / .. / 结尾 /）→ 目录 index 页面。
	if decoded == "." || decoded == ".." || strings.HasSuffix(decoded, "/") {
		var dir string
		if rootRelative {
			dir = path.Clean(decoded)
		} else {
			dir = path.Clean(path.Join(path.Dir(sourcePath), decoded))
		}
		if dir == "." {
			dir = ""
		}
		if dir == ".." || strings.HasPrefix(dir, "../") {
			return "", fmt.Errorf("reference %q resolves outside the wiki repository root", ref)
		}
		idx := "index"
		if dir != "" {
			idx = dir + "/index"
		}
		page, ok := ctx.pageBySource[idx]
		if !ok {
			return "", fmt.Errorf("reference %q: no index page %q in wiki repository", ref, idx)
		}
		return "/wiki/" + githubPathEscape(page) + query + fragment, nil
	}
	var resolved string
	if rootRelative {
		resolved = path.Clean(decoded)
	} else {
		resolved = path.Clean(path.Join(path.Dir(sourcePath), decoded))
	}
	if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("reference %q resolves outside the wiki repository root", ref)
	}
	// 页面引用（.md 后缀）。
	if strings.HasSuffix(resolved, ".md") {
		pageSrc := strings.TrimSuffix(resolved, ".md")
		if page, ok := ctx.pageBySource[pageSrc]; ok {
			return "/wiki/" + githubPathEscape(page) + query + fragment, nil
		}
		if _, exists := ctx.repoFiles[resolved]; exists {
			return "", fmt.Errorf("reference %q: %q is a markdown file but not a projected wiki page", ref, resolved)
		}
		return "", fmt.Errorf("reference %q: no such wiki page %q in wiki repository", ref, pageSrc)
	}
	// 资产（仓库内存在的普通文件）→ GitHub raw 外链。
	if _, ok := ctx.repoFiles[resolved]; ok {
		rawURL := ctx.cfg.RawURL(resolved)
		if rawURL == "" {
			return "", fmt.Errorf("reference %q: wiki git repo not configured, cannot build raw URL", ref)
		}
		return rawURL + query + fragment, nil
	}
	// 扩展名省略的页面/目录 index（兼容旧 VitePress 无扩展名链接）。
	if page, ok := ctx.pageBySource[resolved]; ok {
		return "/wiki/" + githubPathEscape(page) + query + fragment, nil
	}
	if page, ok := ctx.pageBySource[resolved+"/index"]; ok {
		return "/wiki/" + githubPathEscape(page) + query + fragment, nil
	}
	if rootRelative {
		// 根相对且仓库内无法解析：视为站内路由，原样保留（刻意行为）。
		return raw, nil
	}
	return "", fmt.Errorf("reference %q: no such file or page %q in wiki repository", ref, resolved)
}

// splitURLTail 把引用拆为 path 部分 + query + fragment（首个未转义 # 与 ? 为分隔符；
// 仓库路径段本身不允许 ?，见 path.go reservedPathChars）。
func splitURLTail(ref string) (pathPart, query, fragment string) {
	pathPart = ref
	if i := strings.IndexByte(pathPart, '#'); i >= 0 {
		fragment = pathPart[i:]
		pathPart = pathPart[:i]
	}
	if i := strings.IndexByte(pathPart, '?'); i >= 0 {
		query = pathPart[i:]
		pathPart = pathPart[:i]
	}
	return
}

// isAbsoluteURL 判断是否为带 scheme 的绝对 URL（RFC 3986 scheme 语法）。
func isAbsoluteURL(ref string) bool {
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && ((c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'):
		case c == ':':
			return i > 0
		default:
			return false
		}
	}
	return false
}

func getAttr(node *nethtml.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func setAttr(node *nethtml.Node, key, value string) {
	for i := range node.Attr {
		if node.Attr[i].Key == key {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, nethtml.Attribute{Key: key, Val: value})
}

func findFirstElementNode(node *nethtml.Node, tag string) *nethtml.Node {
	if node == nil {
		return nil
	}
	if node.Type == nethtml.ElementNode && node.Data == tag {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirstElementNode(child, tag); found != nil {
			return found
		}
	}
	return nil
}
