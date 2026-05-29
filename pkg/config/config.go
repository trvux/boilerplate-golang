package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
}

type AppConfig struct {
	Env  string
	Name string
}

type ServerConfig struct {
	Host           string
	Port           int
	TimeoutSeconds time.Duration
}

type DatabaseConfig struct {
	Host                   string
	Port                   int
	User                   string
	Password               string
	DBName                 string
	SSLMode                string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers  []string
	ClientID string
	GroupID  string
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	var val int
	_, err := fmt.Sscanf(valueStr, "%d", &val)
	if err != nil {
		return defaultValue
	}
	return val
}

func getEnvSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	parts := strings.Split(valueStr, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func LoadConfig() (*Config, error) {
	// Load .env file if present (useful for local development overrides)
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Name: getEnv("APP_NAME", "boilerplate-golang"),
		},
		Server: ServerConfig{
			Host:           getEnv("SERVER_HOST", "0.0.0.0"),
			Port:           getEnvInt("SERVER_PORT", 8080),
			TimeoutSeconds: time.Duration(getEnvInt("SERVER_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:                   getEnv("DATABASE_HOST", "localhost"),
			Port:                   getEnvInt("DATABASE_PORT", 5432),
			User:                   getEnv("DATABASE_USER", "postgres"),
			Password:               getEnv("DATABASE_PASSWORD", "password"),
			DBName:                 getEnv("DATABASE_DBNAME", "boilerplate_db"),
			SSLMode:                getEnv("DATABASE_SSL_MODE", "disable"),
			MaxOpenConns:           getEnvInt("DATABASE_MAX_OPEN_CONNS", 100),
			MaxIdleConns:           getEnvInt("DATABASE_MAX_IDLE_CONNS", 10),
			ConnMaxLifetimeMinutes: getEnvInt("DATABASE_CONN_MAX_LIFETIME_MINUTES", 30),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers:  getEnvSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			ClientID: getEnv("KAFKA_CLIENT_ID", "boilerplate-golang-client"),
			GroupID:  getEnv("KAFKA_GROUP_ID", "boilerplate-golang-group"),
		},
	}

	return cfg, nil
}
