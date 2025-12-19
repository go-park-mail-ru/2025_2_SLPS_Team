package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Status string
type Category string

//easyjson:json
type Application struct {
	ID            int32     `json:"id"`
	Status        Status    `json:"status" example:"open"`
	Category      Category  `json:"category" example:"app_freezing" `
	Text          string    `json:"text"`
	AuthorID      *string   `json:"authorID"`
	CreatedAt     time.Time `dbconn:"createdAt" json:"createdAt" example:"1990-01-01T00:00:00Z"`
	UpdatedAt     time.Time `dbconn:"updated_at" json:"updatedAt" example:"1990-01-01T00:00:00Z"`
	FullName      string    `json:"fullName"`
	EmailReg      string    `json:"emailReg"`
	EmailFeedBack string    `json:"emailFeedBack"`
}

//easyjson:json
type TempSessionInfo struct {
	UserID        *int32
	TempSessionID *uuid.UUID
}

//easyjson:json
type Applications []Application

//easyjson:json
type ApplicationIDResponse struct {
	ApplicationID int32 `json:"applicationID"`
}
type ApplicationService interface {
	GetApplications(ctx context.Context, params PaginateQueryParams) ([]Application, error)
	UpdateApplicationText(ctx context.Context, id int32, newText string) error
	UpdateApplicationStatus(ctx context.Context, id int32, newStatus string) error
	CreateApplication(ctx context.Context, application Application) (int32, error)
	MergeTempSession(ctx context.Context) error
}

type ApplicationStore interface {
	GetApplicationsByUser(ctx context.Context, limit, offset int32) ([]Application, error)
	GetApplications(ctx context.Context, limit, offset int32) ([]Application, error)
	UpdateApplicationText(ctx context.Context, id int32, newText string) error
	UpdateApplicationStatus(ctx context.Context, id int32, newStatus string) error
	CreateApplication(ctx context.Context, app Application) (int32, error)
	MergeTempSession(ctx context.Context) error
}
