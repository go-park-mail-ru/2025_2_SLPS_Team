package domain

import (
	"context"
	"mime/multipart"
	"time"
)

type Community struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	CreatorID        int       `json:"creatorID"`
	AvatarPath       *string   `json:"avatarPath"`
	CoverPath        *string   `json:"coverPath"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	SubscribersCount int       `json:"subscribersCount"`
}

// Надо для вкладки Подписки/Рекомендации
type ShortCommunity struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	AvatarPath       *string `json:"avatarPath"`
	SubscribersCount int     `json:"subscribersCount"`
}

// Надо когда юзер заходит на сообщество, но тут не хватает состояния подписан ли ты или нет
type ShortCommunityWithCoverPathAndCreatedAt struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	AvatarPath       *string   `json:"avatarPath"`
	CoverPath        *string   `json:"coverPath"`
	CreatedAt        time.Time `json:"createdAt"`
	SubscribersCount int       `json:"subscribersCount"`
}

//
type CommunityForView struct {
    ID               int       `json:"id"`
    Name             string    `json:"name"`
    Description      string    `json:"description"`
    AvatarPath       *string   `json:"avatarPath"`
    CoverPath        *string   `json:"coverPath"`
    CreatedAt        time.Time `json:"createdAt"`
    SubscribersCount int       `json:"subscribersCount"`
}

// Надо когда юзер заходит на сообщество
type CommunityForViewWithSubscription struct {
    CommunityForView
    IsSubscribed bool `json:"isSubscribed"`
}

type CommunityRequest struct {
	Name        string `json:"name" valid:"required,length(3|48)"`
	Description string `json:"description" valid:"optional,length(0|512)"`
}

type CommunityService interface {
	CreateCommunity(ctx context.Context, userID int, req CommunityRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) (*Community, error)
	UpdateCommunity(ctx context.Context, communityID int, userID int, req CommunityRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) error
	DeleteCommunity(ctx context.Context, communityID int, userID int) error
	GetCommunity(ctx context.Context, userID int, communityID int) (*CommunityForViewWithSubscription, error)
	GetUserCommunities(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortCommunity, error)
	GetOtherCommunities(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortCommunity, error)
	Subscribe(ctx context.Context, communityID int, userID int) error
	Unsubscribe(ctx context.Context, communityID int, userID int) error
	GetCommunityPosts(ctx context.Context, userID int, communityID int, params PaginateQueryParams) ([]Post, error)
}

type CommunityStore interface {
	CreateCommunity(ctx context.Context, community *Community) error
	UpdateCommunity(ctx context.Context, community *Community) error
	DeleteCommunity(ctx context.Context, id int, creatorID int) error
	GetCommunityByID(ctx context.Context, id int) (*Community, error)
	GetUserCommunities(ctx context.Context, userID int, limit, offset int) ([]ShortCommunity, error)
	GetOtherCommunities(ctx context.Context, userID int, limit, offset int) ([]ShortCommunity, error)
	Subscribe(ctx context.Context, communityID int, userID int) error
	Unsubscribe(ctx context.Context, communityID int, userID int) error
	IsSubscribed(ctx context.Context, communityID int, userID int) (bool, error)
}
