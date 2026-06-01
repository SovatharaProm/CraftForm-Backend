package handler

import (
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

// List — GET /api/forms?q=&status=&sort= (owner)
func (h *FormHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	filter := model.FormFilter{
		Query:  c.QueryParam("q"),
		Status: c.QueryParam("status"),
		Sort:   c.QueryParam("sort"),
	}
	forms, err := h.svc.List(c.Request().Context(), userID, filter)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, forms)
}

// ListPublic — GET /api/forms/public?q= (no auth)
func (h *FormHandler) ListPublic(c echo.Context) error {
	forms, err := h.svc.ListPublic(c.Request().Context(), c.QueryParam("q"))
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, forms)
}

// Create — POST /api/forms (owner)
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

// GetByID — GET /api/forms/:id (optional auth)
func (h *FormHandler) GetByID(c echo.Context) error {
	requesterID := middleware.GetUserID(c)
	form, err := h.svc.GetByID(c.Request().Context(), c.Param("id"), requesterID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, form)
}

// Update — PUT /api/forms/:id (owner)
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

// Delete — DELETE /api/forms/:id (owner)
func (h *FormHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.svc.Delete(c.Request().Context(), c.Param("id"), userID); err != nil {
		return respondErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Duplicate — POST /api/forms/:id/duplicate (owner)
func (h *FormHandler) Duplicate(c echo.Context) error {
	userID := middleware.GetUserID(c)
	form, err := h.svc.Duplicate(c.Request().Context(), c.Param("id"), userID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusCreated, form)
}
