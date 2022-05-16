package ginx

import (
	"net/http"
)

type Handler func(ctx *Context) error

// POST is a shortcut for router.Handle("POST", path, handle).
func (g *Engine) POST(relativePath string, tpl string, fn Handler) {
	g.wrapper(http.MethodPost, relativePath, tpl, fn)
}

// GET is a shortcut for router.Handle("GET", path, handle).
func (g *Engine) GET(relativePath string, tpl string, fn Handler) {
	g.wrapper(http.MethodGet, relativePath, tpl, fn)
}
