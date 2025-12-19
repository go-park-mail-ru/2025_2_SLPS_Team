package service

import (
	"context"
	"project/domain"
	"project/shared/pb"

	"go.uber.org/zap"
)

type ApplicationService struct {
	authService      pb.AuthServiceClient
	wsHub            domain.WSHub
	applicationStore domain.ApplicationStore
}

func NewApplicationService(authService pb.AuthServiceClient, applicationStore domain.ApplicationStore, wsHub domain.WSHub) domain.ApplicationService {
	return &ApplicationService{
		authService:      authService,
		wsHub:            wsHub,
		applicationStore: applicationStore,
	}
}
func (s *ApplicationService) GetApplications(ctx context.Context, params domain.PaginateQueryParams) ([]domain.Application, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	userID, _ := ctx.Value(domain.UserIDKey).(int32)

	resp, err := s.authService.GetUserRole(ctx, &pb.UserIDRequest{UserId: userID})
	role := resp.Role
	if err != nil {
		domain.FromContext(ctx).Error("can`t find user", zap.Error(err))
		return nil, err
	}
	if role == "admin" {
		return s.applicationStore.GetApplications(ctx, limit, offset)
	}
	return s.applicationStore.GetApplicationsByUser(ctx, limit, offset)
}

func (s *ApplicationService) UpdateApplicationText(ctx context.Context, id int32, newText string) error {
	return s.applicationStore.UpdateApplicationText(ctx, id, newText)
}

func (s *ApplicationService) UpdateApplicationStatus(ctx context.Context, id int32, newStatus string) error {
	userID, _ := ctx.Value(domain.UserIDKey).(int32)

	resp, err := s.authService.GetUserRole(ctx, &pb.UserIDRequest{UserId: userID})
	role := resp.Role
	if err != nil {
		domain.FromContext(ctx).Error("can`t find user", zap.Error(err))
		return nil
	}

	if role != "admin" && newStatus != "closed" {
		return domain.ErrAccessDenied
	}
	return s.applicationStore.UpdateApplicationStatus(ctx, id, newStatus)
}

func (s *ApplicationService) CreateApplication(ctx context.Context, application domain.Application) (int32, error) {
	return s.applicationStore.CreateApplication(ctx, application)
}
func (s *ApplicationService) MergeTempSession(ctx context.Context) error {
	return s.applicationStore.MergeTempSession(ctx)
}
