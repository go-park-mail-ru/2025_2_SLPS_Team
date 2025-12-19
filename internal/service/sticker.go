package service

import (
	"context"
	"project/domain"

	"go.uber.org/zap"
)

type StickerService struct {
	stickerStore domain.StickerStore
}

func NewStickerService(stickerStore domain.StickerStore) domain.StickerService {
	return &StickerService{
		stickerStore: stickerStore,
	}
}

func (s *StickerService) GetStickerPacks(ctx context.Context) ([]domain.StickerPack, error) {
	packs, err := s.stickerStore.GetStickerPacks(ctx)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get sticker packs", zap.Error(err))
		return nil, domain.ErrDB
	}

	domain.FromContext(ctx).Info("Sticker packs retrieved successfully", zap.Int("count", len(packs)))
	return packs, nil
}

func (s *StickerService) GetStickersByPackID(ctx context.Context, packID int32) ([]domain.Sticker, error) {
	stickers, err := s.stickerStore.GetStickersByPackID(ctx, packID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get stickers by pack ID", zap.Error(err), zap.Int32("packID", packID))
		return nil, domain.ErrDB
	}

	domain.FromContext(ctx).Info("Stickers retrieved successfully", zap.Int32("packID", packID), zap.Int("count", len(stickers)))
	return stickers, nil
}

func (s *StickerService) GetStickerByID(ctx context.Context, stickerID int32) (*domain.Sticker, error) {
	sticker, err := s.stickerStore.GetStickerByID(ctx, stickerID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get sticker by ID", zap.Error(err), zap.Int32("stickerID", stickerID))
		return nil, err
	}

	domain.FromContext(ctx).Info("Sticker retrieved successfully", zap.Int32("stickerID", stickerID))
	return sticker, nil
}
