package wikiservice

import (
	"bytes"
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
			_, validationErr = r.resolve(page.sourcePath+".md", string(node.Destination))
		case *ast.Image:
			_, validationErr = r.resolve(page.sourcePath+".md", string(node.Destination))
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
		if node.Type == nethtml.ElementNode {
			var key string
			switch node.Data {
			case "a":
				key = "href"
			case "img":
				key = "src"
			}
			if key != "" {
				for i := range node.Attr {
					if node.Attr[i].Key != key {
						continue
					}
					rewritten, err := r.resolve(page.sourcePath+".md", node.Attr[i].Val)
					if err != nil {
						rewriteErr = fmt.Errorf("wiki source %s.md: %w", page.sourcePath, err)
						return
					}
					node.Attr[i].Val = rewritten
				}
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

func (r *wikiReferenceResolver) resolve(sourceFile, destination string) (string, error) {
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
	if strings.EqualFold(path.Ext(repoPath), ".md") {
		wikiPath, ok := r.pagesBySourcePath[strings.TrimSuffix(repoPath, ".md")]
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
		return "", fmt.Errorf("escapes repository root")
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
		return "", nil, fmt.Errorf("asset resolves outside repository")
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
	if rejectMarkdown && strings.EqualFold(path.Ext(repoPath), ".md") {
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
