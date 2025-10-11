package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const uploadDir = "./uploads"

func UploadFile(header *multipart.FileHeader) (string, error) {

	file, err := header.Open()
	if err != nil {
		return "", err
	}

	defer file.Close()
	ext := filepath.Ext(header.Filename)
	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	filePath := filepath.Join(uploadDir, fileName)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

func UploadFiles(files []*multipart.FileHeader) ([]string, error) {
	var fileNames []string

	for _, header := range files {
		fileName, err := UploadFile(header)
		if err != nil {
			return nil, err
		}
		fileNames = append(fileNames, fileName)

	}

	return fileNames, nil
}

func DeleteFile(fileName string) error {
	fullPath := filepath.Join(uploadDir, fileName)
	if err := os.Remove(fullPath); err != nil {
		return err
	}
	return nil
}

func DeleteFiles(fileNames []string) error {
	for _, fileName := range fileNames {
		if err := DeleteFile(fileName); err != nil {
			return err
		}
	}
	return nil
}
