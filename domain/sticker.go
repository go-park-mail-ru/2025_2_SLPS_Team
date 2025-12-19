package domain

import "context"

//easyjson:json
type StickerPack struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	CoverPath string `json:"coverPath"` // путь к обложке (первый стикер в пачке)
}

//easyjson:json
type Sticker struct {
	ID       int32  `json:"id"`
	PackID   int32  `json:"packID"`
	FilePath string `json:"filePath"` // путь к картинке стикера
	Position int32  `json:"position"` // порядковый номер в пачке
}

//easyjson:json
type StickerPackList []StickerPack

//easyjson:json
type StickerList []Sticker

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
