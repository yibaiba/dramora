package app

import "testing"

func TestLoadConfigDefaultsInlineWorkerForLocalEnv(t *testing.T) {
	isolateConfigEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.InlineWorker {
		t.Fatal("expected inline worker enabled by default in local env")
	}
}

func TestLoadConfigDisablesInlineWorkerOutsideLocalByDefault(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("MANMU_ENV", "production")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.InlineWorker {
		t.Fatal("expected inline worker disabled by default outside local env")
	}
}

func TestLoadConfigParsesInlineWorkerOverride(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("MANMU_ENV", "production")
	t.Setenv("MANMU_INLINE_WORKER", "yes")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.InlineWorker {
		t.Fatal("expected inline worker override to enable worker")
	}
}

func TestLoadConfigRejectsInvalidInlineWorkerOverride(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("MANMU_INLINE_WORKER", "sometimes")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected invalid inline worker value to fail")
	}
}

func isolateConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"MANMU_DATABASE_URL",
		"MANMU_ENV",
		"MANMU_HTTP_ADDR",
		"MANMU_INLINE_WORKER",
		"MANMU_READ_HEADER_TIMEOUT",
		"MANMU_SHUTDOWN_TIMEOUT",
		"MANMU_WORKER_QUEUES",
	} {
		t.Setenv(key, "")
	}
}
