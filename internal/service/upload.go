package service

import (
	"context"
	"errors"
	"fmt"
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

// UploadFile загружает один файл из []byte
func UploadFile(file *domain.File) (string, error) {
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	filePath := filepath.Join(uploadDir, fileName)
	if err := os.WriteFile(filePath, file.Data, 0644); err != nil {
		return "", err
	}

	return fileName, nil
}

// UploadFiles загружает несколько файлов
func UploadFiles(files []*domain.File) ([]string, error) {
	var fileNames []string

	for _, file := range files {
		fileName, err := UploadFile(file)
		if err != nil {
			// В случае ошибки удаляем уже загруженные файлы
			for _, f := range fileNames {
				_ = DeleteFile(f)
			}
			return nil, err
		}
		fileNames = append(fileNames, fileName)
	}

	return fileNames, nil
}

// DeleteFile удаляет один файл
func DeleteFile(fileName string) error {
	fullPath := filepath.Join(uploadDir, fileName)
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// DeleteFiles удаляет несколько файлов
func DeleteFiles(fileNames []*string) error {
	for _, fileName := range fileNames {
		if fileName != nil && *fileName != "" {
			if err := DeleteFile(*fileName); err != nil {
				fmt.Printf("failed to delete file %s: %v\n", *fileName, err)
				return err
			}
		}
	}
	return nil
}

// HandleFileUpload сначала загружает новые файлы, затем удаляет старые
func HandleFileUpload(files []*domain.File, oldPaths []*string) ([]string, error) {
	newPaths, err := UploadFiles(files)
	if err != nil {
		return nil, err
	}

	// Только после успешной загрузки удаляем старые файлы
	DeleteFiles(oldPaths)

	return newPaths, nil
}

// Uploads проверяет путь и возвращает безопасный полный путь к файлу
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
		domain.FromContext(ctx).Warn("Attempt to access forbidden file", zap.String("cleanPath", cleanPath))
		return nil, domain.ErrAccessDenied
	}

	fullPath := filepath.Join(absStaticDir, filepath.FromSlash(cleanPath))

	// Проверка на выход за пределы директории
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil || !strings.HasPrefix(absFullPath, absStaticDir) {
		domain.FromContext(ctx).Warn("Attempt to access forbidden file", zap.String("cleanPath", cleanPath))
		return nil, domain.ErrAccessDenied
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			domain.FromContext(ctx).Info("File not found", zap.String("cleanPath", cleanPath))
			return nil, domain.ErrNotExist
		}
		domain.FromContext(ctx).Error("Failed to stat file", zap.String("cleanPath", cleanPath), zap.Error(err))
		return nil, domain.ErrService
	}
	if info.IsDir() {
		domain.FromContext(ctx).Info("File is a directory", zap.String("cleanPath", cleanPath))
		return nil, domain.ErrNotExist
	}

	domain.FromContext(ctx).Info("Returning full path")
	return &fullPath, nil
}
