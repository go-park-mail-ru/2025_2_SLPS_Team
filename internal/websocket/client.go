package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int]*Client),
		mu:      sync.RWMutex{},
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Hub) removeClient(userID int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		client.conn.Close()
		delete(h.clients, userID)
	}
}
func (h *Hub) SendJSON(userID int, eventType string, data interface{}) error {
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
		return err
	}

	h.sendToUser(userID, message)
	return nil
}

func (h *Hub) sendToUser(userID int, message []byte) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
	if ok {
		client.send <- message
	}
}

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

const pongWait = 60 * time.Second
const pingPeriod = 54 * time.Second
const writeWait = 10 * time.Second

func (c *Client) readPump(hub *Hub, router *Router) {
	defer func() {
		hub.removeClient(c.userID)
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var envelope Envelope
		if err := json.Unmarshal(message, &envelope); err != nil {
			continue
		}

		ctx := &WSContext{
			Conn:   c.conn,
			UserID: c.userID,
			Hub:    hub,
			Data:   envelope.Data,
		}

		if err := router.Route(envelope.Type, ctx); err != nil {
			log.Println("Route error:", err)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, router *Router, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	userID, _ := r.Context().Value("userID").(int)

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	hub.mu.Lock()
	hub.clients[userID] = client
	hub.mu.Unlock()

	go client.writePump()
	go client.readPump(hub, router)
}
