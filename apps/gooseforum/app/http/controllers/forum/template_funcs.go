package forum

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
)

var templateFuncMap = template.FuncMap{
	"SafeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"Nl2br": func(text string) template.HTML {
		escaped := template.HTMLEscapeString(text)
		result := strings.ReplaceAll(escaped, "\n", "<br>")
		return template.HTML(result)
	},
	"json": func(v any) template.JS {
		return template.JS(jsonopt.Encode(v))
	},
	// t localizes a server-rendered string, e.g. {{ t .Lang "search" }}.
	// Extra args are alternating name/value pairs for {name} placeholders.
	"t": i18n.T,
	// serverMessage resolves a backend messageCode into a localized message,
	// mirroring the frontend resolveApiMessage. Usage:
	// {{ serverMessage .Lang .Payload.Props.MessageCode .Payload.Props.Params (t .Lang "common.loadFailed") }}.
	// The last argument is a translated fallback.
	"serverMessage": serverMessage,
}

// serverMessage adapts i18n.ServerMessage for templates: payload fields are
// typed component.MessageCode / component.MessageParams, which Go templates
// pass as interface{}, so the adapter unwraps them before delegating.
func serverMessage(lang string, code any, params any, fallback string) string {
	var codeStr string
	switch v := code.(type) {
	case nil:
	case string:
		codeStr = v
	case component.MessageCode:
		codeStr = string(v)
	default:
		codeStr = fmt.Sprint(v)
	}
	var paramsMap map[string]any
	switch v := params.(type) {
	case nil:
	case map[string]any:
		paramsMap = v
	case component.MessageParams:
		paramsMap = map[string]any(v)
	default:
		paramsMap = nil
	}
	return i18n.ServerMessage(lang, codeStr, paramsMap, fallback)
}

// requestLang resolves and normalizes the request locale. It delegates to
// component.RequestLang so the templates and the activation page share one
// source of truth. The normalized value is also safe as an <html lang> value.
func requestLang(c *gin.Context) string {
	return component.RequestLang(c)
}
