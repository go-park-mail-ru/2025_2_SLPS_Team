package domain

import (
	"context"
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	AuthorID  int       `json:"authorID"`
	ChatID    int       `json:"chatID"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	//IsEdited bool `json:"isEdited"`
}
type MessagesWithAuthors struct {
	Messages []Message
	Authors  map[int]ShortProfile
}
type MessageStore interface {
	CreateMessage(ctx context.Context, message Message) (int, error)
	//UpdateMessage(text string, messageID int) (int, error)
	GetMessagesByChatId(ctx context.Context, chatID int, limit int, offset int) ([]Message, error)
}
