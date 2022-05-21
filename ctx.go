package ginx

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	XAccountId = "x-account-id"
	XToken     = "x-token"
	ErrorTpl   = "error.gohtml"
)

var (
	DefaultStaticRoot = "/static"
)

type Context struct {
	*gin.Context

	DomainConfig *DomainConfig

	TemplateName string `json:"templateName"` // 模板名称

	data map[string]interface{}

	Theme string

	AccountId string
}

func (c Context) Db() *DB {
	return db
}

type TemplateFunc func(ctx *Context) template.FuncMap

type Engine struct {
	*gin.Engine

	templateFuncs map[string]TemplateFunc
	staticRoot    string // 静态文件根路径
	templateRoot  string // 模板根路径

	themes map[string]map[string]*template.Template
}

func Default() *Engine {
	e := &Engine{
		Engine:        gin.Default(),
		templateFuncs: make(map[string]TemplateFunc),
		staticRoot:    GlobalConfig().WebConfig.StaticRoot,
		templateRoot:  GlobalConfig().WebConfig.TemplateRoot,
		themes:        make(map[string]map[string]*template.Template),
	}

	e.init()

	return e
}

func (e *Engine) init() {
	e.Static(DefaultStaticRoot, e.staticRoot)
	e.GET("/favicon.ico", "", func(ctx *Context) error {
		ctx.File(filepath.Join(e.staticRoot, ctx.DomainConfig.Template, "favicon.ico"))
		return nil
	})
	// ads.txt
	e.GET("root.txt", "", func(ctx *Context) error {
		ctx.String(200, ctx.DomainConfig.RootTxt)
		return nil
	})
	// ads.txt
	e.GET("ads.txt", "", func(ctx *Context) error {
		ctx.String(200, ctx.DomainConfig.AdsTxt)
		return nil
	})

	e.Any("/api/reload", func(ctx *gin.Context) {
		if ctx.GetHeader(XToken) != GlobalConfig().Token {
			ctx.AbortWithStatus(403)
			return
		}
		ReloadConfig()

		data, _ := yaml.Marshal(GlobalConfig())
		ctx.Header("Content-Type", "text/html")
		ctx.String(200, `<pre style='font-size:16px;font-family:"serif"'>%s</pre>`, string(data))
	})

	e.NoRoute(func(ctx *gin.Context) {
		myCtx := toMyCtx(ctx, ``)
		if ctx.Request.RequestURI == myCtx.DomainConfig.GoogleSiteVerify {
			ctx.String(200, myCtx.DomainConfig.GoogleSiteVerifyText)
		}
	})
}

func (e *Engine) initTemplate(ctx *Context) {
	// 模板渲染查找.
	if err := filepath.Walk(e.templateRoot, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		paths := strings.Split(path, string(filepath.Separator))
		if len(paths) != 3 {
			return fmt.Errorf("文件路劲只能为3级: %v", path)
		}
		theme, name := paths[1], paths[2]
		_, ok := e.themes[theme]
		if !ok {
			e.themes[theme] = make(map[string]*template.Template)
		}
		t, err := template.New(name).Funcs(newTplFunc(ctx).TemplateFunc()).ParseGlob(filepath.Dir(path) + "/*.gohtml")
		if err != nil {
			return err
		}
		e.themes[theme][name] = t
		return nil
	}); err != nil {
		panic(err)
	}
}

func toMyCtx(ctx *gin.Context, tpl string) *Context {
	myCtx := &Context{
		Context:      ctx,
		DomainConfig: nil,
		TemplateName: tpl,
		data:         make(map[string]interface{}),
	}
	acid := ctx.GetHeader(XAccountId)
	if GlobalConfig().DevCid != "" {
		acid = GlobalConfig().DevCid
	}
	// 查找 config.
	cfg := GlobalConfig().Domains[acid]
	if cfg == nil {
		ctx.AbortWithError(400, fmt.Errorf("account id is not found: %v", acid))
		return nil
	}
	myCtx.DomainConfig = cfg
	myCtx.Theme = cfg.Template
	myCtx.data["Config"] = cfg
	return myCtx
}

func (e *Engine) wrapper(method string, path string, tpl string, fn func(ctx *Context) error) {
	e.Handle(method, path, func(ctx *gin.Context) {
		myCtx := toMyCtx(ctx, tpl)
		if myCtx == nil {
			return
		}

		if err := fn(myCtx); err != nil {
			ctx.HTML(200, ErrorTpl, myCtx.data)
			return
		}

		if tpl == "" {
			return
		}
		tmpl, err := template.New(tpl).
			Funcs(newTplFunc(myCtx).TemplateFunc()).
			ParseGlob(filepath.Join(e.templateRoot, myCtx.DomainConfig.Template, "*.gohtml"))
		if err != nil {
			ctx.Error(err)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, myCtx.data); err != nil {
			ctx.AbortWithError(400, fmt.Errorf("render html failed: %v", err))
			return
		}
		ctx.DataFromReader(200, int64(buf.Len()), "text/html", &buf, nil)
	})
}

func (e *Engine) Run() error {
	fmt.Println("")
	fmt.Println("")
	fmt.Println("---------------- 配置信息 ------------------------")
	yaml.NewEncoder(os.Stdout).Encode(globalConfig)
	fmt.Println("")
	fmt.Println("")
	fmt.Println(fmt.Sprintf("http://%s:%s", globalConfig.WebConfig.Addr, globalConfig.WebConfig.Port))
	return e.Engine.Run(fmt.Sprintf("%s:%s", globalConfig.WebConfig.Addr, globalConfig.WebConfig.Port))
}

func (c *Context) Assign(key string, value interface{}) {
	c.data[key] = value
}
