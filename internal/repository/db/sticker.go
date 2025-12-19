package db

import (
	"context"
	"database/sql"
	"fmt"
	"project/domain"
	"time"

	"go.uber.org/zap"
)

type DBStickerStore struct {
	db *sql.DB
}

func NewDBStickerStore(db *sql.DB) domain.StickerStore {
	return &DBStickerStore{db: db}
}

func (store *DBStickerStore) GetStickerPacks(ctx context.Context) ([]domain.StickerPack, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "stickerStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetStickerPacks")
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `SELECT id, name, cover_path FROM sticker_packs ORDER BY created_at DESC`

	rows, err := store.db.QueryContext(ctx, query)
	dblogger = dblogger.With(zap.String("query", query))
	if err != nil {
		dblogger.Error("Failed to get sticker packs", zap.Error(err))
		return nil, fmt.Errorf("failed to get sticker packs: %w", err)
	}
	defer rows.Close()

	packs := []domain.StickerPack{}
	for rows.Next() {
		var pack domain.StickerPack
		if err := rows.Scan(&pack.ID, &pack.Name, &pack.CoverPath); err != nil {
			dblogger.Error("Failed to scan sticker pack", zap.Error(err))
			return nil, fmt.Errorf("failed to scan sticker pack: %w", err)
		}
		packs = append(packs, pack)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Rows iteration error", zap.Error(err))
		return nil, err
	}

	dblogger.Info("Sticker packs retrieved successfully", zap.Int("count", len(packs)))
	return packs, nil
}

func (store *DBStickerStore) GetStickersByPackID(ctx context.Context, packID int32) ([]domain.Sticker, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "stickerStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetStickersByPackID", zap.Int32("packID", packID))
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `SELECT id, pack_id, file_path, position FROM stickers WHERE pack_id = $1 ORDER BY position`

	rows, err := store.db.QueryContext(ctx, query, packID)
	dblogger = dblogger.With(zap.String("query", query))
	if err != nil {
		dblogger.Error("Failed to get stickers by pack ID", zap.Error(err))
		return nil, fmt.Errorf("failed to get stickers by pack ID: %w", err)
	}
	defer rows.Close()

	stickers := []domain.Sticker{}
	for rows.Next() {
		var sticker domain.Sticker
		if err := rows.Scan(&sticker.ID, &sticker.PackID, &sticker.FilePath, &sticker.Position); err != nil {
			dblogger.Error("Failed to scan sticker", zap.Error(err))
			return nil, fmt.Errorf("failed to scan sticker: %w", err)
		}
		stickers = append(stickers, sticker)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Rows iteration error", zap.Error(err))
		return nil, err
	}

	dblogger.Info("Stickers retrieved successfully", zap.Int("count", len(stickers)))
	return stickers, nil
}

func (store *DBStickerStore) GetStickerByID(ctx context.Context, stickerID int32) (*domain.Sticker, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "stickerStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetStickerByID", zap.Int32("stickerID", stickerID))
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `SELECT id, pack_id, file_path, position FROM stickers WHERE id = $1`

	var sticker domain.Sticker
	err := store.db.QueryRowContext(ctx, query, stickerID).Scan(
		&sticker.ID, &sticker.PackID, &sticker.FilePath, &sticker.Position,
	)
	dblogger = dblogger.With(zap.String("query", query))

	if err != nil {
		if err == sql.ErrNoRows {
			dblogger.Info("Sticker not found")
			return nil, domain.ErrNotFound
		}
		dblogger.Error("Failed to get sticker by ID", zap.Error(err))
		return nil, fmt.Errorf("failed to get sticker by ID: %w", err)
	}

	dblogger.Info("Sticker retrieved successfully")
	return &sticker, nil
}
