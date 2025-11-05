package domain

import (
	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Envelope struct {
	Type string
	Data json.RawMessage
}

type WSHub interface {
	RemoveClient(ctx context.Context, userID int)
	SendJSON(ctx context.Context, userID int, eventType string, data interface{}) error
	SendToUser(ctx context.Context, userID int, message []byte)
	AddClient(ctx context.Context, userID int, conn *websocket.Conn)
}
