package service

import (
	"context"
	"errors"
	"project/domain"
	"project/internal/repository/mocks"
	"project/shared/pb"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func newChatServiceMocks(t *testing.T) (*ChatService, *mocks.MockChatStore, *mocks.MockMessageStore, domain.WSHub, *gomock.Controller) {
	ctrl := gomock.NewController(t)

	// Создаем моки для gRPC клиентов
	authConn := &grpc.ClientConn{}
	profileConn := &grpc.ClientConn{}
	authClient := pb.NewAuthServiceClient(authConn)
	profileClient := pb.NewProfileServiceClient(profileConn)

	chatStore := mocks.NewMockChatStore(ctrl)
	messageStore := mocks.NewMockMessageStore(ctrl)
	wsHub := NewHub()

	svc := &ChatService{
		authService:    authClient,
		profileService: profileClient,
		chatStore:      chatStore,
		messageStore:   messageStore,
		wsHub:          wsHub,
	}
	return svc, chatStore, messageStore, wsHub, ctrl
}

func TestChatService_GetOrCreateChatWithUser(t *testing.T) {
	svc, chatStore, _, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		chatStore.EXPECT().GetOrCreateChatWithUser(ctx, int32(1), int32(2)).Return(int32(42), nil)

		chatID, err := svc.GetOrCreateChatWithUser(ctx, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, int32(42), chatID)
	})

	t.Run("Same userID", func(t *testing.T) {
		chatID, err := svc.GetOrCreateChatWithUser(ctx, 1, 1)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Equal(t, int32(0), chatID)
	})

	t.Run("DB error", func(t *testing.T) {
		chatStore.EXPECT().GetOrCreateChatWithUser(ctx, int32(1), int32(2)).Return(int32(0), errors.New("db error"))
		chatID, err := svc.GetOrCreateChatWithUser(ctx, 1, 2)
		assert.Error(t, err)
		assert.Equal(t, int32(0), chatID)
	})
}

func TestChatService_GetMessagesByChatId(t *testing.T) {
	svc, chatStore, messageStore, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		chatID := int32(10)
		userID := int32(1)
		messages := []domain.Message{
			{ID: 1, AuthorID: 2, ChatID: chatID, Text: "Hello"},
			{ID: 2, AuthorID: 3, ChatID: chatID, Text: "World"},
		}

		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(true, nil)
		messageStore.EXPECT().GetMessagesByChatId(ctx, chatID, int32(20), int32(0)).Return(messages, nil)

		params := domain.PaginateQueryParams{Limit: 20, Page: 0}
		result, err := svc.GetMessagesByChatId(ctx, params, userID, chatID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Messages, 2)
	})

	t.Run("Not member", func(t *testing.T) {
		chatID := int32(10)
		userID := int32(1)

		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(false, nil)

		params := domain.PaginateQueryParams{Limit: 20, Page: 0}
		result, err := svc.GetMessagesByChatId(ctx, params, userID, chatID)

		assert.ErrorIs(t, err, domain.ErrAccessDenied)
		assert.Nil(t, result)
	})
}

func TestChatService_CreateMessage(t *testing.T) {
	svc, chatStore, messageStore, _, ctrl := newChatServiceMocks(t) // Заменили wsHub на _
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		chatID := int32(10)
		userID := int32(1)
		message := domain.Message{Text: "Hello"}
		messageID := int32(100)

		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(true, nil)
		messageStore.EXPECT().CreateMessage(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, msg domain.Message) (int32, error) {
				assert.Equal(t, userID, msg.AuthorID)
				assert.Equal(t, chatID, msg.ChatID)
				assert.Equal(t, "Hello", msg.Text)
				return messageID, nil
			},
		)

		// Игнорируем вызовы в горутине для упрощения теста
		chatStore.EXPECT().GetFullChatByIDAndSenderID(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		chatStore.EXPECT().GetOtherChatMembersIdByAuthorId(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		resultID, err := svc.CreateMessage(ctx, userID, chatID, message)

		assert.NoError(t, err)
		assert.Equal(t, messageID, resultID)
	})

	t.Run("Chat not exists", func(t *testing.T) {
		chatID := int32(10)
		userID := int32(1)
		message := domain.Message{Text: "Hello"}

		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(false, nil)

		resultID, err := svc.CreateMessage(ctx, userID, chatID, message)

		assert.ErrorIs(t, err, domain.ErrNotExist)
		assert.Equal(t, int32(0), resultID)
	})

	t.Run("DB error on chat check", func(t *testing.T) {
		chatID := int32(10)
		userID := int32(1)
		message := domain.Message{Text: "Hello"}

		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(false, errors.New("db error"))

		resultID, err := svc.CreateMessage(ctx, userID, chatID, message)

		assert.Error(t, err)
		assert.Equal(t, int32(0), resultID)
	})

	t.Run("DB error on message creation", func(t *testing.T) {
		chatID := int32(10)
		userID := int32(1)
		message := domain.Message{Text: "Hello"}

		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(true, nil)
		messageStore.EXPECT().CreateMessage(ctx, gomock.Any()).Return(int32(0), errors.New("db error"))

		resultID, err := svc.CreateMessage(ctx, userID, chatID, message)

		assert.Error(t, err)
		assert.Equal(t, int32(0), resultID)
	})
}

func TestChatService_UpdateLastReadMessage(t *testing.T) {
	svc, chatStore, _, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		lastReadMessageID := int32(100)

		chatStore.EXPECT().UpdateLastReadMessageByUserIDAndChatID(ctx, userID, chatID, lastReadMessageID).Return(nil)

		err := svc.UpdateLastReadMessage(ctx, userID, chatID, lastReadMessageID)
		assert.NoError(t, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		lastReadMessageID := int32(100)

		chatStore.EXPECT().UpdateLastReadMessageByUserIDAndChatID(ctx, userID, chatID, lastReadMessageID).Return(errors.New("db error"))

		err := svc.UpdateLastReadMessage(ctx, userID, chatID, lastReadMessageID)
		assert.Error(t, err)
	})
}
