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
