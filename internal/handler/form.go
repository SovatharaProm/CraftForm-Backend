package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sovatharaprom/craftform-backend/internal/middleware"
	"github.com/sovatharaprom/craftform-backend/internal/model"
	"github.com/sovatharaprom/craftform-backend/internal/service"
)

type FormHandler struct {
	svc *service.FormService
}

func NewFormHandler(svc *service.FormService) *FormHandler {
	return &FormHandler{svc: svc}
}

func (h *FormHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	forms, err := h.svc.List(c.Request().Context(), userID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, forms)
}

func (h *FormHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req model.FormRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errResp("invalid request body"))
	}
	form, err := h.svc.Create(c.Request().Context(), userID, req)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusCreated, form)
}

func (h *FormHandler) GetByID(c echo.Context) error {
	requesterID := middleware.GetUserID(c) // may be empty for public forms
	form, err := h.svc.GetByID(c.Request().Context(), c.Param("id"), requesterID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, form)
}

func (h *FormHandler) Update(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req model.FormRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errResp("invalid request body"))
	}
	form, err := h.svc.Update(c.Request().Context(), c.Param("id"), userID, req)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, form)
}

func (h *FormHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.svc.Delete(c.Request().Context(), c.Param("id"), userID); err != nil {
		return respondErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FormHandler) Duplicate(c echo.Context) error {
	userID := middleware.GetUserID(c)
	form, err := h.svc.Duplicate(c.Request().Context(), c.Param("id"), userID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusCreated, form)
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

func respondErr(c echo.Context, err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return c.JSON(http.StatusNotFound, errResp("not found"))
	case errors.Is(err, model.ErrForbidden):
		return c.JSON(http.StatusForbidden, errResp("forbidden"))
	case errors.Is(err, model.ErrConflict):
		return c.JSON(http.StatusConflict, errResp("conflict"))
	default:
		return c.JSON(http.StatusInternalServerError, errResp("internal error"))
	}
}

func errResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}
