package httpapi

import (
	"net/http"

	"game/backend/internal/player"
	"game/backend/internal/realtime"
)

type realtimeHandler struct {
	sessions     PlayerSession
	hub          *realtime.Hub
	router       *realtime.Router
	onDisconnect func(*realtime.Client)
}

func RegisterRealtime(
	mux *http.ServeMux,
	sessions PlayerSession,
	hub *realtime.Hub,
	router *realtime.Router,
	onDisconnect func(*realtime.Client),
) {
	h := &realtimeHandler{
		sessions:     sessions,
		hub:          hub,
		router:       router,
		onDisconnect: onDisconnect,
	}
	mux.HandleFunc("GET /api/ws", h.live)
}

// live открывает канал живых сообщений сессии.
//
//	@Summary		Канал сообщений
//	@Tags			realtime
//	@Description	Переключение протокола на WebSocket. Каталог событий в эту спецификацию не входит. Игра и сид запрашиваются обычными HTTP-ручками.
//	@Security		AccessCookie
//	@Success		101	{string}	string		"WebSocket"
//	@Failure		401	{object}	errorBody	"unauthorized"
//	@Router			/api/ws [get]
func (h *realtimeHandler) live(w http.ResponseWriter, r *http.Request) {
	current, ok := authenticate(w, r, h.sessions)
	if !ok {
		return
	}
	_ = realtime.Serve(w, r, current.ID, current.Login, h.hub, h.router, h.onDisconnect)
}

func authenticate(w http.ResponseWriter, r *http.Request, sessions PlayerSession) (player.Player, bool) {
	raw := cookieValue(r, accessCookie)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return player.Player{}, false
	}
	current, err := sessions.Current(r.Context(), raw)
	if err != nil {
		writeServiceError(w, r, err)
		return player.Player{}, false
	}
	return current, true
}
