package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	productDelivery "github.com/tranvux/boilerplate_golang/internal/modules/product/delivery/http"
	productRepository "github.com/tranvux/boilerplate_golang/internal/modules/product/repository"
	productUsecase "github.com/tranvux/boilerplate_golang/internal/modules/product/usecase"
	"github.com/tranvux/boilerplate_golang/pkg/config"
	"github.com/tranvux/boilerplate_golang/pkg/database"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"github.com/tranvux/boilerplate_golang/pkg/messaging"
	"go.uber.org/zap"
)

type App struct {
	Cfg        *config.Config
	Log        logger.Logger
	Postgres   *database.PostgresDB
	Redis      *database.RedisClient
	Producer   messaging.Producer
	HttpServer *http.Server
}

func NewApp(cfg *config.Config, log logger.Logger) (*App, error) {
	// Initialize PostgreSQL
	postgres, err := database.NewPostgresDB(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres: %w", err)
	}

	// Initialize Redis (optional - if fails, log but proceed depending on config, but here we enforce it)
	redis, err := database.NewRedisClient(cfg, log)
	if err != nil {
		log.Warn("Redis connection failed. Features requiring cache may fail.", zap.Error(err))
	}

	// Initialize Kafka Producer (optional - if fails, log and proceed)
	producer, err := messaging.NewKafkaProducer(cfg, log)
	if err != nil {
		log.Warn("Kafka Producer initialization failed. Event publishing disabled.", zap.Error(err))
	}

	// Initialize Gin Engine
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Apply Middlewares
	router.Use(RequestIDMiddleware())
	router.Use(ZapLoggerMiddleware(log))
	router.Use(RecoveryMiddleware(log))
	router.Use(CORSMiddleware())

	// Health Check Endpoint
	router.GET("/health", func(c *gin.Context) {
		status := gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
			"postgres": "up",
		}
		if redis != nil {
			status["redis"] = "up"
		} else {
			status["redis"] = "down"
		}
		c.JSON(http.StatusOK, status)
	})

	// API Version Group
	apiV1 := router.Group("/api/v1")

	// ==========================================
	// MODULE WIRING (MANUAL DEPENDENCY INJECTION)
	// ==========================================

	// 1. Product Module wiring
	prodRepo := productRepository.NewPostgresProductRepository(postgres)
	prodUsecase := productUsecase.NewProductUsecase(prodRepo, log)
	productDelivery.RegisterHandlers(apiV1, prodUsecase)

	// ==========================================

	// Configure HTTP Server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.TimeoutSeconds * time.Second,
		WriteTimeout: cfg.Server.TimeoutSeconds * time.Second,
	}

	return &App{
		Cfg:        cfg,
		Log:        log,
		Postgres:   postgres,
		Redis:      redis,
		Producer:   producer,
		HttpServer: server,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.Log.Info("Starting HTTP Server", zap.String("addr", a.HttpServer.Addr))
	if err := a.HttpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start http server: %w", err)
	}
	return nil
}

func (a *App) Close(ctx context.Context) error {
	a.Log.Info("Shutting down Application Container gracefully...")

	// 1. Shutdown HTTP Server
	if err := a.HttpServer.Shutdown(ctx); err != nil {
		a.Log.Error("Failed to shutdown HTTP server cleanly", zap.Error(err))
	}

	// 2. Shutdown Kafka Producer
	if a.Producer != nil {
		if err := a.Producer.Close(); err != nil {
			a.Log.Error("Failed to close Kafka Producer", zap.Error(err))
		}
	}

	// 3. Shutdown Redis Client
	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			a.Log.Error("Failed to close Redis client", zap.Error(err))
		}
	}

	// 4. Shutdown Postgres DB
	if a.Postgres != nil {
		if err := a.Postgres.Close(); err != nil {
			a.Log.Error("Failed to close Postgres connection", zap.Error(err))
		}
	}

	a.Log.Info("Graceful shutdown completed successfully")
	return nil
}
