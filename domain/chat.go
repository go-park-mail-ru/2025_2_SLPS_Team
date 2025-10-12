package domain

type Chat struct {
	ID      int `json:"id"`
	Members []ShortProfile
}
type ChatStore interface {
	CreateChat(chat Chat) error
	GetOtherChatMembersIdByAuthorId(userID int, chatID int) ([]int, error)
}
