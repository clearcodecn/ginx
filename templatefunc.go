package ginx

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"
)

type TplFunc struct {
	ctx *Context
}

func (t *TplFunc) Moment(s string) string {
	return s
}

func (t *TplFunc) Split(s string) string {
	return "split + " + s
}

func newTplFunc(ctx *Context) *TplFunc {
	return &TplFunc{
		ctx: ctx,
	}
}

func (t *TplFunc) TemplateFunc() template.FuncMap {
	return template.FuncMap{
		"moment": t.Moment,
		"split":  t.Split,
		"assets": t.assets,
		"base64": t.base64,
		"html":   html,
	}
}

func (t *TplFunc) assets(s string) string {
	s = strings.TrimPrefix(s, "/")
	return fmt.Sprintf("/static/%s/%s", t.ctx.Theme, s)
}

func (t *TplFunc) base64(s string) string {
	if len(s) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func html(s string) template.HTML {
	return template.HTML(s)
}
