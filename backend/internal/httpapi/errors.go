package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"game/backend/internal/auth"
	"game/backend/internal/game"
)

type apiFailure struct {
	status int
	code   string
}

var serviceFailures = map[error]apiFailure{
	auth.ErrInvalidLogin:       {status: http.StatusBadRequest, code: "invalid_login"},
	auth.ErrInvalidPassword:    {status: http.StatusBadRequest, code: "invalid_password"},
	auth.ErrLoginTaken:         {status: http.StatusConflict, code: "login_taken"},
	auth.ErrInvalidCredentials: {status: http.StatusUnauthorized, code: "invalid_credentials"},
	auth.ErrUnauthorized:       {status: http.StatusUnauthorized, code: "unauthorized"},
	game.ErrInvalidName:        {status: http.StatusBadRequest, code: "invalid_name"},
	game.ErrNotFound:           {status: http.StatusNotFound, code: "game_not_found"},
	game.ErrForbidden:          {status: http.StatusForbidden, code: "forbidden"},
}

var errorMessage = map[string]string{
	"invalid_body":        "Некорректный запрос",
	"invalid_login":       "Логин: 3–24 символа, латиница, цифры и подчёркивание",
	"invalid_password":    "Пароль должен быть не короче 8 символов",
	"login_taken":         "Этот логин уже занят",
	"invalid_credentials": "Неверный логин или пароль",
	"unauthorized":        "Нужно войти снова",
	"invalid_name":        "Название: от 1 до 40 символов",
	"game_not_found":      "Игра не найдена",
	"forbidden":           "Менять название может только владелец",
	"internal":            "Внутренняя ошибка",
}

type errorBody struct {
	Error struct {
		Code    string `json:"code" example:"invalid_login"`
		Message string `json:"message" example:"Логин: 3–24 символа, латиница, цифры и подчёркивание"`
	} `json:"error"`
} //@name errorResponse

func writeError(w http.ResponseWriter, status int, code string) {
	message, ok := errorMessage[code]
	if !ok {
		message = errorMessage["internal"]
	}
	body := errorBody{}
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	for sentinel, failure := range serviceFailures {
		if errors.Is(err, sentinel) {
			writeError(w, failure.status, failure.code)
			return
		}
	}
	slog.Default().Error("unhandled request error",
		"err", err,
		"method", r.Method,
		"path", r.URL.Path,
	)
	writeError(w, http.StatusInternalServerError, "internal")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
