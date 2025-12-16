package handler

import (
	"net/http"
	"project/domain"
)

type StickerHandler struct {
	stickerService domain.StickerService
}

func NewStickerHandler(stickerService domain.StickerService) *StickerHandler {
	return &StickerHandler{
		stickerService: stickerService,
	}
}

// GetStickerPacks возвращает список всех стикерпаков
// @Summary Получить список стикерпаков
// @Description Возвращает список всех доступных стикерпаков
// @Tags stickers
// @Produce json
// @Success 200 {array} domain.StickerPack
// @Failure 500 {object} JSONResponse
// @Router /sticker-packs [get]
func (h *StickerHandler) GetStickerPacks(w http.ResponseWriter, r *http.Request) {
	packs, err := h.stickerService.GetStickerPacks(r.Context())
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, packs)
}

// GetStickersByPackID возвращает стикеры из указанного пака
// @Summary Получить стикеры из пака
// @Description Возвращает все стикеры из указанного стикерпака
// @Tags stickers
// @Produce json
// @Param packID path int32 true "ID стикерпака"
// @Success 200 {array} domain.Sticker
// @Failure 400 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /sticker-packs/{packID}/stickers [get]
func (h *StickerHandler) GetStickersByPackID(w http.ResponseWriter, r *http.Request) {
	packID, err := PathInt32(r, "packID")

	stickers, err := h.stickerService.GetStickersByPackID(r.Context(), int32(packID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, stickers)
}
