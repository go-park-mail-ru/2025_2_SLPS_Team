package domain

import "context"

// StickerPack - стикерпак (набор стикеров)
type StickerPack struct {
    ID        int32  `json:"id"`
    Name      string `json:"name"`
    CoverPath string `json:"coverPath"` // путь к обложке (первый стикер в пачке)
}

// Sticker - отдельный стикер
type Sticker struct {
    ID       int32  `json:"id"`
    PackID   int32  `json:"packID"`
    FilePath string `json:"filePath"` // путь к картинке стикера
    Position int32  `json:"position"` // порядковый номер в пачке
}

// StickerService - интерфейс сервиса стикеров
type StickerService interface {
    GetStickerPacks(ctx context.Context) ([]StickerPack, error)
    GetStickersByPackID(ctx context.Context, packID int32) ([]Sticker, error)
    GetStickerByID(ctx context.Context, stickerID int32) (*Sticker, error)
}

// StickerStore - интерфейс хранилища стикеров
type StickerStore interface {
    GetStickerPacks(ctx context.Context) ([]StickerPack, error)
    GetStickersByPackID(ctx context.Context, packID int32) ([]Sticker, error)
    GetStickerByID(ctx context.Context, stickerID int32) (*Sticker, error)
}