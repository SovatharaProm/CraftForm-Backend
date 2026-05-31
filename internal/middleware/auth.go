package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sovatharaprom/craftform-backend/internal/service"
)

const UserIDKey = "userID"

type AuthMiddleware struct {
	svc *service.AuthService
}

func NewAuthMiddleware(svc *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{svc: svc}
}

// Require rejects requests without a valid JWT.
func (m *AuthMiddleware) Require() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, err := m.extractUserID(c)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			c.Set(UserIDKey, userID)
			return next(c)
		}
	}
}

// Optional attaches the user ID when a valid JWT is present but does not block the request.
func (m *AuthMiddleware) Optional() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if userID, err := m.extractUserID(c); err == nil {
				c.Set(UserIDKey, userID)
			}
			return next(c)
		}
	}
}

func (m *AuthMiddleware) extractUserID(c echo.Context) (string, error) {
	header := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", echo.ErrUnauthorized
	}
	return m.svc.VerifyJWT(strings.TrimPrefix(header, "Bearer "))
}

// GetUserID returns the authenticated user ID from context, or empty string if not set.
func GetUserID(c echo.Context) string {
	v, _ := c.Get(UserIDKey).(string)
	return v
}
