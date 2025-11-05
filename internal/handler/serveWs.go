package handler

import (
	"log"
	"net/http"
	"project/domain"

	"github.com/gorilla/websocket"
)

type WSHandler struct {
	wsHub domain.WSHub
}

func NewWSHandler(wsHub domain.WSHub) *WSHandler {
	return &WSHandler{
		wsHub: wsHub,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWs подключает пользователя к WebSocket хабу
// @Summary Подключение к WebSocket
// @Description Устанавливает WebSocket соединение для текущего пользователя и регистрирует его в WSHub
// @Tags WS
// @Accept json
// @Produce json
// @Success 101 "Switching Protocols"
// @Failure 400 {object} JSONResponse "Неверный запрос или отсутствует userID в контексте"
// @Failure 500 {object} JSONResponse "Ошибка сервера при апгрейде соединения"
// @Router /ws [get]
func (api *WSHandler) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	api.wsHub.AddClient(r.Context(), userID, conn)

}
