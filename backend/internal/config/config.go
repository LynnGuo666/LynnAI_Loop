package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port                  string
	DBPath                string
	AdminToken            string
	EnvPath               string
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
	envPath := getEnvPath()
	fileEnv := loadDotEnv(envPath)

	return Config{
		Port:                  getConfigValue("PORT", "8080", fileEnv),
		DBPath:                getConfigValue("DB_PATH", "loop.db", fileEnv),
		AdminToken:            getConfigValue("ADMIN_TOKEN", "", fileEnv),
		EnvPath:               envPath,
		DisableThreshold:      getConfigInt("DISABLE_THRESHOLD", 5, fileEnv),
		RecoveryProbeEnabled:  getConfigBool("RECOVERY_PROBE_ENABLED", true, fileEnv),
		ProbeBackoffBaseMin:   getConfigInt("PROBE_BACKOFF_BASE_MIN", 60, fileEnv),
		ProbeBackoffMaxMin:    getConfigInt("PROBE_BACKOFF_MAX_MIN", 1440, fileEnv),
		ProbeCheckIntervalSec: getConfigInt("PROBE_CHECK_INTERVAL_SEC", 300, fileEnv),
		MaxProxyAttempts:      getConfigInt("MAX_PROXY_ATTEMPTS", 5, fileEnv),
		UpstreamTimeoutSec:    getConfigInt("UPSTREAM_TIMEOUT_SEC", 300, fileEnv),

		DBReadPoolSize:           getConfigInt("DB_READ_POOL_SIZE", 8, fileEnv),
		ProxyMaxConcurrency:      getConfigInt("PROXY_MAX_CONCURRENCY", 200, fileEnv),
		ProxyBacklog:             getConfigInt("PROXY_BACKLOG", 100, fileEnv),
		ProxyBacklogTimeoutSec:   getConfigInt("PROXY_BACKLOG_TIMEOUT_SEC", 5, fileEnv),
		MaxRequestBodyMB:         getConfigInt("MAX_REQUEST_BODY_MB", 32, fileEnv),
		ProbeBatchConcurrency:    getConfigInt("PROBE_BATCH_CONCURRENCY", 10, fileEnv),
		ResponseHeaderTimeoutSec: getConfigInt("RESPONSE_HEADER_TIMEOUT_SEC", 60, fileEnv),
	}
}

func SaveAdminToken(envPath, token string) error {
	return upsertDotEnvValue(envPath, "ADMIN_TOKEN", token)
}

func getEnvPath() string {
	if v := os.Getenv("ENV_PATH"); v != "" {
		return v
	}
	return ".env"
}

func getConfigValue(key, fallback string, fileEnv map[string]string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v := fileEnv[key]; v != "" {
		return v
	}
	return fallback
}

func getConfigInt(key string, fallback int, fileEnv map[string]string) int {
	if v := getConfigValue(key, "", fileEnv); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getConfigBool(key string, fallback bool, fileEnv map[string]string) bool {
	if v := getConfigValue(key, "", fileEnv); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func loadDotEnv(path string) map[string]string {
	values := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return values
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		values[key] = strings.Trim(value, `"'`)
	}
	return values
}

func upsertDotEnvValue(path, key, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := []string{}
	if len(content) > 0 {
		lines = strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	}

	entry := key + "=" + value
	updated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		lines = append(lines, entry)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}
