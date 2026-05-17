// Package config provides configuration loading from environment variables
// for all Furab microservices.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration values for a microservice.
type Config struct {
	// Server settings
	ServerHost string
	ServerPort int

	// Database settings
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Kafka settings
	KafkaBrokers []string

	// RabbitMQ settings
	RabbitMQURL string

	// Redis settings
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// JWT settings
	JWTSecret     string
	JWTExpiration int // in hours

	// Service identification
	ServiceName string
	Environment string // development, staging, production
}

// Load reads configuration from environment variables with sensible defaults.
func Load(serviceName string) *Config {
	return &Config{
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort: getEnvInt("SERVER_PORT", 8080),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "furab"),
		DBPassword: getEnv("DB_PASSWORD", "furab_secret"),
		DBName:     getEnv("DB_NAME", serviceName),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		KafkaBrokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		JWTSecret:     getEnv("JWT_SECRET", "furab-default-secret-change-in-production"),
		JWTExpiration: getEnvInt("JWT_EXPIRATION_HOURS", 24),

		ServiceName: serviceName,
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// DatabaseURL returns the PostgreSQL connection URL.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

// ServerAddr returns the full server address (host:port).
func (c *Config) ServerAddr() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

// RedisAddr returns the full Redis address (host:port).
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt reads an integer environment variable or returns a default value.
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
