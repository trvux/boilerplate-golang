package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tranvux/boilerplate_golang/database"
	"github.com/tranvux/boilerplate_golang/internal/app"
	"github.com/tranvux/boilerplate_golang/pkg/config"
	pkgdatabase "github.com/tranvux/boilerplate_golang/pkg/database"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 1. Load Application Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		// If config load fails, print to stderr and exit (logger not initialized yet)
		zapLogger, _ := zap.NewProduction()
		zapLogger.Fatal("Failed to load application configuration", zap.Error(err))
	}

	// 2. Initialize Zap Logger
	log, err := logger.NewLogger(cfg.App.Env)
	if err != nil {
		panic("Failed to initialize system logger: " + err.Error())
	}

	log.Info("Starting server boot sequence...",
		zap.String("app_name", cfg.App.Name),
		zap.String("env", cfg.App.Env),
	)

	// 3. Initialize Application Container & manual DI wiring
	appInstance, err := app.NewApp(cfg, log)
	if err != nil {
		log.Fatal("Failed to initialize application container", zap.Error(err))
	}

	// 4. Run database migrations using embedded Goose migration files
	// The database.MigrationsFS holds sql scripts embedded directly inside the binary.
	err = pkgdatabase.RunMigrations(appInstance.Postgres, log, database.MigrationsFS, "migrations")
	if err != nil {
		log.Fatal("Database migrations failed. Aborting startup.", zap.Error(err))
	}

	// 5. Setup context with OS signal listening for Graceful Shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Start HTTP Server in a separate goroutine
	go func() {
		if err := appInstance.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP server failed during runtime", zap.Error(err))
		}
	}()

	// 7. Await termination signal
	<-ctx.Done()
	log.Info("Termination signal received. Initiating graceful shutdown...")

	// 8. Execute graceful shutdown with a strict timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := appInstance.Close(shutdownCtx); err != nil {
		log.Error("Graceful shutdown encountered errors", zap.Error(err))
		os.Exit(1)
	}

	log.Info("Application stopped cleanly")
}
