package domain

import "context"

type Chat struct {
	ID      int `json:"id"`
	Members []ShortProfile
}
type FullChat struct {
	ID                int          `json:"id"`
	IsGroup           bool         `json:"isGroup"`
	AvatarPath        *string      `json:"avatarPath"`
	Name              string       `json:"name"`
	LastMessage       Message      `json:"lastMessage"`
	LastMessageAuthor ShortProfile `json:"lastMessageAuthor"`
}

type ChatStore interface {
	//CreateChat(chat Chat) error
	//GetOtherChatMembersIdByAuthorId(userID int, chatID int) ([]int, error)
	GetOrCreateChatWithUser(ctx context.Context, selfUserID int, userID int) (int, error)
	IsMemberOfChat(ctx context.Context, userID int, chatID int) (bool, error)
	IsChatExist(ctx context.Context, chatID int) (bool, error)
	GetUserFullChats(ctx context.Context, userID int, limit, offset int) ([]FullChat, error)
	//GetChatMembers(chatID int, limit int, offset int) ([]ShortProfile, error)
}
