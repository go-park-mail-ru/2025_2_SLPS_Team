package WS

import (
	"encoding/json"
	"log"
	"project/domain"
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

func NewHub() domain.WSHub {
	return &Hub{
		clients: make(map[int]*Client),
	}
}

func (h *Hub) RemoveClient(userID int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		client.conn.Close()
		delete(h.clients, userID)
	}
}

func (h *Hub) AddClient(userID int, conn *websocket.Conn) {
	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256), // буфер на случай bursts сообщений
		userID: userID,
	}

	h.mu.Lock()
	h.clients[userID] = client
	h.mu.Unlock()

	go h.writePump(client)
	println("client added")
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

	h.SendToUser(userID, message)
	return nil
}

func (h *Hub) SendToUser(userID int, message []byte) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
	if ok {
		log.Println("сообщение отправлено")
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
func (hub *Hub) writePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		hub.RemoveClient(c.userID)
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
