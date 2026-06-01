package service

import (
	"crypto/rand"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sovatharaprom/craftform-backend/internal/config"
)

const maxUploadSize = 5 * 1024 * 1024 // 5 MB

var allowedMIME = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true,
	"image/webp": true, "image/svg+xml": true,
	"application/pdf":  true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"text/plain": true,
}

type UploadService struct {
	dir string
}

func NewUploadService(cfg *config.Config) *UploadService {
	return &UploadService{dir: cfg.UploadDir}
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *UploadService) Store(fh *multipart.FileHeader) (string, error) {
	if fh.Size > maxUploadSize {
		return "", fmt.Errorf("file exceeds 5 MB limit")
	}

	src, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	// Sniff content type from first 512 bytes
	sniff := make([]byte, 512)
	n, _ := src.Read(sniff)
	ct := http.DetectContentType(sniff[:n])

	if !allowedMIME[ct] {
		return "", fmt.Errorf("file type not allowed: %s", ct)
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	filename := randomHex(16) + ext
	dstPath := filepath.Join(s.dir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath)
		return "", fmt.Errorf("write file: %w", err)
	}

	return "/uploads/" + filename, nil
}
