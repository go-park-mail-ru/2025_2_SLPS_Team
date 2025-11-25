package domain

import (
	"io"
	"mime/multipart"
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
