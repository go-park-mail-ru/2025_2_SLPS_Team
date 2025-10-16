package domain

import "time"

type Message struct {
	ID        int       `json:"id"`
	AuthorID  int       `json:"authorID"`
	ChatID    int       `json:"chatID"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	//IsEdited bool `json:"isEdited"`
}
type MessageWithAuthors struct {
	Message []Message
	Authors []ShortProfile
}
type MessageStore interface {
	CreateMessage(message Message) (int, error)
	//UpdateMessage(text string, messageID int) (int, error)
	GetMessagesByChatId(chatID int, limit int, offset int) ([]Message, error)
}
