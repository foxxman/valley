package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Response struct {
	Status   string `json:"status" example:"ok"`
	Database bool   `json:"database" example:"true"`
} //@name health

type databasePinger interface {
	Ping(context.Context) error
}

// NewHandler проверяет процесс и доступность базы.
//
//	@Summary	Проверка живости
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	Response
//	@Failure	503	{object}	Response	"База недоступна"
//	@Router		/healthcheck [get]
func NewHandler(pool databasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		statusCode := http.StatusOK
		resp := Response{Status: "ok", Database: true}
		if err := pool.Ping(ctx); err != nil {
			slog.Default().Error("database ping failed", "err", err)
			statusCode = http.StatusServiceUnavailable
			resp = Response{Status: "unavailable", Database: false}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
