package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/sovatharaprom/craftform-backend/internal/config"
	"github.com/sovatharaprom/craftform-backend/internal/db"
	"github.com/sovatharaprom/craftform-backend/internal/handler"
	appmw "github.com/sovatharaprom/craftform-backend/internal/middleware"
	"github.com/sovatharaprom/craftform-backend/internal/repository"
	"github.com/sovatharaprom/craftform-backend/internal/service"
)

func main() {
	cfg := config.Load()

	pool, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("connected to database")

	// ── Dependencies ──────────────────────────────────────────────────────────
	userRepo := repository.NewUserRepo(pool)
	formRepo := repository.NewFormRepo(pool)

	authSvc := service.NewAuthService(cfg, userRepo)
	formSvc := service.NewFormService(formRepo)

	authHandler := handler.NewAuthHandler(authSvc, cfg)
	formHandler := handler.NewFormHandler(formSvc)

	authMW := appmw.NewAuthMiddleware(authSvc)

	// ── Echo setup ────────────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true

	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	e.Static("/uploads", cfg.UploadDir)

	// ── Routes ────────────────────────────────────────────────────────────────
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth
	e.GET("/auth/google", authHandler.RedirectToGoogle)
	e.GET("/auth/google/callback", authHandler.GoogleCallback)
	e.POST("/auth/logout", authHandler.Logout)

	// Protected: requires valid JWT
	priv := e.Group("/api", authMW.Require())
	priv.GET("/me", authHandler.Me)
	priv.GET("/forms", formHandler.List)
	priv.POST("/forms", formHandler.Create)
	priv.PUT("/forms/:id", formHandler.Update)
	priv.DELETE("/forms/:id", formHandler.Delete)
	priv.POST("/forms/:id/duplicate", formHandler.Duplicate)

	// Public: optional JWT (owner sees drafts, others see only active)
	pub := e.Group("/api", authMW.Optional())
	pub.GET("/forms/:id", formHandler.GetByID)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
