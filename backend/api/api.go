// Package api holds the API documentation: one OpenAPI file and a page that
// shows it with Swagger UI, so every route can be read and tried in a browser.
package api

import (
	_ "embed" // puts the two files below inside the server binary
	"net/http"
)

//go:embed openapi.json
var openAPI []byte

//go:embed docs.html
var docsPage []byte

// Register adds GET /docs (the Swagger UI page) and GET /openapi.json (the
// description it reads). Neither shows any data: they only describe the routes,
// and every "Try it out" call still goes through the access decision.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(openAPI)
	})
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(docsPage)
	})
}
