package database

import (
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
)

func RunMigrations(db *PostgresDB, log logger.Logger, embedFS embed.FS, dir string) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from GORM database: %w", err)
	}

	// Route Goose logging through Zap Logger using a custom writer
	goose.SetLogger(&gooseZapLogger{log: log})

	goose.SetBaseFS(embedFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set database dialect for goose: %w", err)
	}

	log.Info("Starting database migrations...")
	if err := goose.Up(sqlDB, dir); err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}

	log.Info("Database migrations completed successfully")
	return nil
}

type gooseZapLogger struct {
	log logger.Logger
}

func (l *gooseZapLogger) Fatalf(format string, v ...interface{}) {
	l.log.Fatal(fmt.Sprintf(format, v...))
}

func (l *gooseZapLogger) Printf(format string, v ...interface{}) {
	l.log.Info(fmt.Sprintf(format, v...))
}
