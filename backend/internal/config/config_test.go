package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnv(t *testing.T) {
	t.Setenv("ENV_PATH", filepath.Join(t.TempDir(), ".env"))
	if err := os.WriteFile(os.Getenv("ENV_PATH"), []byte("PORT=9090\nADMIN_TOKEN=loop_saved\nDISABLE_THRESHOLD=7\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	cfg := Load()

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.AdminToken != "loop_saved" {
		t.Fatalf("AdminToken = %q, want loop_saved", cfg.AdminToken)
	}
	if cfg.DisableThreshold != 7 {
		t.Fatalf("DisableThreshold = %d, want 7", cfg.DisableThreshold)
	}
}

func TestLoadEnvOverridesDotEnv(t *testing.T) {
	t.Setenv("ENV_PATH", filepath.Join(t.TempDir(), ".env"))
	t.Setenv("ADMIN_TOKEN", "loop_real_env")
	if err := os.WriteFile(os.Getenv("ENV_PATH"), []byte("ADMIN_TOKEN=loop_file\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	cfg := Load()

	if cfg.AdminToken != "loop_real_env" {
		t.Fatalf("AdminToken = %q, want loop_real_env", cfg.AdminToken)
	}
}

func TestSaveAdminTokenUpsertsDotEnv(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("PORT=8080\nADMIN_TOKEN=old\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	if err := SaveAdminToken(envPath, "loop_new"); err != nil {
		t.Fatalf("save token: %v", err)
	}

	got, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	want := "PORT=8080\nADMIN_TOKEN=loop_new\n"
	if string(got) != want {
		t.Fatalf("env content = %q, want %q", string(got), want)
	}
}
