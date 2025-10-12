package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type WSContext struct {
	Conn   *websocket.Conn
	UserID int
	Hub    *Hub
	Data   json.RawMessage
}

type WSHandlerFunc func(ctx *WSContext) error

type Router struct {
	routes map[string]WSHandlerFunc
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]WSHandlerFunc)}
}

func (r *Router) Handle(messageType string, handler WSHandlerFunc) {
	r.routes[messageType] = handler
}

func (r *Router) Route(messageType string, ctx *WSContext) error {
	handler, ok := r.routes[messageType]
	if !ok {
		return fmt.Errorf("unknown message type: %s", messageType)
	}
	return handler(ctx)
}
