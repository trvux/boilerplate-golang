package database

import (
	"context"
	"fmt"
	"time"

	"github.com/tranvux/boilerplate_golang/pkg/config"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type PostgresDB struct {
	*gorm.DB
}

func NewPostgresDB(cfg *config.Config, log logger.Logger) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: newGormZapLogger(log, cfg.App.Env),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql database instance: %w", err)
	}

	// Setup connection pooling
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetimeMinutes) * time.Minute)

	// Validate connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Successfully connected to PostgreSQL database",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.DBName),
	)

	return &PostgresDB{DB: db}, nil
}

func (db *PostgresDB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// gormZapLogger bridges GORM logging calls to Zap Logger.
type gormZapLogger struct {
	ZapLogger     logger.Logger
	LogLevel      gormlogger.LogLevel
	SlowThreshold time.Duration
}

func newGormZapLogger(log logger.Logger, env string) gormlogger.Interface {
	level := gormlogger.Warn
	if env == "development" {
		level = gormlogger.Info
	}

	return &gormZapLogger{
		ZapLogger:     log,
		LogLevel:      level,
		SlowThreshold: 200 * time.Millisecond,
	}
}

func (l *gormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *gormZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Info(fmt.Sprintf(msg, data...))
	}
}

func (l *gormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.ZapLogger.Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *gormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.ZapLogger.Error(fmt.Sprintf(msg, data...))
	}
}

func (l *gormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.Duration("elapsed_ms", elapsed),
		zap.Int64("rows_affected", rows),
		zap.String("sql", sql),
	}

	switch {
	case err != nil && err != gorm.ErrRecordNotFound && l.LogLevel >= gormlogger.Error:
		l.ZapLogger.Error("GORM database error", append(fields, zap.Error(err))...)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		l.ZapLogger.Warn("SLOW SQL QUERY (>200ms)", fields...)
	case l.LogLevel >= gormlogger.Info:
		l.ZapLogger.Debug("SQL execution details", fields...)
	}
}
