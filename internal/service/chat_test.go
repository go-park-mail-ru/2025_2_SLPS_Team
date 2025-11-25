package service

import (
	"context"
	"errors"
	"project/domain"
	"project/internal/repository/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newChatServiceMocks(t *testing.T) (*ChatService,
	*mocks.MockUserStore,
	*mocks.MockProfileStore,
	*mocks.MockChatStore,
	*mocks.MockMessageStore,
	domain.WSHub,
	*gomock.Controller) {

	ctrl := gomock.NewController(t)
	userStore := mocks.NewMockUserStore(ctrl)
	profileStore := mocks.NewMockProfileStore(ctrl)
	chatStore := mocks.NewMockChatStore(ctrl)
	messageStore := mocks.NewMockMessageStore(ctrl)
	wsHub := NewHub()
	svc := &ChatService{
		userStore:    userStore,
		profileStore: profileStore,
		chatStore:    chatStore,
		messageStore: messageStore,
		wsHub:        wsHub,
	}
	return svc, userStore, profileStore, chatStore, messageStore, wsHub, ctrl
}

func TestChatService_GetOrCreateChatWithUser(t *testing.T) {
	svc, userStore, _, chatStore, _, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userStore.EXPECT().IsUserExists(ctx, 2).Return(true, nil)
		chatStore.EXPECT().GetOrCreateChatWithUser(ctx, 1, 2).Return(42, nil)
		chatID, err := svc.GetOrCreateChatWithUser(ctx, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, 42, chatID)
	})

	t.Run("User not exists", func(t *testing.T) {
		userStore.EXPECT().IsUserExists(ctx, 2).Return(false, nil)
		chatID, err := svc.GetOrCreateChatWithUser(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotExist)
		assert.Equal(t, 0, chatID)
	})

	t.Run("Same userID", func(t *testing.T) {
		userStore.EXPECT().IsUserExists(ctx, 1).Return(true, nil)
		chatID, err := svc.GetOrCreateChatWithUser(ctx, 1, 1)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Equal(t, 0, chatID)
	})

	t.Run("DB error", func(t *testing.T) {
		userStore.EXPECT().IsUserExists(ctx, 2).Return(false, errors.New("db"))
		chatID, err := svc.GetOrCreateChatWithUser(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Equal(t, 0, chatID)
	})
}

func TestChatService_GetMessagesByChatId(t *testing.T) {
	svc, _, profileStore, chatStore, messageStore, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()
	chatID := 10
	userID := 1
	messages := []domain.Message{{ID: 1, AuthorID: 2, ChatID: chatID, Text: "Hi"}}
	authors := map[int32]domain.ShortProfile{2: {UserID: 2, FullName: "user2"}}

	t.Run("Success", func(t *testing.T) {
		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(true, nil)
		messageStore.EXPECT().GetMessagesByChatId(ctx, chatID, 10, 0).Return(messages, nil)
		profileStore.EXPECT().GetShortProfileByUserIDs(ctx, []int32{2}).Return(authors, nil)
		res, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{Limit: 10, Page: 1}, userID, chatID)
		assert.NoError(t, err)
		assert.Len(t, res.Messages, 1)
		assert.Len(t, res.Authors, 1)
	})

	t.Run("Not member", func(t *testing.T) {
		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(false, nil)
		res, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{Limit: 10, Page: 1}, userID, chatID)
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
		assert.Nil(t, res)
	})

	t.Run("Get messages error", func(t *testing.T) {
		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(true, nil)
		messageStore.EXPECT().GetMessagesByChatId(ctx, chatID, 10, 0).Return(nil, errors.New("db"))
		res, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{Limit: 10, Page: 1}, userID, chatID)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})

	t.Run("Get authors error", func(t *testing.T) {
		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(true, nil)
		messageStore.EXPECT().GetMessagesByChatId(ctx, chatID, 10, 0).Return(messages, nil)
		profileStore.EXPECT().GetShortProfileByUserIDs(ctx, []int32{2}).Return(nil, errors.New("db"))
		res, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{Limit: 10, Page: 1}, userID, chatID)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestChatService_CreateMessage(t *testing.T) {
	svc, _, _, chatStore, messageStore, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()
	chatID := 10
	userID := 1
	msg := domain.Message{Text: "Hi"}

	t.Run("Success", func(t *testing.T) {
		done := make(chan struct{})
		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(true, nil)
		messageStore.EXPECT().CreateMessage(ctx, gomock.Any()).Return(42, nil)
		chatStore.EXPECT().GetFullChatByIDAndSenderID(ctx, userID, chatID).Return(&domain.FullChat{ID: chatID}, nil)
		chatStore.EXPECT().GetOtherChatMembersIdByAuthorId(ctx, userID, chatID).DoAndReturn(func(_ context.Context, _ int32, _ int32) ([]int32, error) {
			close(done)
			return []int32{2}, nil
		})
		msgID, err := svc.CreateMessage(ctx, userID, chatID, msg)
		assert.NoError(t, err)
		assert.Equal(t, 42, msgID)
		<-done
	})

	t.Run("Chat not exist", func(t *testing.T) {
		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(false, nil)
		msgID, err := svc.CreateMessage(ctx, userID, chatID, msg)
		assert.ErrorIs(t, err, domain.ErrNotExist)
		assert.Equal(t, 0, msgID)
	})

	t.Run("DB error", func(t *testing.T) {
		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(false, errors.New("db"))
		msgID, err := svc.CreateMessage(ctx, userID, chatID, msg)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Equal(t, 0, msgID)
	})
}

func TestChatService_GetUserChats(t *testing.T) {
	svc, _, _, chatStore, _, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()
	userID := 1
	chats := []domain.FullChat{{ID: 1}, {ID: 2}}

	t.Run("Success", func(t *testing.T) {
		chatStore.EXPECT().GetUserFullChats(ctx, userID, 10, 0).Return(chats, nil)
		res, err := svc.GetUserChats(ctx, userID, domain.PaginateQueryParams{Limit: 10, Page: 0})
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})

	t.Run("DB error", func(t *testing.T) {
		chatStore.EXPECT().GetUserFullChats(ctx, userID, 10, 0).Return(nil, errors.New("db"))
		res, err := svc.GetUserChats(ctx, userID, domain.PaginateQueryParams{Limit: 10, Page: 0})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}
