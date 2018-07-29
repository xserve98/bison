package main

import (
	"log"
	"net/http"

	"github.com/buaazp/fasthttprouter"
	"github.com/raggaer/bison/lua"

	"github.com/valyala/fasthttp"
)

func main() {
	// Load config file
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Compile all lua files
	files, err := lua.CompileFiles("controllers")
	if err != nil {
		log.Fatal(err)
	}

	// Load all templates
	tpl, err := loadTemplates(&TemplateFuncData{
		Config: config,
		Files:  files,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create fasthttp router
	router := fasthttprouter.New()
	routes, err := loadRoutes()
	if err != nil {
		log.Fatal(err)
	}

	// Create fasthttp server
	handler := &Handler{
		Config: config,
		Routes: routes,
		Files:  files,
		Tpl:    tpl,
	}

	for _, r := range routes {
		if r.Method == http.MethodGet {
			router.GET(r.Path, handler.MainRoute)
		}
		if r.Method == http.MethodPost {
			router.POST(r.Path, handler.MainRoute)
		}
	}
	if config.DevMode {
		log.Println("Running development mode - bison listening on address '" + config.Address + "'")
	} else {
		log.Println("bison listening on address '" + config.Address + "'")
	}
	fasthttp.ListenAndServe(config.Address, router.Handler)
}
