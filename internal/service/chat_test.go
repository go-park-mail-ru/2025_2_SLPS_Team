package service

import (
	"context"
	"errors"
	"project/domain"
	repo_mocks "project/internal/repository/mocks"
	grpc_mocks "project/internal/service/mocks"
	pb "project/shared/pb"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newChatServiceMocks(t *testing.T) (*ChatService,
	*repo_mocks.MockChatStore,
	*repo_mocks.MockMessageStore,
	*grpc_mocks.MockAuthServiceClient,
	*grpc_mocks.MockProfileServiceClient,
	*gomock.Controller) {

	ctrl := gomock.NewController(t)
	chatStore := repo_mocks.NewMockChatStore(ctrl)
	messageStore := repo_mocks.NewMockMessageStore(ctrl)
	authService := grpc_mocks.NewMockAuthServiceClient(ctrl)
	profileService := grpc_mocks.NewMockProfileServiceClient(ctrl)
	svc := &ChatService{
		chatStore:      chatStore,
		authService:    authService,
		profileService: profileService,
		messageStore:   messageStore,
	}
	return svc, chatStore, messageStore, authService, profileService, ctrl
}

var ava = "123"

func TestChatService_GetOrCreateChatWithUser(t *testing.T) {
	svc, chatStore, _, authService, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success - create new chat", func(t *testing.T) {
		selfUserID := int32(2)
		userID := int32(1)
		chatID := int32(100)

		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: userID}).Return(
			&pb.UserExistsResponse{Exists: true},
			nil,
		)

		chatStore.EXPECT().GetOrCreateChatWithUser(ctx, selfUserID, userID).Return(
			chatID,
			nil,
		)

		result, err := svc.GetOrCreateChatWithUser(ctx, selfUserID, userID)
		assert.NoError(t, err)
		assert.Equal(t, chatID, result)
	})

	t.Run("User not exists", func(t *testing.T) {
		selfUserID := int32(999)
		userID := int32(1)

		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: userID}).Return(
			&pb.UserExistsResponse{Exists: false},
			nil,
		)

		result, err := svc.GetOrCreateChatWithUser(ctx, selfUserID, userID)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrNotExist, err)
	})

	t.Run("DB error on user check", func(t *testing.T) {
		selfUserID := int32(1)
		userID := int32(2)

		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: userID}).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.GetOrCreateChatWithUser(ctx, selfUserID, userID)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("DB error on chat creation", func(t *testing.T) {
		selfUserID := int32(1)
		userID := int32(2)

		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: userID}).Return(
			&pb.UserExistsResponse{Exists: true},
			nil,
		)

		chatStore.EXPECT().GetOrCreateChatWithUser(ctx, selfUserID, userID).Return(
			int32(0),
			errors.New("db error"),
		)

		result, err := svc.GetOrCreateChatWithUser(ctx, selfUserID, userID)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestChatService_GetMessagesByChatId(t *testing.T) {
	svc, chatStore, messageStore, _, profileService, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		limit := int32(20)
		offset := int32(0)

		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(true, nil)

		messages := []domain.Message{
			{
				ID:       int32(100),
				AuthorID: int32(2),
				ChatID:   chatID,
				Text:     "Hello",
			},
			{
				ID:       int32(101),
				AuthorID: int32(3),
				ChatID:   chatID,
				Text:     "Hi",
			},
		}

		messageStore.EXPECT().GetMessagesByChatId(ctx, chatID, limit, offset).Return(
			messages,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(2), int32(3)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(2): {
						UserID:     int32(2),
						FullName:   "John Doe",
						AvatarPath: &ava,
						Dob:        timestamppb.Now(),
					},
					int32(3): {
						UserID:     int32(3),
						FullName:   "Jane Doe",
						AvatarPath: &ava,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{
			Limit: limit,
			Page:  1,
		}, userID, chatID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Messages, 2)
		assert.Len(t, result.Authors, 2)
		assert.Equal(t, "John Doe", result.Authors[2].FullName)
	})

	t.Run("Not a member", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)

		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(false, nil)

		result, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{}, userID, chatID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	t.Run("DB error on membership check", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)

		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(
			false,
			errors.New("db error"),
		)

		result, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{}, userID, chatID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("DB error on get messages", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)

		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(true, nil)

		messageStore.EXPECT().GetMessagesByChatId(ctx, chatID, gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetMessagesByChatId(ctx, domain.PaginateQueryParams{}, userID, chatID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

}

func TestChatService_CreateMessage(t *testing.T) {
	svc, chatStore, messageStore, _, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Invalid message - empty with no attachments/sticker", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)

		// Проверки чата не должны вызываться при невалидном сообщении
		result, err := svc.CreateMessage(ctx, userID, chatID, "", nil, nil)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Not a member", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		text := "Hello"

		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(true, nil)
		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(false, nil)

		result, err := svc.CreateMessage(ctx, userID, chatID, text, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	t.Run("DB error on chat check", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		text := "Hello"

		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(
			false,
			errors.New("db error"),
		)

		result, err := svc.CreateMessage(ctx, userID, chatID, text, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("DB error on create message", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		text := "Hello"

		chatStore.EXPECT().IsChatExist(ctx, chatID).Return(true, nil)
		chatStore.EXPECT().IsMemberOfChat(ctx, userID, chatID).Return(true, nil)

		messageStore.EXPECT().CreateMessage(ctx, gomock.Any()).Return(
			int32(0),
			errors.New("db error"),
		)

		result, err := svc.CreateMessage(ctx, userID, chatID, text, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Invalid - sticker with text", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		stickerID := int32(5)

		result, err := svc.CreateMessage(ctx, userID, chatID, "text", nil, &stickerID)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Invalid - sticker with attachments", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		stickerID := int32(5)
		files := []*domain.File{{Filename: "test.jpg", Data: []byte("data")}}

		result, err := svc.CreateMessage(ctx, userID, chatID, "", files, &stickerID)
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})
}

func TestChatService_GetUserChats(t *testing.T) {
	svc, chatStore, _, _, profileService, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with personal chat", func(t *testing.T) {
		userID := int32(1)
		limit := int32(20)
		offset := int32(0)

		fullChats := []domain.FullChat{
			{
				ID:         int32(10),
				IsGroup:    false,
				UserIDWith: intPtr(int32(2)),
				LastMessage: domain.Message{
					ID:       int32(100),
					AuthorID: int32(2),
					ChatID:   int32(10),
					Text:     "Hello",
				},
			},
		}

		chatStore.EXPECT().GetUserFullChats(ctx, userID, limit, offset).Return(
			fullChats,
			[]int32{int32(2)}, // userIDs для запроса профилей
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(2)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(2): {
						UserID:     int32(2),
						FullName:   "John Doe",
						AvatarPath: stringPtr("/avatar2.jpg"),
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.GetUserChats(ctx, userID, domain.PaginateQueryParams{
			Limit: limit,
			Page:  1,
		})

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "John Doe", *result[0].Name)
		assert.Equal(t, "/avatar2.jpg", *result[0].AvatarPath)
	})

	t.Run("Success with group chat", func(t *testing.T) {
		userID := int32(1)

		fullChats := []domain.FullChat{
			{
				ID:      int32(20),
				IsGroup: true,
				Name:    stringPtr("Group Chat"),
				LastMessage: domain.Message{
					ID:       int32(101),
					AuthorID: int32(3),
					ChatID:   int32(20),
					Text:     "Group message",
				},
			},
		}

		chatStore.EXPECT().GetUserFullChats(ctx, userID, gomock.Any(), gomock.Any()).Return(
			fullChats,
			[]int32{int32(3)}, // только автор последнего сообщения
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(3): {
						UserID:   int32(3),
						FullName: "Jane Doe",
					},
				},
			},
			nil,
		)

		result, err := svc.GetUserChats(ctx, userID, domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.True(t, result[0].IsGroup)
		assert.Equal(t, "Group Chat", *result[0].Name)
	})

	t.Run("DB error", func(t *testing.T) {
		userID := int32(1)

		chatStore.EXPECT().GetUserFullChats(ctx, userID, gomock.Any(), gomock.Any()).Return(
			nil,
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetUserChats(ctx, userID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Profile service error", func(t *testing.T) {
		userID := int32(1)

		fullChats := []domain.FullChat{
			{
				ID:         int32(10),
				IsGroup:    false,
				UserIDWith: intPtr(int32(2)),
				LastMessage: domain.Message{
					AuthorID: int32(2),
				},
			},
		}

		chatStore.EXPECT().GetUserFullChats(ctx, userID, gomock.Any(), gomock.Any()).Return(
			fullChats,
			[]int32{int32(2)},
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.GetUserChats(ctx, userID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestChatService_UpdateLastReadMessage(t *testing.T) {
	svc, chatStore, _, _, _, ctrl := newChatServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		lastReadMessageID := int32(100)

		chatStore.EXPECT().UpdateLastReadMessageByUserIDAndChatID(
			ctx, userID, chatID, lastReadMessageID,
		).Return(nil)

		err := svc.UpdateLastReadMessage(ctx, userID, chatID, lastReadMessageID)
		assert.NoError(t, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		lastReadMessageID := int32(100)

		chatStore.EXPECT().UpdateLastReadMessageByUserIDAndChatID(
			ctx, userID, chatID, lastReadMessageID,
		).Return(errors.New("db error"))

		err := svc.UpdateLastReadMessage(ctx, userID, chatID, lastReadMessageID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestValidateMessageContent(t *testing.T) {
	t.Run("Valid text only", func(t *testing.T) {
		err := validateMessageContent("Hello", 0, nil)
		assert.NoError(t, err)
	})

	t.Run("Valid attachments only", func(t *testing.T) {
		err := validateMessageContent("", 2, nil)
		assert.NoError(t, err)
	})

	t.Run("Valid text with attachments", func(t *testing.T) {
		err := validateMessageContent("Check this", 1, nil)
		assert.NoError(t, err)
	})

	t.Run("Valid sticker only", func(t *testing.T) {
		stickerID := int32(5)
		err := validateMessageContent("", 0, &stickerID)
		assert.NoError(t, err)
	})

	t.Run("Invalid empty message", func(t *testing.T) {
		err := validateMessageContent("", 0, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Invalid sticker with text", func(t *testing.T) {
		stickerID := int32(5)
		err := validateMessageContent("text", 0, &stickerID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Invalid sticker with attachments", func(t *testing.T) {
		stickerID := int32(5)
		err := validateMessageContent("", 2, &stickerID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Valid whitespace text", func(t *testing.T) {
		err := validateMessageContent("   Hello   ", 0, nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid whitespace only", func(t *testing.T) {
		err := validateMessageContent("   ", 0, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})
}

func intPtr(i int32) *int32 {
	return &i
}
