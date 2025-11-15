package service

import (
	"context"
	"go.uber.org/zap"
	"project/domain"
)

type ApplicationService struct {
	userStore        domain.UserStore
	wsHub            domain.WSHub
	applicationStore domain.ApplicationStore
}

func NewApplicationService(userStore domain.UserStore, applicationStore domain.ApplicationStore, wsHub domain.WSHub) domain.ApplicationService {
	return &ApplicationService{
		userStore:        userStore,
		wsHub:            wsHub,
		applicationStore: applicationStore,
	}
}
func (s *ApplicationService) GetApplications(ctx context.Context, params domain.PaginateQueryParams) ([]domain.Application, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	isAdmin, err := s.userStore.IsUserAdmin(ctx)
	if err != nil {
		domain.FromContext(ctx).Error("can`t find user", zap.Error(err))
		return nil, err
	}
	if isAdmin {
		return s.applicationStore.GetApplications(ctx, limit, offset)
	}
	return s.applicationStore.GetApplicationsByUser(ctx, limit, offset)
}

func (s *ApplicationService) UpdateApplicationText(ctx context.Context, id int, newText string) error {
	return s.applicationStore.UpdateApplicationText(ctx, id, newText)
}

func (s *ApplicationService) UpdateApplicationStatus(ctx context.Context, id int, newStatus string) error {
	isAdmin, err := s.userStore.IsUserAdmin(ctx)
	if err != nil {
		return err
	}
	if !isAdmin && newStatus != "closed" {
		return domain.ErrAccessDenied
	}
	return s.applicationStore.UpdateApplicationStatus(ctx, id, newStatus)
}

func (s *ApplicationService) CreateApplication(ctx context.Context, application domain.Application) (int, error) {
	return s.applicationStore.CreateApplication(ctx, application)
}
func (s *ApplicationService) MergeTempSession(ctx context.Context) error {
	return s.applicationStore.MergeTempSession(ctx)
}
