package httpapi

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "game/backend/docs"
)

// RegisterSwagger serves Swagger UI and the generated OpenAPI document.
func RegisterSwagger(mux *http.ServeMux) {
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	})
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DocExpansion("list"),
		httpSwagger.PersistAuthorization(true),
		httpSwagger.UIConfig(map[string]string{
			"withCredentials": "true",
		}),
	))
}
