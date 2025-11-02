package domain

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Envelope struct {
	Type string
	Data json.RawMessage
}
type WSHub interface {
	RemoveClient(userID int)
	SendJSON(userID int, eventType string, data interface{}) error
	SendToUser(userID int, message []byte)
	AddClient(userID int, conn *websocket.Conn)
}
