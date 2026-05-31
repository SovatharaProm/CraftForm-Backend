package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sovatharaprom/craftform-backend/internal/config"
	"github.com/sovatharaprom/craftform-backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
	cfg *config.Config
}

func NewAuthHandler(svc *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

func (h *AuthHandler) RedirectToGoogle(c echo.Context) error {
	state := randomState()
	c.SetCookie(&http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		MaxAge:   300,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	return c.Redirect(http.StatusTemporaryRedirect, h.svc.AuthURL(state))
}

func (h *AuthHandler) GoogleCallback(c echo.Context) error {
	cookie, err := c.Cookie("oauth_state")
	if err != nil || cookie.Value != c.QueryParam("state") {
		return c.JSON(http.StatusBadRequest, errResp("invalid oauth state"))
	}

	user, err := h.svc.ExchangeCode(c.Request().Context(), c.QueryParam("code"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResp("authentication failed"))
	}

	token, err := h.svc.IssueJWT(user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResp("could not issue token"))
	}

	return c.Redirect(http.StatusTemporaryRedirect, h.cfg.FrontendURL+"/auth/callback?token="+token)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID := c.Get("userID").(string)
	return c.JSON(http.StatusOK, map[string]string{"userId": userID})
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
