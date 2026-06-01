package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                  string
	DBPath                string
	AdminToken            string
	DisableThreshold      int
	RecoveryProbeEnabled  bool
	ProbeBackoffBaseMin   int
	ProbeBackoffMaxMin    int
	ProbeCheckIntervalSec int
	MaxProxyAttempts      int
	UpstreamTimeoutSec    int

	// Concurrency / pooling hardening
	DBReadPoolSize           int
	ProxyMaxConcurrency      int
	ProxyBacklog             int
	ProxyBacklogTimeoutSec   int
	MaxRequestBodyMB         int
	ProbeBatchConcurrency    int
	ResponseHeaderTimeoutSec int
}

func Load() Config {
	return Config{
		Port:                  getEnv("PORT", "8080"),
		DBPath:                getEnv("DB_PATH", "loop.db"),
		AdminToken:            os.Getenv("ADMIN_TOKEN"),
		DisableThreshold:      getEnvInt("DISABLE_THRESHOLD", 5),
		RecoveryProbeEnabled:  getEnvBool("RECOVERY_PROBE_ENABLED", true),
		ProbeBackoffBaseMin:   getEnvInt("PROBE_BACKOFF_BASE_MIN", 60),
		ProbeBackoffMaxMin:    getEnvInt("PROBE_BACKOFF_MAX_MIN", 1440),
		ProbeCheckIntervalSec: getEnvInt("PROBE_CHECK_INTERVAL_SEC", 300),
		MaxProxyAttempts:      getEnvInt("MAX_PROXY_ATTEMPTS", 5),
		UpstreamTimeoutSec:    getEnvInt("UPSTREAM_TIMEOUT_SEC", 300),

		DBReadPoolSize:           getEnvInt("DB_READ_POOL_SIZE", 8),
		ProxyMaxConcurrency:      getEnvInt("PROXY_MAX_CONCURRENCY", 200),
		ProxyBacklog:             getEnvInt("PROXY_BACKLOG", 100),
		ProxyBacklogTimeoutSec:   getEnvInt("PROXY_BACKLOG_TIMEOUT_SEC", 5),
		MaxRequestBodyMB:         getEnvInt("MAX_REQUEST_BODY_MB", 32),
		ProbeBatchConcurrency:    getEnvInt("PROBE_BATCH_CONCURRENCY", 10),
		ResponseHeaderTimeoutSec: getEnvInt("RESPONSE_HEADER_TIMEOUT_SEC", 60),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
