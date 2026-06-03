package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sovatharaprom/craftform-backend/internal/middleware"
	"github.com/sovatharaprom/craftform-backend/internal/model"
	"github.com/sovatharaprom/craftform-backend/internal/service"
)

type ResponseHandler struct {
	svc *service.ResponseService
}

func NewResponseHandler(svc *service.ResponseService) *ResponseHandler {
	return &ResponseHandler{svc: svc}
}

// Submit — POST /api/forms/:id/responses (optional auth)
func (h *ResponseHandler) Submit(c echo.Context) error {
	var req model.SubmitResponseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errResp("invalid request body"))
	}

	userID := optionalUserID(c)
	resp, err := h.svc.Submit(c.Request().Context(), c.Param("id"), userID, req)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// List — GET /api/forms/:id/responses (owner only)
func (h *ResponseHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	filter := model.ResponseFilter{
		Query:    c.QueryParam("q"),
		Email:    c.QueryParam("email"),
		DateFrom: parseTime(c.QueryParam("from")),
		DateTo:   parseTime(c.QueryParam("to")),
	}
	responses, err := h.svc.List(c.Request().Context(), c.Param("id"), userID, filter)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, responses)
}

// GetByID — GET /api/forms/:id/responses/:rid (owner only)
func (h *ResponseHandler) GetByID(c echo.Context) error {
	userID := middleware.GetUserID(c)
	resp, err := h.svc.GetByID(c.Request().Context(), c.Param("rid"), c.Param("id"), userID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Delete — DELETE /api/forms/:id/responses/:rid (owner only)
func (h *ResponseHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.svc.Delete(c.Request().Context(), c.Param("rid"), c.Param("id"), userID); err != nil {
		return respondErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Analytics — GET /api/forms/:id/responses/summary (owner only)
func (h *ResponseHandler) Analytics(c echo.Context) error {
	userID := middleware.GetUserID(c)
	analytics, err := h.svc.GetAnalytics(c.Request().Context(), c.Param("id"), userID)
	if err != nil {
		return respondErr(c, err)
	}
	return c.JSON(http.StatusOK, analytics)
}

func optionalUserID(c echo.Context) *string {
	id := middleware.GetUserID(c)
	if id == "" {
		return nil
	}
	return &id
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
