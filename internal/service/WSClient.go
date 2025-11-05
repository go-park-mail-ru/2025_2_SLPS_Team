package service

import (
	"context"
	"encoding/json"
	"project/domain"
	"strings"
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
	go h.readPump(ctx, client)

	domain.FromContext(ctx).Info("WS client added", zap.Int("userID", userID))
}

func (h *Hub) RemoveClient(ctx context.Context, userID int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.clients[userID]
	if !ok {
		domain.FromContext(ctx).Warn("Client does not exist", zap.Int("userID", userID))
		return
	}

	_ = client.conn.Close()
	close(client.send)
	delete(h.clients, userID)

	domain.FromContext(ctx).Info("WS client removed successfully", zap.Int("userID", userID))
}

func (h *Hub) SendToUser(ctx context.Context, userID int, message []byte) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		domain.FromContext(ctx).Warn("WS client does not exist", zap.Int("userID", userID))
		return
	}

	select {
	case client.send <- message:
		domain.FromContext(ctx).Debug("WS message sent", zap.Int("userID", userID))
	default:
		domain.FromContext(ctx).Warn("WS send channel full — dropping message", zap.Int("userID", userID))
	}
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

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func (hub *Hub) writePump(ctx context.Context, c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		hub.RemoveClient(ctx, c.userID)
		domain.FromContext(ctx).Info("WS connection closed", zap.Int("userID", c.userID))
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					domain.FromContext(ctx).Error("Failed to send message", zap.Error(err))
				}
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					domain.FromContext(ctx).Error("Failed to send ping message", zap.Error(err))
				}
				return
			}
		}
	}
}

func (hub *Hub) readPump(ctx context.Context, c *Client) {
	defer func() {
		hub.RemoveClient(ctx, c.userID)
		_ = c.conn.Close()
		domain.FromContext(ctx).Info("WS readPump closed", zap.Int("userID", c.userID))
	}()

	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				domain.FromContext(ctx).Error("Unexpected WS close", zap.Error(err))
			}
			break
		}
	}
}
