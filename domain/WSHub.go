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
	RemoveClient(ctx context.Context, userID int32)
	SendJSON(ctx context.Context, userID int32, eventType string, data interface{}) error
	SendToUser(ctx context.Context, userID int32, message []byte)
	AddClient(ctx context.Context, userID int32, conn *websocket.Conn)
}
