package domain

type Chat struct {
	ID      int `json:"id"`
	Members []ShortProfile
}
type ChatStore interface {
	//CreateChat(chat Chat) error
	//GetOtherChatMembersIdByAuthorId(userID int, chatID int) ([]int, error)
	GetOrCreateChatWithUser(selfUserID int, userID int) (int, error)
	IsMemberOfChat(userID int, chatID int) (bool, error)
	IsChatExist(chatID int) (bool, error)
	//GetChatMembers(chatID int, limit int, offset int) ([]ShortProfile, error)
}
