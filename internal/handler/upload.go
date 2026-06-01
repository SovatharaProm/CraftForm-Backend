package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sovatharaprom/craftform-backend/internal/service"
)

type UploadHandler struct {
	svc *service.UploadService
}

func NewUploadHandler(svc *service.UploadService) *UploadHandler {
	return &UploadHandler{svc: svc}
}

// Upload — POST /api/upload (requires auth)
func (h *UploadHandler) Upload(c echo.Context) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errResp("no file provided"))
	}

	url, err := h.svc.Store(fh)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errResp(err.Error()))
	}

	return c.JSON(http.StatusOK, map[string]string{"fileUrl": url})
}
