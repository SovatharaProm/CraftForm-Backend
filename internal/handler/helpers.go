package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sovatharaprom/craftform-backend/internal/model"
)

func respondErr(c echo.Context, err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return c.JSON(http.StatusNotFound, errResp("not found"))
	case errors.Is(err, model.ErrForbidden):
		return c.JSON(http.StatusForbidden, errResp("forbidden"))
	case errors.Is(err, model.ErrConflict):
		return c.JSON(http.StatusConflict, errResp(err.Error()))
	case errors.Is(err, model.ErrUnauthorized):
		return c.JSON(http.StatusUnauthorized, errResp(err.Error()))
	case errors.Is(err, model.ErrFormNotActive):
		return c.JSON(http.StatusForbidden, errResp(err.Error()))
	case errors.Is(err, model.ErrFormNotOpenYet):
		return c.JSON(http.StatusForbidden, errResp(err.Error()))
	case errors.Is(err, model.ErrFormExpired):
		return c.JSON(http.StatusForbidden, errResp(err.Error()))
	case errors.Is(err, model.ErrFormFull):
		return c.JSON(http.StatusForbidden, errResp(err.Error()))
	default:
		return c.JSON(http.StatusInternalServerError, errResp("internal error"))
	}
}

func errResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}
