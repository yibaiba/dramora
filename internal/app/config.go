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
	Env               string
	HTTPAddr          string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
	DatabaseURL       string
	DataDir           string
	JWTSecret         string
	InlineWorker      bool
	WorkerQueues      []string
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

	env := envString("MANMU_ENV", "local")
	inlineWorker, err := envBool("MANMU_INLINE_WORKER", env == "local")
	if err != nil {
		return Config{}, err
	}

	return Config{
		Env:               env,
		HTTPAddr:          envString("MANMU_HTTP_ADDR", ":8080"),
		ReadHeaderTimeout: readHeaderTimeout,
		ShutdownTimeout:   shutdownTimeout,
		DatabaseURL:       os.Getenv("MANMU_DATABASE_URL"),
		DataDir:           envString("MANMU_DATA_DIR", defaultDataDir()),
		JWTSecret:         envString("MANMU_JWT_SECRET", "dramora-local-dev-secret"),
		InlineWorker:      inlineWorker,
		WorkerQueues:      envCSV("MANMU_WORKER_QUEUES", []string{"default"}),
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

func envBool(key string, fallback bool) (bool, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback, nil
	}
	switch raw {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("parse %s bool: %q", key, raw)
	}
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

func defaultDataDir() string {
	return ".data"
}
