package main

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -d ../.. -g cmd/server/doc.go -o ../../docs --parseInternal

// @title       Game API
// @version     1.0
// @description HTTP API игры. Спека собрана из комментариев над хендлерами, на сами запросы страница не смотрит.
// @BasePath    /

// @securityDefinitions.apikey AccessCookie
// @in                         cookie
// @name                       access_token
// @description                HttpOnly cookie сессии. Её выставляют register и login, браузер шлёт её сам.

// @securityDefinitions.apikey RefreshCookie
// @in                         cookie
// @name                       refresh_token
// @description                HttpOnly cookie обновления, path /api/auth. Нужна для refresh и logout.
