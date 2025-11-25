package domain

import (
	"context"
	"mime/multipart"
	"time"
)

type Community struct {
	ID               int32     `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	CreatorID        int32     `json:"creatorID"`
	AvatarPath       *string   `json:"avatarPath"`
	CoverPath        *string   `json:"coverPath"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	SubscribersCount int32     `json:"subscribersCount"`
}

// Надо для вкладки Подписки/Рекомендации
type ShortCommunity struct {
	ID               int32   `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	AvatarPath       *string `json:"avatarPath"`
	SubscribersCount int32   `json:"subscribersCount"`
}

// Надо когда юзер заходит на сообщество, но тут не хватает состояния подписан ли ты или нет
type ShortCommunityWithCoverPathAndCreatedAt struct {
	ID               int32     `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	AvatarPath       *string   `json:"avatarPath"`
	CoverPath        *string   `json:"coverPath"`
	CreatedAt        time.Time `json:"createdAt"`
	SubscribersCount int32     `json:"subscribersCount"`
}

type CommunityForMyCommunity struct {
	ID         int32   `json:"id"`
	Name       string  `json:"name"`
	AvatarPath *string `json:"avatarPath"`
}

// Надо когда юзер заходит на сообщество
type CommunityForView struct {
	ID               int32     `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	CreatorID        int32     `json:"creatorID"`
	AvatarPath       *string   `json:"avatarPath"`
	CoverPath        *string   `json:"coverPath"`
	CreatedAt        time.Time `json:"createdAt"`
	SubscribersCount int32     `json:"subscribersCount"`
	IsSubscribed     bool      `json:"isSubscribed"`
}

// Структура для подписчика сообщества
type CommunitySubscriber struct {
	UserID     int32   `json:"userID"`
	FullName   string  `json:"fullName"`
	AvatarPath *string `json:"avatarPath"`
}
type CommunityType string

const (
	Subscriber  CommunityType = "subscriber"
	Recommended CommunityType = "recommended"
	Owned       CommunityType = "owned"
)

type CommunityRequest struct {
	Name        string `json:"name" valid:"required,length(3|48)"`
	Description string `json:"description" valid:"optional,length(0|512)"`
}

type CommunityService interface {
	CreateCommunity(ctx context.Context, userID int32, req CommunityRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) (*Community, error)
	UpdateCommunity(ctx context.Context, communityID int32, userID int32, req CommunityRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) error
	DeleteCommunity(ctx context.Context, communityID int32, userID int32) error
	GetCommunity(ctx context.Context, userID int32, communityID int32) (*CommunityForView, error)
	GetUserCommunities(ctx context.Context, userID int32, params PaginateQueryParams) ([]ShortCommunity, error)
	GetOtherCommunities(ctx context.Context, userID int32, params PaginateQueryParams) ([]ShortCommunity, error)
	GetUserCommunitiesByID(ctx context.Context, targetUserID int32, params PaginateQueryParams) ([]ShortCommunity, error)
	GetUserSubscribedCommunityIDs(ctx context.Context, targetUserID int32) ([]int32, error)
	GetCreatedCommunities(ctx context.Context, userID int32, params PaginateQueryParams) ([]CommunityForMyCommunity, error)
	GetCommunitySubscribers(ctx context.Context, communityID int32, params PaginateQueryParams) ([]CommunitySubscriber, error)
	Subscribe(ctx context.Context, communityID int32, userID int32) error
	Unsubscribe(ctx context.Context, communityID int32, userID int32) error
	SearchShortCommunityByNameAndType(ctx context.Context, userID int32, params PaginateQueryParams, name string, cType CommunityType) ([]ShortCommunity, error)
}

type ElasticCommunityStore interface {
	CreateCommunity(ctx context.Context, name string, communityID int32) error
	UpdateCommunity(ctx context.Context, name string, communityID int32) error
	DeleteCommunity(ctx context.Context, communityID int32) error
	SearchCommunityIDsByName(ctx context.Context, name string, filterIDs []int32, isTerms bool, limit, offset int32) ([]int32, error)
}

type CommunityStore interface {
	CreateCommunity(ctx context.Context, community *Community) error
	UpdateCommunity(ctx context.Context, community *Community) error
	DeleteCommunity(ctx context.Context, id int32, creatorID int32) error
	GetCommunityByID(ctx context.Context, id int32) (*Community, error)
	GetUserCommunities(ctx context.Context, userID int32, limit, offset int32) ([]ShortCommunity, error)
	GetOtherCommunities(ctx context.Context, userID int32, limit, offset int32) ([]ShortCommunity, error)
	GetUserCommunitiesByID(ctx context.Context, targetUserID int32, limit, offset int32) ([]ShortCommunity, error)
	GetUserSubscribedCommunityIDs(ctx context.Context, targetUserID int32) ([]int32, error)
	GetCreatedCommunities(ctx context.Context, userID int32, limit, offset int32) ([]CommunityForMyCommunity, error)
	GetCommunitySubscribers(ctx context.Context, communityID int32, limit, offset int32) ([]int32, error)
	Subscribe(ctx context.Context, communityID int32, userID int32) error
	Unsubscribe(ctx context.Context, communityID int32, userID int32) error
	IsSubscribed(ctx context.Context, communityID int32, userID int32) (bool, error)
	GetCommunitiesByIDs(ctx context.Context, communityIDs []int32) ([]ShortCommunity, error)
}
