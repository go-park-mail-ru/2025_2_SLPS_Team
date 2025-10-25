package domain

import "context"

type Chat struct {
	ID      int `json:"id"`
	Members []ShortProfile
}
type ChatStore interface {
	//CreateChat(chat Chat) error
	//GetOtherChatMembersIdByAuthorId(userID int, chatID int) ([]int, error)
	GetOrCreateChatWithUser(ctx context.Context, selfUserID int, userID int) (int, error)
	IsMemberOfChat(ctx context.Context, userID int, chatID int) (bool, error)
	IsChatExist(ctx context.Context, chatID int) (bool, error)
	//GetChatMembers(chatID int, limit int, offset int) ([]ShortProfile, error)
}
