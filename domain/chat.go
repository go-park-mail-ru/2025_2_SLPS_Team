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

type ChatService interface {
	GetOrCreateChatWithUser(ctx context.Context, selfUserID int, userID int) (int, error)
	GetMessagesByChatId(ctx context.Context, params PaginateQueryParams, userID int, chatID int) (*MessagesWithAuthors, error)
	CreateMessage(ctx context.Context, userID int, chatID int, message Message) (int, error)
	GetUserChats(ctx context.Context, userID int, params PaginateQueryParams) ([]FullChat, error)
}

type ChatStore interface {
	//CreateChat(chat Chat) error
	GetOtherChatMembersIdByAuthorId(ctx context.Context, userID int, chatID int) ([]int, error)
	GetOrCreateChatWithUser(ctx context.Context, selfUserID int, userID int) (int, error)
	IsMemberOfChat(ctx context.Context, userID int, chatID int) (bool, error)
	IsChatExist(ctx context.Context, chatID int) (bool, error)
	GetUserFullChats(ctx context.Context, userID int, limit, offset int) ([]FullChat, error)
	GetFullChatByIDAndSenderID(ctx context.Context, userID int, chatID int) (*FullChat, error)
	//GetChatMembers(chatID int, limit int, offset int) ([]ShortProfile, error)
}
