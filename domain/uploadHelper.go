package domain

import (
	"io"
	"mime/multipart"
	"net/http"

	"go.uber.org/zap"
)

type File struct {
	Filename    string
	ContentType string
	Data        []byte
}

func MultipartToFile(h *multipart.FileHeader) (*File, error) {
	f, err := h.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return &File{
		Filename:    h.Filename,
		ContentType: h.Header.Get("Content-Type"),
		Data:        data,
	}, nil
}
func MultipartListToFiles(headers []*multipart.FileHeader) ([]*File, error) {
	files := make([]*File, 0, len(headers))

	for _, h := range headers {
		f, err := MultipartToFile(h)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	return files, nil
}

func MultipartFiles(r *http.Request, field string) ([]*File, error) {

	form := r.MultipartForm
	if form == nil || form.File == nil {
		return nil, nil
	}

	files, ok := form.File[field]
	if !ok || len(files) == 0 {
		return nil, nil
	}

	result, err := MultipartListToFiles(files)
	if err != nil {
		FromContext(r.Context()).Error("Failed to parse multipart form to files", zap.Error(err))
		return nil, ErrInvalidParams
	}

	return result, nil
}
