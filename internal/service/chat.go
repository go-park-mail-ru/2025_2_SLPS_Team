package service

import (
	"context"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"
	"strings"
	"time"

	"go.uber.org/zap"
)

type ChatService struct {
	chatStore      domain.ChatStore
	authService    pb.AuthServiceClient
	profileService pb.ProfileServiceClient
	messageStore   domain.MessageStore
	wsHub          domain.WSHub
}

func NewChatService(authService pb.AuthServiceClient, profileService pb.ProfileServiceClient, chatStore domain.ChatStore, messageStore domain.MessageStore, wsHub domain.WSHub) domain.ChatService {
	return &ChatService{
		authService:    authService,
		chatStore:      chatStore,
		profileService: profileService,
		messageStore:   messageStore,
		wsHub:          wsHub,
	}
}

func (api *ChatService) GetOrCreateChatWithUser(ctx context.Context, selfUserID int32, userID int32) (int32, error) {
	resp, err := api.authService.IsUserExists(ctx, &pb.UserIDRequest{UserId: userID})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check user existence", zap.Error(err))
		return 0, domain.ErrDB
	}
	isUserExist := resp.Exists
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

	domain.FromContext(ctx).Info("Chat created or retrieved", zap.Int32("chatID", chatID), zap.Int32("chatWithUserID", userID))
	return chatID, nil
}

func (api *ChatService) GetMessagesByChatId(ctx context.Context, params domain.PaginateQueryParams, userID int32, chatID int32) (*domain.MessagesWithAuthors, error) {
	isMember, err := api.chatStore.IsMemberOfChat(ctx, userID, chatID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check membership", zap.Error(err), zap.Int32("chatID", chatID))
		return nil, domain.ErrDB
	}
	if !isMember {
		domain.FromContext(ctx).Warn("User not a member of chat", zap.Int32("chatID", chatID))
		return nil, domain.ErrAccessDenied
	}
	offset, limit := domain.ValidatePaginationParams(params)
	messages, err := api.messageStore.GetMessagesByChatId(ctx, chatID, limit, offset)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get messages", zap.Error(err), zap.Int32("chatID", chatID))
		return nil, domain.ErrDB
	}

	mapIDs := make(map[int32]struct{})
	for _, msg := range messages {
		mapIDs[msg.AuthorID] = struct{}{}
	}

	authorIDs := make([]int32, 0, len(mapIDs))
	for id := range mapIDs {
		authorIDs = append(authorIDs, id)
	}

	authors, err := api.profileService.GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{UserIDs: authorIDs})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get authors", zap.Error(err), zap.Int32s("authorIDs", authorIDs))
		return nil, domain.ErrDB
	}

	messagesWithAuthors := domain.MessagesWithAuthors{
		Messages: messages,
		Authors:  generated.FromProtoShortProfileMap(authors),
	}
	domain.FromContext(ctx).Info("Messages retrieved successfully", zap.Int32("chatID", chatID), zap.Int32("limit", limit), zap.Int32("offset", offset))
	return &messagesWithAuthors, nil
}

func (api *ChatService) CreateMessage(ctx context.Context, userID int32, chatID int32, text string, attachmentFiles []*domain.File, stickerID *int32) (int32, error) {
	// Валидация содержания сообщения
	if err := validateMessageContent(text, len(attachmentFiles), stickerID); err != nil {
		domain.FromContext(ctx).Warn("Invalid message content",
			zap.String("text", text),
			zap.Int("attachments", len(attachmentFiles)),
			zap.Any("stickerID", stickerID))
		return 0, err
	}

	// Проверяем существование чата
	exists, err := api.chatStore.IsChatExist(ctx, chatID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get chat", zap.Error(err), zap.Int32("chatID", chatID))
		return 0, domain.ErrDB
	}

	if !exists {
		domain.FromContext(ctx).Warn("Chat not found", zap.Int32("chatID", chatID))
		return 0, domain.ErrNotFound
	}

	// Проверяем, является ли пользователь участником чата
	isMember, err := api.chatStore.IsMemberOfChat(ctx, userID, chatID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check membership", zap.Error(err), zap.Int32("chatID", chatID))
		return 0, domain.ErrDB
	}
	if !isMember {
		domain.FromContext(ctx).Warn("User not a member of chat", zap.Int32("chatID", chatID))
		return 0, domain.ErrAccessDenied
	}

	// Обработка вложений (только если нет стикера)
	var attachmentPaths []string
	if len(attachmentFiles) > 0 && stickerID == nil {
		attachmentPaths, err = UploadFiles(attachmentFiles)
		if err != nil {
			domain.FromContext(ctx).Error("Failed to upload attachments", zap.Error(err))
			return 0, domain.ErrService
		}
	}

	// Создаем объект сообщения
	message := domain.Message{
		AuthorID:    userID,
		ChatID:      chatID,
		Text:        text, // Может быть пустым для стикера или только вложений
		StickerID:   stickerID,
		Attachments: attachmentPaths,
	}

	// Сохраняем сообщение в БД
	messageID, err := api.messageStore.CreateMessage(ctx, message)
	if err != nil {
		// Удаляем загруженные файлы в случае ошибки
		if len(attachmentPaths) > 0 {
			DeleteFiles(convertToPointerSlice(attachmentPaths))
		}
		domain.FromContext(ctx).Error("Failed to create message", zap.Error(err), zap.Int32("chatID", chatID))
		return 0, domain.ErrDB
	}

	// Отправляем сообщение через WebSocket
	go func(ctx context.Context, userID int32, chatID int32, messageID int32) {
		ctx2, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		// Получаем чат для отправки через WS
		chat, userIDs, err := api.chatStore.GetFullChatByIDAndSenderID(ctx2, userID, chatID)
		if err != nil {
			domain.FromContext(ctx).Error("Fail to get chat", zap.Error(err))
			return
		}

		// Получаем профили участников
		resp, err := api.profileService.GetShortProfileMapByUserIDs(ctx2, &pb.GetShortProfileMapByUserIDsRequest{UserIDs: userIDs})
		if err != nil {
			domain.FromContext(ctx).Error("Fail to get profiles", zap.Error(err))
			return
		}

		profiles := generated.FromProtoShortProfileMap(resp)
		if !chat.IsGroup {
			chat.AvatarPath = profiles[*chat.UserIDWith].AvatarPath
			name := profiles[*chat.UserIDWith].FullName
			chat.Name = &name
		}
		chat.LastMessageAuthor = profiles[chat.LastMessage.AuthorID]

		// Получаем получателей
		recipients, err := api.chatStore.GetOtherChatMembersIdByAuthorId(ctx2, userID, chatID)
		if err != nil {
			domain.FromContext(ctx).Error("Fail to get recipients", zap.Error(err))
			return
		}

		// Отправляем сообщение всем получателям через WS
		for _, recipient := range recipients {
			chat.LastReadMessageID = recipient.LastReadMessageID
			chat.UnreadCounts = recipient.UnreadCounts
			err = api.wsHub.SendJSON(ctx2, recipient.MemberID, "new_message", chat)
			if err != nil {
				domain.FromContext(ctx).Error("Fail to marshal chat", zap.Error(err))
			}
		}

		domain.FromContext(ctx).Info("message send to recipients")
	}(ctx, userID, chatID, messageID)

	domain.FromContext(ctx).Info("Message created successfully",
		zap.Int32("messageID", messageID),
		zap.Int32("chatID", chatID),
		zap.Any("stickerID", stickerID),
		zap.Bool("hasText", text != ""),
		zap.Int("attachmentsCount", len(attachmentPaths)))

	return messageID, nil
}

func (api *ChatService) GetUserChats(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.FullChat, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	chats, userIDs, err := api.chatStore.GetUserFullChats(ctx, userID, limit, offset)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get chats", zap.Error(err))
		return nil, domain.ErrDB
	}

	resp, err := api.profileService.GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{UserIDs: userIDs})
	if err != nil {
		domain.FromContext(ctx).Error("Fail to get profiles", zap.Error(err))
		return nil, err
	}

	profiles := generated.FromProtoShortProfileMap(resp)
	for i := range chats {
		chat := &chats[i]
		if !chat.IsGroup && chat.UserIDWith != nil {
			chat.AvatarPath = profiles[*chat.UserIDWith].AvatarPath
			name := profiles[*chat.UserIDWith].FullName
			chat.Name = &name
		}
		chat.LastMessageAuthor = profiles[chat.LastMessage.AuthorID]
	}

	domain.FromContext(ctx).Info("Chats retrieved successfully", zap.Int32("limit", limit), zap.Int32("offset", offset))
	return chats, nil
}

func (api *ChatService) UpdateLastReadMessage(ctx context.Context, userID, chatID, lastReadMessageID int32) error {
	err := api.chatStore.UpdateLastReadMessageByUserIDAndChatID(ctx, userID, chatID, lastReadMessageID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed update last read message", zap.Error(err))
		return domain.ErrDB
	}

	domain.FromContext(ctx).Info("last read message updated")
	return nil
}

func validateMessageContent(text string, attachmentsCount int, stickerID *int32) error {
	if stickerID != nil {
		// Сообщение со стикером
		trimmedText := strings.TrimSpace(text)
		if trimmedText != "" {
			return domain.ErrInvalidInput
		}
		if attachmentsCount > 0 {
			return domain.ErrInvalidInput
		}
		return nil
	} else {
		// Сообщение без стикера
		trimmedText := strings.TrimSpace(text)
		if trimmedText == "" && attachmentsCount == 0 {
			return domain.ErrInvalidInput
		}
		return nil
	}
}
