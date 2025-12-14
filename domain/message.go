package domain

import (
	"context"
	"time"
)

type Message struct {
	ID          int32     `json:"id"`
	AuthorID    int32     `json:"authorID"`
	ChatID      int32     `json:"chatID"`
	Text        string    `json:"text"`
	CreatedAt   time.Time `json:"createdAt"`
	Attachments []string  `json:"attachments,omitempty"`
	//IsEdited bool `json:"isEdited"`
}
type MessagesWithAuthors struct {
	Messages []Message
	Authors  map[int32]ShortProfile
}
type MessageStore interface {
	CreateMessage(ctx context.Context, message Message) (int32, error)
	//UpdateMessage(text string, messageID int32) (int32, error)
	GetMessagesByChatId(ctx context.Context, chatID int32, limit int32, offset int32) ([]Message, error)
	GetMessageAttachments(ctx context.Context, messageID int32) ([]string, error)
	SaveMessageAttachments(ctx context.Context, messageID int32, attachments []string) error
}
