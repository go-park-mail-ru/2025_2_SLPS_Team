package websocket

import (
	"encoding/json"
	"project/domain"
	"time"
)

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

type ChatWSHandler struct {
	UserStore    domain.UserStore
	ChatStore    domain.ChatStore
	MessageStore domain.MessageStore
}

func NewWebSocketHandler(userStore domain.UserStore, chatStore domain.ChatStore, msgStore domain.MessageStore) *ChatWSHandler {
	return &ChatWSHandler{
		UserStore:    userStore,
		ChatStore:    chatStore,
		MessageStore: msgStore,
	}
}

type Message struct {
	ID        int       `json:"id"`
	AuthorID  int       `json:"authorID"`
	ChatID    int       `json:"chatID"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	//IsEdited bool `json:"isEdited"`
}

func (h *ChatWSHandler) HandleSendMessage() WSHandlerFunc {
	return func(ctx *WSContext) error {
		var payload struct {
			ChatID int    `json:"chat_id"`
			Text   string `json:"text"`
		}

		if err := json.Unmarshal(ctx.Data, &payload); err != nil {
			return err
		}

		msg := domain.Message{
			ChatID:    payload.ChatID,
			AuthorID:  ctx.UserID,
			Text:      payload.Text,
			CreatedAt: time.Now(),
		}
		msgId, err := h.MessageStore.CreateMessage(msg)
		if err != nil {
			return err
		}
		recipients, err := h.ChatStore.GetOtherChatMembersIdByAuthorId(msg.ChatID, msg.AuthorID)
		if err != nil {
			return err
		}
		msg.ID = msgId
		response := Envelope{
			Type: "new_message",
			Data: mustMarshal(msg),
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return err
		}
		for _, recipient := range recipients {
			ctx.Hub.sendToUser(recipient, jsonResponse)
		}

		return nil
	}
}
