package domain

import (
	"context"
	"mime/multipart"
	"time"
)

type Community struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatorID   int        `json:"creatorID"`
	AvatarPath  *string    `json:"avatarPath"`
	CoverPath   *string    `json:"coverPath"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type CommunityWithSubscription struct {
	Community
	SubscribersCount int  `json:"subscribersCount"`
	IsSubscribed     bool `json:"isSubscribed"`
}

type CommunityCreateRequest struct {
	Name        string `json:"name" valid:"required,length(3|48)"`
	Description string `json:"description" valid:"optional,length(0|512)"`
}

type CommunityUpdateRequest struct {
	Name        string `json:"name" valid:"optional,length(3|48)"`
	Description string `json:"description" valid:"optional,length(0|512)"`
}

type CommunityService interface {
	CreateCommunity(ctx context.Context, userID int, req CommunityCreateRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) (*Community, error)
	UpdateCommunity(ctx context.Context, communityID int, userID int, req CommunityUpdateRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) error
	DeleteCommunity(ctx context.Context, communityID int, userID int) error
	GetCommunity(ctx context.Context, userID int, communityID int) (*CommunityWithSubscription, error)
	GetUserCommunities(ctx context.Context, userID int, params PaginateQueryParams) ([]CommunityWithSubscription, error)
	GetOtherCommunities(ctx context.Context, userID int, params PaginateQueryParams) ([]CommunityWithSubscription, error)
	Subscribe(ctx context.Context, communityID int, userID int) error
	Unsubscribe(ctx context.Context, communityID int, userID int) error
	CountSubscribers(ctx context.Context, communityID int) (int, error)
	GetCommunityPosts(ctx context.Context, userID int, communityID int, params PaginateQueryParams) ([]Post, error)
}

type CommunityStore interface {
	CreateCommunity(ctx context.Context, community *Community) error
	UpdateCommunity(ctx context.Context, community *Community) error
	DeleteCommunity(ctx context.Context, id int, creatorID int) error
	GetCommunityByID(ctx context.Context, id int) (*Community, error)
	GetUserCommunities(ctx context.Context, userID int, limit, offset int) ([]Community, error)
	GetOtherCommunities(ctx context.Context, userID int, limit, offset int) ([]Community, error)
	Subscribe(ctx context.Context, communityID int, userID int) error
	Unsubscribe(ctx context.Context, communityID int, userID int) error
	IsSubscribed(ctx context.Context, communityID int, userID int) (bool, error)
	CountSubscribers(ctx context.Context, communityID int) (int, error)
}