package app

import (
	"fmt"
	"log/slog"
	"net/http"
	handler "puff/handler"
	response "puff/response"
	route "puff/route"
	router "puff/router"
	"time"
)

type Config struct {
	Network bool   // host to the entire network?
	Port    int    // port number to use
	Name    string // the name of the project, defaults to "Puff API"
}

type App struct {
	*Config
	Routes  []route.Route
	Routers []*router.Router
	// Middlewares
}

func (a *App) IncludeRouter(r *router.Router) {
	a.Routers = append(a.Routers, r)
}

func indexPage() interface{} { // FIX ME: Written temporarily only for tests.
	return response.HTMLResponse{
		Content: "<h1> hello world </h1> <p> this is a temporary index page </p>",
	}
}

func (a *App) sendToHandler(w http.ResponseWriter, req *http.Request) {
	handler.Handler(w, req, indexPage)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		next.ServeHTTP(w, r)
		processingTime := time.Since(startTime).String()
		slog.Info(
			"HTTP Request",
			slog.String("HTTP METHOD", r.Method),
			slog.String("URL", r.URL.String()),
			slog.String("Processing Time", processingTime),
		)
	})
}

func (a *App) ListenAndServe() {
	mux := http.NewServeMux()
	router := loggingMiddleware(mux)

	mux.HandleFunc("/", a.sendToHandler)

	addr := ""
	if a.Network {
		addr += "0.0.0.0"
	}
	addr += fmt.Sprintf(":%d", a.Port)

	slog.Info(fmt.Sprintf("Running Puff 💨 on port %d", a.Port))

	http.ListenAndServe(addr, router)
}
