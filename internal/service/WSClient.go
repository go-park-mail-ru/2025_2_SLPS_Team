package service

import (
	"context"
	"encoding/json"
	"project/domain"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	userID int
}
type Hub struct {
	clients map[int]*Client
	mu      sync.RWMutex
}

func NewHub() domain.WSHub {
	return &Hub{
		clients: make(map[int]*Client),
	}
}

func (h *Hub) RemoveClient(ctx context.Context, userID int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		client.conn.Close()
		delete(h.clients, userID)
		domain.FromContext(ctx).Info("WS client removed successfully")
	}
	domain.FromContext(ctx).Warn("Client does not exist")
}

func (h *Hub) AddClient(ctx context.Context, userID int, conn *websocket.Conn) {
	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	h.mu.Lock()
	h.clients[userID] = client
	h.mu.Unlock()

	go h.writePump(ctx, client)

	domain.FromContext(ctx).Info("WS client added")
}

func (h *Hub) SendJSON(ctx context.Context, userID int, eventType string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	envelope := Envelope{
		Type: eventType,
		Data: payload,
	}

	message, err := json.Marshal(envelope)
	if err != nil {
		domain.FromContext(ctx).Error(domain.FailToEncode, zap.Error(err))
		return err
	}

	h.SendToUser(ctx, userID, message)
	return nil
}

func (h *Hub) SendToUser(ctx context.Context, userID int, message []byte) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
	if ok {
		client.send <- message
		domain.FromContext(ctx).Info("Message sent")
	}
	domain.FromContext(ctx).Warn("WS client does not exist")
}

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

const pongWait = 60 * time.Second
const pingPeriod = 54 * time.Second
const writeWait = 10 * time.Second

//	func (c *Client) writePump() {
//		defer c.conn.Close()
//
//		for message := range c.send {
//			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
//			if err := c.conn.WriteMessage(WS.TextMessage, message); err != nil {
//				return
//			}
//		}
//	} без пинпонга
func (hub *Hub) writePump(ctx context.Context, c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		hub.RemoveClient(ctx, c.userID)
		domain.FromContext(ctx).Info("WS connection closed", zap.Int("userID", c.userID))
	}()

	for {
		select {
		case message, ok := <-c.send:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				domain.FromContext(ctx).Error("Failed set deadline to conn", zap.Error(err))
			}
			if !ok {
				err = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					domain.FromContext(ctx).Error("Failed to send closed message", zap.Error(err))
				}
				domain.FromContext(ctx).Info("Conn closed")
				return
			}

			err = c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				domain.FromContext(ctx).Error("Failed to send message", zap.Error(err))
				domain.FromContext(ctx).Info("Conn closed")
				return
			}

		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(pongWait))
			if err != nil {
				domain.FromContext(ctx).Error("Failed set deadline to conn", zap.Error(err))
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				domain.FromContext(ctx).Error("Failed to send ping message", zap.Error(err))
				return
			}
		}
	}
}
