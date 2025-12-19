package domain

import "context"

//easyjson:json
type Chat struct {
	ID      int32 `json:"id"`
	Members []ShortProfile
}

//easyjson:json
type FullChat struct {
	ID                int32        `json:"id"`
	IsGroup           bool         `json:"isGroup"`
	UserIDWith        *int32       `json:"-"`
	AvatarPath        *string      `json:"avatarPath"`
	Name              *string      `json:"name"`
	LastMessage       Message      `json:"lastMessage"`
	LastMessageAuthor ShortProfile `json:"lastMessageAuthor"`
	LastReadMessageID int32        `json:"lastReadMessageID"`
	UnreadCounts      int32        `json:"unReadCounts"`
}

//easyjson:json
type FullChats []FullChat

//easyjson:json
type MemberWithLastReadMessage struct {
	MemberID          int32 `json:"memeberID"`
	LastReadMessageID int32 `json:"lastReadMessageID"`
	UnreadCounts      int32 `json:"unReadCounts"`
}

//easyjson:json
type ChatIDResponse struct {
	ChatID int32 `json:"chatID"`
}

//easyjson:json
type MessageIDResponse struct {
	MessageID int32 `json:"messageID"`
}

//easyjson:json
type UpdateLastReadRequest struct {
	LastReadMessageID int32 `json:"lastReadMessageID"`
}

type ChatService interface {
	GetOrCreateChatWithUser(ctx context.Context, selfUserID int32, userID int32) (int32, error)
	GetMessagesByChatId(ctx context.Context, params PaginateQueryParams, userID int32, chatID int32) (*MessagesWithAuthors, error)
	CreateMessage(ctx context.Context, userID int32, chatID int32, text string, attachmentFiles []*File, stickerID *int32) (int32, error)
	GetUserChats(ctx context.Context, userID int32, params PaginateQueryParams) ([]FullChat, error)
	UpdateLastReadMessage(ctx context.Context, userID, chatID, lastReadMessageID int32) error
}

type ChatStore interface {
	//CreateChat(chat Chat) error
	GetOtherChatMembersIdByAuthorId(ctx context.Context, userID int32, chatID int32) ([]MemberWithLastReadMessage, error)
	GetOrCreateChatWithUser(ctx context.Context, selfUserID int32, userID int32) (int32, error)
	IsMemberOfChat(ctx context.Context, userID int32, chatID int32) (bool, error)
	IsChatExist(ctx context.Context, chatID int32) (bool, error)
	GetUserFullChats(ctx context.Context, userID int32, limit, offset int32) ([]FullChat, []int32, error)
	GetFullChatByIDAndSenderID(ctx context.Context, userID int32, chatID int32) (*FullChat, []int32, error)
	UpdateLastReadMessageByUserIDAndChatID(ctx context.Context, userID, chatID, lastReadMessageID int32) error
	//GetChatMembers(chatID int32, limit int32, offset int32) ([]ShortProfile, error)
}
