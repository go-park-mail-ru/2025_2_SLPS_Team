package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"project/domain"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const uploadDir = "./uploads"

func UploadFile(header *domain.File) (string, error) {

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

func UploadFiles(files []*domain.File) ([]string, error) {
	var fileNames []string

	for _, header := range files {
		fileName, err := UploadFile(header)
		if err != nil {
			for _, uploadedFile := range fileNames {
				DeleteFile(uploadedFile)
			}
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

func DeleteFiles(fileNames []*string) error {
	for _, fileName := range fileNames {
		if fileName != nil && *fileName != "" {
			if err := DeleteFile(*fileName); err != nil {
				continue
			}
		}
	}
	return nil
}

func HandleFileUpload(
	files []*domain.File,
	oldPaths []*string,
) ([]string, error) {
	var newPaths []string

	err := DeleteFiles(oldPaths)
	if err != nil {
		return nil, err
	}

	newPaths, err = UploadFiles(files)
	if err != nil {
		return nil, err
	}

	return newPaths, nil
}

func Uploads(ctx context.Context, absStaticDir string, prefix string, URLPath string) (*string, error) {

	if !strings.HasPrefix(URLPath, prefix) {
		domain.FromContext(ctx).Info("Prefix mismatch", zap.String("url", URLPath))
		return nil, domain.ErrNotExist
	}

	relPath, err := url.PathUnescape(strings.TrimPrefix(URLPath, prefix))
	if err != nil {
		domain.FromContext(ctx).Error("Invalid URL encoding", zap.String("url", URLPath), zap.Error(err))
		return nil, domain.ErrService
	}

	cleanPath := path.Clean("/" + relPath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	if strings.Contains(cleanPath, "..") {
		domain.FromContext(ctx).Warn("Try get access to forbidden file", zap.String("cleanPath", cleanPath))
		return nil, domain.ErrAccessDenied
	}

	fullPath := filepath.Join(absStaticDir, filepath.FromSlash(cleanPath))

	if !strings.HasPrefix(fullPath, absStaticDir) {
		domain.FromContext(ctx).Warn("Try get access to forbidden file", zap.String("cleanPath", cleanPath))
		return nil, domain.ErrAccessDenied
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			domain.FromContext(ctx).Info("File not found", zap.String("cleanPath", cleanPath))
			return nil, domain.ErrNotExist
		}
		domain.FromContext(ctx).Error("Failed to stat filed", zap.String("cleanPath", cleanPath), zap.Error(err))
		return nil, domain.ErrService
	}
	if info.IsDir() {
		domain.FromContext(ctx).Info("File is a directory", zap.String("cleanPath", cleanPath))
		return nil, domain.ErrNotExist
	}

	domain.FromContext(ctx).Info("Return correct full path")
	return &fullPath, nil
}
