package forum

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/resource"
	"github.com/gin-gonic/gin"
)

type templateRegistry struct {
	templates map[string]*template.Template
}

type templateData struct {
	Payload PagePayload
	Lang    string
}

var currentRegistry = mustNewRegistry()
var errCurrentRegistry error

func ReloadTemplates() {
	currentRegistry = mustNewRegistry()
}

func mustNewRegistry() *templateRegistry {
	registry, err := newRegistry(resource.GetTemplateFS())
	if err != nil {
		errCurrentRegistry = err
		slog.Error("failed to load resource templates", "err", err)
	}
	return registry
}

func newRegistry(fileSystem fs.FS) (*templateRegistry, error) {
	base := template.New("goose_resource").
		Funcs(templateFuncs()).
		Funcs(sprig.FuncMap())

	sharedFiles, err := templateFiles(fileSystem, "templates/layout", "templates/partials")
	if err != nil {
		return nil, err
	}
	if _, err := base.ParseFS(fileSystem, sharedFiles...); err != nil {
		return nil, err
	}

	registry := &templateRegistry{templates: map[string]*template.Template{}}
	err = fs.WalkDir(fileSystem, "templates/pages", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".gohtml") {
			return nil
		}
		tmpl, err := base.Clone()
		if err != nil {
			return err
		}
		if _, err := tmpl.ParseFS(fileSystem, path); err != nil {
			return err
		}
		registry.templates[d.Name()] = tmpl
		return nil
	})
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func templateFiles(fileSystem fs.FS, roots ...string) ([]string, error) {
	var files []string
	for _, root := range roots {
		err := fs.WalkDir(fileSystem, root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".gohtml") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (r *templateRegistry) render(w io.Writer, name string, data any) error {
	tmpl, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	return tmpl.ExecuteTemplate(w, name, data)
}

func renderPage(c *gin.Context, templateName string, payload PagePayload) {
	renderPageWithStatus(c, http.StatusOK, templateName, payload)
}

func renderAppShell(c *gin.Context, payload PagePayload) {
	renderPage(c, "app_shell.gohtml", payload)
}

// renderInternalError 渲染 500 错误页（区别于 404，避免把存储故障伪装成内容不存在）。
func renderInternalError(c *gin.Context) {
	payload := PagePayload{
		Component: PageComponentError,
		Props: ErrorPageProps{
			Code:        "500",
			Title:       i18n.T(requestLang(c), "meta.internalError"),
			MessageCode: component.MessageOperationFailed,
		},
		Meta: PageMeta{
			Title: pageTitle(i18n.T(requestLang(c), "meta.internalError")),
		},
		Layout:  buildLayout(c, "topics"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	renderPageWithStatus(c, http.StatusInternalServerError, "error.gohtml", payload)
}
func renderPageWithStatus(c *gin.Context, status int, templateName string, payload PagePayload) {
	c.Status(status)
	c.Header("Vary", "X-Goose-Page, Accept")
	if isPageRequest(c) {
		c.Header("Cache-Control", "no-store")
		c.JSON(status, payload)
		return
	}
	// HTML 文档同样禁用缓存：goose-payload 内嵌管理端配置（如 /schedule 节次作息），
	// 缺头时浏览器启发式缓存/bfcache 会把保存后的新配置继续以旧 DOM 呈现。
	c.Header("Cache-Control", "no-store")
	// Content-Type 必须显式声明：view 路由组挂了 gzip 中间件，模板又是流式写入，
	// 压缩 writer 会在首字节提交响应头，Go 的内容嗅探没有机会补 Content-Type；
	// 缺头叠加 nosniff 会被浏览器按 text/plain 展示源码。
	c.Header("Content-Type", "text/html; charset=utf-8")
	if currentRegistry == nil {
		if errCurrentRegistry != nil {
			// 500 详情只落服务端日志，不回显内部错误（review 备注：避免向客户端泄漏
			// 内部路径/实现细节）。
			slog.Error("render template registry unavailable", "err", errCurrentRegistry)
		} else {
			slog.Error("render template registry is not initialized")
		}
		c.String(http.StatusInternalServerError, "internal server error")
		return
	}
	if err := currentRegistry.render(c.Writer, filepath.Base(templateName), templateData{
		Payload: payload,
		Lang:    requestLang(c),
	}); err != nil {
		slog.Error("render resource template failed", "template", templateName, "err", err)
		c.String(http.StatusInternalServerError, "internal server error")
	}
}

func isPageRequest(c *gin.Context) bool {
	return c.GetHeader("X-Goose-Page") == "true"
}
