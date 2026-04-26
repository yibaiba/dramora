package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const Version = "dev"

type Config struct {
	Env                   string
	HTTPAddr              string
	ReadHeaderTimeout     time.Duration
	ShutdownTimeout       time.Duration
	DatabaseURL           string
	DefaultOrganizationID string
	WorkerQueues          []string
}

func LoadConfig() (Config, error) {
	readHeaderTimeout, err := envDuration("MANMU_READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := envDuration("MANMU_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Env:                   envString("MANMU_ENV", "local"),
		HTTPAddr:              envString("MANMU_HTTP_ADDR", ":8080"),
		ReadHeaderTimeout:     readHeaderTimeout,
		ShutdownTimeout:       shutdownTimeout,
		DatabaseURL:           os.Getenv("MANMU_DATABASE_URL"),
		DefaultOrganizationID: envString("MANMU_DEFAULT_ORGANIZATION_ID", "00000000-0000-0000-0000-000000000001"),
		WorkerQueues:          envCSV("MANMU_WORKER_QUEUES", []string{"default"}),
	}, nil
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envCSV(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(raw)
	if err == nil {
		return duration, nil
	}

	seconds, convErr := strconv.Atoi(raw)
	if convErr != nil {
		return 0, fmt.Errorf("parse %s duration: %w", key, err)
	}
	return time.Duration(seconds) * time.Second, nil
}
