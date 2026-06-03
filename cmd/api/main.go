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

	if err := db.Migrate(pool, "migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("migrations applied")

	// ── Dependencies ──────────────────────────────────────────────────────────
	userRepo     := repository.NewUserRepo(pool)
	formRepo     := repository.NewFormRepo(pool)
	responseRepo := repository.NewResponseRepo(pool)

	authSvc     := service.NewAuthService(cfg, userRepo)
	formSvc     := service.NewFormService(formRepo)
	responseSvc := service.NewResponseService(responseRepo, formRepo, userRepo)
	uploadSvc   := service.NewUploadService(cfg)

	authHandler     := handler.NewAuthHandler(authSvc, cfg)
	formHandler     := handler.NewFormHandler(formSvc)
	responseHandler := handler.NewResponseHandler(responseSvc)
	uploadHandler   := handler.NewUploadHandler(uploadSvc)

	authMW := appmw.NewAuthMiddleware(authSvc)

	// ── Echo setup ────────────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.Server.MaxHeaderBytes = 5 * 1024 * 1024 // allow up to 5 MB bodies

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

	// Public (no auth required)
	e.GET("/api/forms/public", formHandler.ListPublic)

	// Optional auth (owner sees drafts + analytics, others see only active)
	opt := e.Group("/api", authMW.Optional())
	opt.GET("/forms/:id", formHandler.GetByID)
	opt.POST("/forms/:id/responses", responseHandler.Submit)

	// Protected (valid JWT required)
	priv := e.Group("/api", authMW.Require())
	priv.GET("/me", authHandler.Me)

	priv.GET("/forms", formHandler.List)
	priv.POST("/forms", formHandler.Create)
	priv.PUT("/forms/:id", formHandler.Update)
	priv.DELETE("/forms/:id", formHandler.Delete)
	priv.POST("/forms/:id/duplicate", formHandler.Duplicate)

	priv.GET("/forms/:id/responses", responseHandler.List)
	priv.GET("/forms/:id/responses/summary", responseHandler.Analytics)
	priv.GET("/forms/:id/responses/:rid", responseHandler.GetByID)
	priv.DELETE("/forms/:id/responses/:rid", responseHandler.Delete)

	priv.POST("/upload", uploadHandler.Upload)

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
