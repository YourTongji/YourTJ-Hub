package wikiservice

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	nethtml "golang.org/x/net/html"
)

const wikiAssetRoutePrefix = "/wiki/_assets/"

// errWikiRefEscapesRepo 标记引用解析中的安全类错误（路径逃逸/符号链接越界）：
// 这类错误必须整体失败（sync 停止），不允许 per-page 降级——否则恶意仓库
// 可通过「坏链接页跳过」绕过安全校验。普通内容错误（链接的页面/资产不存在、
// 图片引用 .md 等）降级为单页跳过并聚合告警。
var errWikiRefEscapesRepo = errors.New("wiki reference escapes repository")

// wikiReferenceResolver turns repository-relative Markdown destinations into
// public Wiki page or asset URLs. Source Markdown remains unchanged in the DB.
type wikiReferenceResolver struct {
	cloneDir          string
	pagesBySourcePath map[string]string
}

func newWikiReferenceResolver(cloneDir string, pages []wantedPage) *wikiReferenceResolver {
	pagesBySourcePath := make(map[string]string, len(pages))
	for _, page := range pages {
		pagesBySourcePath[page.sourcePath] = page.path
	}
	return &wikiReferenceResolver{cloneDir: cloneDir, pagesBySourcePath: pagesBySourcePath}
}

// Validate checks every rendered Markdown link and image before the sync mutates page rows.
func (r *wikiReferenceResolver) Validate(page wantedPage) error {
	reader := text.NewReader([]byte(page.body))
	doc := markdown2html.GetParser().Parser().Parse(reader)
	var validationErr error
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if validationErr != nil {
			return ast.WalkStop, nil
		}
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.Link:
			_, validationErr = r.resolve(page.sourcePath+".md", string(node.Destination), false)
		case *ast.Image:
			// review M3：图片指向 Markdown 页面会渲染成永久坏图（浏览器把
			// HTML 当图片加载），直接拒绝并给出可操作错误。
			_, validationErr = r.resolve(page.sourcePath+".md", string(node.Destination), true)
		}
		return ast.WalkContinue, nil
	})
	if validationErr != nil {
		return fmt.Errorf("wiki source %s.md: %w", page.sourcePath, validationErr)
	}
	return nil
}

func (r *wikiReferenceResolver) Render(page wantedPage) (string, error) {
	raw := markdown2html.PostMarkdownToHTML(page.body)
	root, err := nethtml.Parse(strings.NewReader("<div>" + raw + "</div>"))
	if err != nil {
		return "", fmt.Errorf("parse rendered HTML: %w", err)
	}

	var rewriteErr error
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if rewriteErr != nil {
			return
		}
		var key string
		isImage := false
		switch node.Data {
		case "a":
			key = "href"
		case "img":
			key = "src"
			isImage = true
		}
		if key != "" {
			for i := range node.Attr {
				if node.Attr[i].Key != key {
					continue
				}
				rewritten, err := r.resolve(page.sourcePath+".md", node.Attr[i].Val, isImage)
				if err != nil {
					rewriteErr = fmt.Errorf("wiki source %s.md: %w", page.sourcePath, err)
					return
				}
				node.Attr[i].Val = rewritten
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if rewriteErr != nil {
		return "", rewriteErr
	}

	container := findWikiHTMLElement(root, "div")
	if container == nil {
		return "", fmt.Errorf("rendered HTML missing container")
	}
	var output bytes.Buffer
	for child := container.FirstChild; child != nil; child = child.NextSibling {
		if err := nethtml.Render(&output, child); err != nil {
			return "", fmt.Errorf("render rewritten HTML: %w", err)
		}
	}
	return output.String(), nil
}

func (r *wikiReferenceResolver) resolve(sourceFile, destination string, isImage bool) (string, error) {
	trimmed := strings.TrimSpace(destination)
	if trimmed == "" {
		return destination, nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", destination, err)
	}
	// Absolute, protocol-relative, anchor-only, query-only, and site-root URLs
	// deliberately retain their author-provided meaning.
	if parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(trimmed, "//") || parsed.Path == "" || strings.HasPrefix(parsed.Path, "/") {
		return destination, nil
	}

	repoPath, err := resolveWikiRelativePath(sourceFile, parsed.EscapedPath())
	if err != nil {
		return "", fmt.Errorf("relative URL %q: %w", destination, err)
	}
	if isMarkdownPath(repoPath) {
		if isImage {
			return "", fmt.Errorf("image cannot reference a Markdown page %q", repoPath)
		}
		// review M4：扩展名大小写不敏感（.MD/.md 同义），但 TrimSuffix 必须
		// 用精确的 ext（path.Ext 保留大小写）切掉，map 查找才一致。
		key := strings.TrimSuffix(repoPath, path.Ext(repoPath))
		wikiPath, ok := r.pagesBySourcePath[key]
		if !ok {
			return "", fmt.Errorf("linked page %q does not exist", repoPath)
		}
		parsed.Path = "/wiki/" + wikiPath
		parsed.RawPath = ""
		return parsed.String(), nil
	}
	if _, _, err := resolveWikiAssetFile(r.cloneDir, repoPath); err != nil {
		return "", fmt.Errorf("asset %q: %w", repoPath, err)
	}
	parsed.Path = wikiAssetRoutePrefix + repoPath
	parsed.RawPath = ""
	return parsed.String(), nil
}

// isMarkdownPath 判断仓库相对路径是否为 Markdown 源文件（.md 家族，大小写不敏感）。
// 与 scanRepoFiles 只投影 `.md` 一致：仅 `.md` 是页面；.markdown/.mdown/.mkd
// 既不是页面也不得作为资产（F4：防止「不是页面的 Markdown」被原样吐给浏览器）。
func isMarkdownPath(repoPath string) bool {
	lower := strings.ToLower(repoPath)
	for _, ext := range []string{".md", ".markdown", ".mdown", ".mkd"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func resolveWikiRelativePath(sourceFile, encodedPath string) (string, error) {
	decoded, err := url.PathUnescape(encodedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path encoding: %w", err)
	}
	if strings.ContainsRune(decoded, '\x00') || strings.ContainsRune(decoded, '\\') {
		return "", fmt.Errorf("invalid repository path")
	}
	resolved := path.Clean(path.Join(path.Dir(sourceFile), decoded))
	if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
		// 安全类错误：仓库根逃逸必须致命（恶意链接不得降级跳过）。
		return "", fmt.Errorf("escapes repository root: %w", errWikiRefEscapesRepo)
	}
	if err := validateWikiAssetPath(resolved, false); err != nil {
		return "", err
	}
	return resolved, nil
}

// OpenWikiAsset opens a non-Markdown repository file after resolving symlinks
// and verifying that the result remains inside the configured clone directory.
func OpenWikiAsset(cloneDir, repoPath string) (*os.File, os.FileInfo, error) {
	assetPath, info, err := resolveWikiAssetFile(cloneDir, repoPath)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(assetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open asset: %w", err)
	}
	return file, info, nil
}

func resolveWikiAssetFile(cloneDir, repoPath string) (string, os.FileInfo, error) {
	if err := validateWikiAssetPath(repoPath, true); err != nil {
		return "", nil, err
	}
	rootAbs, err := filepath.Abs(cloneDir)
	if err != nil {
		return "", nil, fmt.Errorf("resolve clone directory: %w", err)
	}
	root, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", nil, fmt.Errorf("resolve clone directory: %w", err)
	}
	asset, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(repoPath)))
	if err != nil {
		return "", nil, fmt.Errorf("asset not found")
	}
	if !isPathWithin(root, asset) {
		// 安全类错误：符号链接越界必须致命。
		return "", nil, fmt.Errorf("asset resolves outside repository: %w", errWikiRefEscapesRepo)
	}
	info, err := os.Stat(asset)
	if err != nil {
		return "", nil, fmt.Errorf("inspect asset: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("asset is not a regular file")
	}
	return asset, info, nil
}

func validateWikiAssetPath(repoPath string, rejectMarkdown bool) error {
	if repoPath == "" || strings.HasPrefix(repoPath, "/") || path.Clean(repoPath) != repoPath {
		return fmt.Errorf("invalid repository path")
	}
	for _, segment := range strings.Split(repoPath, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.HasPrefix(segment, ".") || strings.ContainsRune(segment, '\\') {
			return fmt.Errorf("invalid repository path")
		}
	}
	if rejectMarkdown && isMarkdownPath(repoPath) {
		return fmt.Errorf("Markdown source files are not assets")
	}
	return nil
}

func isPathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func findWikiHTMLElement(node *nethtml.Node, tag string) *nethtml.Node {
	if node == nil {
		return nil
	}
	if node.Type == nethtml.ElementNode && node.Data == tag {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findWikiHTMLElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}
