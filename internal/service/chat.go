package service

import (
	"context"
	"encoding/json"
	"project/domain"

	"go.uber.org/zap"
)

type ChatService struct {
	userStore    domain.UserStore
	profileStore domain.ProfileStore
	chatStore    domain.ChatStore
	messageStore domain.MessageStore
	wsHub        domain.WSHub
}

func NewChatService(userStore domain.UserStore, profileStore domain.ProfileStore, chatStore domain.ChatStore, messageStore domain.MessageStore, wsHub domain.WSHub) domain.ChatService {
	return &ChatService{
		userStore:    userStore,
		profileStore: profileStore,
		chatStore:    chatStore,
		messageStore: messageStore,
		wsHub:        wsHub,
	}
}

func (api *ChatService) GetOrCreateChatWithUser(ctx context.Context, selfUserID int, userID int) (int, error) {

	isUserExist, err := api.userStore.IsUserExists(ctx, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check user existence", zap.Error(err))
		return 0, domain.ErrDB
	}
	if !isUserExist {
		domain.FromContext(ctx).Warn("User not found")
		return 0, domain.ErrNotExist
	}
	if userID == selfUserID {
		domain.FromContext(ctx).Warn("Failed to create or get chat with same self user")
		return 0, domain.ErrInvalidInput
	}
	chatID, err := api.chatStore.GetOrCreateChatWithUser(ctx, selfUserID, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to create or get chat with user", zap.Error(err))
		return 0, domain.ErrDB
	}

	domain.FromContext(ctx).Info("Chat created or retrieved", zap.Int("chatID", chatID), zap.Int("chatWithUserID", userID))
	return chatID, nil
}

func (api *ChatService) GetMessagesByChatId(ctx context.Context, params domain.PaginateQueryParams, userID int, chatID int) (*domain.MessagesWithAuthors, error) {

	isMember, err := api.chatStore.IsMemberOfChat(ctx, userID, chatID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check membership", zap.Error(err), zap.Int("chatID", chatID))
		return nil, domain.ErrDB
	}
	if !isMember {
		domain.FromContext(ctx).Warn("User not a member of chat", zap.Int("chatID", chatID))
		return nil, domain.ErrAccessDenied
	}

	messages, err := api.messageStore.GetMessagesByChatId(ctx, chatID, params.Limit, params.Offset)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get messages", zap.Error(err), zap.Int("chatID", chatID))
		return nil, domain.ErrDB
	}

	mapIDs := make(map[int]struct{})
	for _, msg := range messages {
		mapIDs[msg.AuthorID] = struct{}{}
	}

	authorIDs := make([]int, 0, len(mapIDs))
	for id := range mapIDs {
		authorIDs = append(authorIDs, id)
	}

	authors, err := api.profileStore.GetShortProfileByUserIDs(ctx, authorIDs)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get authors", zap.Error(err), zap.Ints("authorIDs", authorIDs))
		return nil, domain.ErrDB
	}

	messagesWithAuthors := domain.MessagesWithAuthors{
		Messages: messages,
		Authors:  authors,
	}
	domain.FromContext(ctx).Info("Messages retrieved successfully", zap.Int("chatID", chatID), zap.Int("limit", params.Limit), zap.Int("offset", params.Offset))
	return &messagesWithAuthors, nil
}

func (api *ChatService) CreateMessage(ctx context.Context, userID int, chatID int, message domain.Message) (int, error) {
	exits, err := api.chatStore.IsChatExist(ctx, chatID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get chat", zap.Error(err), zap.Int("chatID", chatID))
		return 0, domain.ErrDB
	}

	if !exits {
		domain.FromContext(ctx).Warn("Chat not found", zap.Int("chatID", chatID))
		return 0, domain.ErrNotExist
	}
	message.AuthorID = userID
	message.ChatID = chatID
	messageID, err := api.messageStore.CreateMessage(ctx, message)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to create message", zap.Error(err), zap.Int("chatID", chatID))
		return 0, domain.ErrDB
	}
	message.ID = messageID
	go func(ctx context.Context, userId int, chatID int) {
		chat, err := api.chatStore.GetFullChatByIDAndSenderID(ctx, userID, chatID)
		if err != nil {
			domain.FromContext(ctx).Error("Fail to get chat", zap.Error(err))
			return
		}
		recipients, err := api.chatStore.GetOtherChatMembersIdByAuthorId(ctx, userID, chatID)
		if err != nil {
			domain.FromContext(ctx).Error("Fail to get recipients", zap.Error(err))
			return
		}
		data, err := json.Marshal(chat)
		if err != nil {
			domain.FromContext(ctx).Error("Fail to marshal chat", zap.Error(err))
			return
		}
		response := domain.Envelope{
			Type: "new_message",
			Data: data,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			domain.FromContext(ctx).Error("Fail to marshal chat", zap.Error(err))
			return
		}

		for _, recipient := range recipients {
			api.wsHub.SendToUser(ctx, recipient, jsonResponse)
		}

		domain.FromContext(ctx).Info("message send to recipients")
	}(ctx, userID, chatID)

	domain.FromContext(ctx).Info("Message created successfully", zap.Int("messageID", messageID), zap.Int("chatID", chatID))
	return messageID, nil
}

func (api *ChatService) GetUserChats(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.FullChat, error) {

	chats, err := api.chatStore.GetUserFullChats(ctx, userID, params.Limit, params.Offset)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get chats", zap.Error(err))
		return nil, domain.ErrDB
	}

	domain.FromContext(ctx).Info("Chats retrieved successfully", zap.Int("limit", params.Limit), zap.Int("offset", params.Offset))
	return chats, nil
}
