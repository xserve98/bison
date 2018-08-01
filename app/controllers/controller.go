package controllers

import (
	"fmt"
	tpl "html/template"
	"log"
	"path/filepath"

	"github.com/raggaer/bison/app/template"

	"github.com/raggaer/bison/app/config"
	"github.com/raggaer/bison/app/lua"
	"github.com/raggaer/bison/app/router"
	glua "github.com/tul/gopher-lua"
	"github.com/valyala/fasthttp"
)

// Handler main fasthttp handler
type Handler struct {
	Config *config.Config
	Routes []*router.Route
	Files  map[string]*glua.FunctionProto
	Tpl    *tpl.Template
}

// MainRoute handles all http requests
func (h *Handler) MainRoute(ctx *fasthttp.RequestCtx) {
	// If we are running under development mode reload stuff
	if h.Config.DevMode {
		routes, err := router.LoadRoutes(filepath.Join("app", "router", "router.lua"))
		if err != nil {
			ctx.Error("Unable to reload routes", 500)
			return
		}
		h.Routes = routes
		luaFiles, err := lua.CompileFiles(filepath.Join("app", "controllers"))
		if err != nil {
			ctx.Error("Unable to reload controllers", 500)
			return
		}
		h.Files = luaFiles
		tpl, err := template.LoadTemplates(filepath.Join("app", "views"), &template.TemplateFuncData{
			Config: h.Config,
			Files:  h.Files,
		})
		if err != nil {
			log.Println(err)
			ctx.Error("Unable to reload templates", 500)
			return
		}
		h.Tpl = tpl
	}

	// Retrieve current route
	params := map[string]string{}
	ctx.VisitUserValues(func(b []byte, i interface{}) {
		params[string(b)] = fmt.Sprint(i)
	})
	route := router.RetrieveCurrentRoute(params, string(ctx.Method()), string(ctx.Path()), h.Routes)

	// Retrieve compiled file for this route
	p := filepath.Join("app", "controllers", route.File)
	if h.Config.TestMode {
		p = filepath.Join("controllers", route.File)
	}
	fmt.Println(p)
	proto, ok := h.Files[p]
	if !ok {
		ctx.NotFound()
		return
	}

	// Create state with bison modules
	state := lua.NewState([]*lua.Module{
		lua.NewHTTPModule(ctx, params),
		lua.NewConfigModule(h.Config.Custom),
		lua.NewTemplateModule(h.Tpl, ctx),
		lua.NewURLModule(),
	})
	defer state.Close()

	// Execute compiled state
	if err := lua.DoCompiledFile(state, proto); err != nil {
		log.Println(err)
		ctx.Error("Unable to execute "+route.Path, 500)
		return
	}
}
