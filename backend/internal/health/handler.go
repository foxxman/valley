package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Response struct {
	Status   string `json:"status"`
	Database bool   `json:"database"`
}

type databasePinger interface {
	Ping(context.Context) error
}

func NewHandler(pool databasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		databaseUp := pool.Ping(ctx) == nil
		statusCode := http.StatusOK
		resp := Response{Status: "ok", Database: true}
		if !databaseUp {
			statusCode = http.StatusServiceUnavailable
			resp = Response{Status: "unavailable", Database: false}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
