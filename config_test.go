package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeConfigUsesSimpleActivationDefaults(t *testing.T) {
	cfg, err := decodeConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ActivationEnabled || cfg.ActivationTimesText != "08:00,20:00" {
		t.Fatalf("unexpected schedule defaults: %+v", cfg)
	}
	if cfg.ActivationRequestsPerRun != 2 || cfg.ActivationRandomDelaySecond != 60 || cfg.ActivationConcurrency != 2 {
		t.Fatalf("unexpected execution defaults: %+v", cfg)
	}
}

func TestDecodeConfigNormalizesTimesAndExecutionSettings(t *testing.T) {
	raw := []byte(`
activation_enabled: false
activation_times: "20:00，08:00,20:00"
activation_timezone: UTC
activation_model: gpt-5.5
activation_requests_per_run: 3
activation_random_delay_seconds: 30
activation_concurrency: 4
`)
	cfg, err := decodeConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActivationEnabled || cfg.ActivationTimesText != "08:00,20:00" {
		t.Fatalf("unexpected normalized schedule: %+v", cfg)
	}
	if cfg.ActivationRequestsPerRun != 3 || cfg.ActivationRandomDelaySecond != 30 || cfg.ActivationConcurrency != 4 {
		t.Fatalf("unexpected execution settings: %+v", cfg)
	}
}

func TestDecodeConfigRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"activation_times: 25:00",
		"activation_timezone: Invalid/Timezone",
		"activation_requests_per_run: 0",
		"activation_random_delay_seconds: -1",
		"activation_concurrency: 33",
	}
	for _, raw := range tests {
		t.Run(strings.ReplaceAll(raw, ":", "_"), func(t *testing.T) {
			if _, err := decodeConfig([]byte(raw)); err == nil {
				t.Fatalf("expected invalid config for %q", raw)
			}
		})
	}
}

func TestDecodeConfigWithRestoreUsesPersistedValuesAndExplicitOverrides(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	persisted := defaultConfig()
	persisted.DataDir = dir
	persisted.ConfigPath = configPath
	persisted.LogsPath = filepath.Join(dir, "logs.json")
	persisted.ActivationEnabled = false
	persisted.ActivationTimesText = "06:30,18:45"
	persisted.ActivationTimes, _, _ = parseActivationTimes(persisted.ActivationTimesText)
	persisted.ActivationTimezone = "UTC"
	persisted.ActivationModel = "gpt-4.1"
	persisted.ActivationRequestsPerRun = 5
	persisted.ActivationRandomDelaySecond = 120
	persisted.ActivationConcurrency = 7

	if err := saveConfigFile(configPath, persisted); err != nil {
		t.Fatal(err)
	}

	cfg, err := decodeConfigWithRestore([]byte("data_dir: " + dir + "\nactivation_model: gpt-5.5\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActivationEnabled || cfg.ActivationTimesText != "06:30,18:45" || cfg.ActivationTimezone != "UTC" {
		t.Fatalf("persisted values were not restored from config.json: %+v", cfg)
	}
	if cfg.ActivationModel != "gpt-5.5" || cfg.ActivationRequestsPerRun != 5 || cfg.ActivationRandomDelaySecond != 120 || cfg.ActivationConcurrency != 7 {
		t.Fatalf("explicit value or persisted execution settings were not applied: %+v", cfg)
	}
}

func TestSeparateConfigAndLogsPersistence(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultConfig()
	cfg.DataDir = dir
	cfg.ConfigPath = filepath.Join(dir, "config.json")
	cfg.LogsPath = filepath.Join(dir, "logs.json")
	cfg.ActivationEnabled = false

	service, err := NewService(cfg, &fakeHost{})
	if err != nil {
		t.Fatal(err)
	}

	// 1. 验证已生成独立的 config.json
	loadedCfg, err := loadConfigFile(cfg.ConfigPath)
	if err != nil || loadedCfg == nil {
		t.Fatalf("config.json should be created: %v", err)
	}
	if loadedCfg.ActivationEnabled {
		t.Fatal("expected ActivationEnabled false in config.json")
	}

	// 2. 追加日志，验证只有 logs.json 被写入，config.json 不受影响
	service.appendLog(ExecutionLog{RunID: "test:1", Account: "user@example.com", Trigger: "manual", Status: "success", Detail: "ok"})
	logs, err := loadLogsFile(cfg.LogsPath)
	if err != nil || len(logs) != 1 {
		t.Fatalf("logs.json should contain 1 log: %v", err)
	}

	// 3. 清空日志，验证 logs.json 被重置为空，而 config.json 仍完好无损
	if err := service.ClearLogs(); err != nil {
		t.Fatal(err)
	}
	logsAfterClear, err := loadLogsFile(cfg.LogsPath)
	if err != nil || len(logsAfterClear) != 0 {
		t.Fatalf("logs.json should be empty after clear: %v", err)
	}
	loadedCfgAfter, err := loadConfigFile(cfg.ConfigPath)
	if err != nil || loadedCfgAfter == nil || loadedCfgAfter.ActivationEnabled {
		t.Fatal("config.json must remain intact after clear logs")
	}
}

func TestNewServiceIgnoresMalformedLogsDuringRegistration(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultConfig()
	cfg.DataDir = dir
	cfg.ConfigPath = filepath.Join(dir, "config.json")
	cfg.LogsPath = filepath.Join(dir, "logs.json")
	if err := os.WriteFile(cfg.LogsPath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(cfg, &fakeHost{})
	if err != nil {
		t.Fatalf("malformed logs must not block service registration: %v", err)
	}
	if page := service.QueryLogs(LogQuery{Page: 1, PageSize: 20}); page.Total != 0 {
		t.Fatalf("malformed logs should be treated as empty, got %d entries", page.Total)
	}
}

func TestNewServiceFallsBackWhenPersistedConfigIsInvalid(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultConfig()
	cfg.DataDir = dir
	cfg.ConfigPath = filepath.Join(dir, "config.json")
	cfg.LogsPath = filepath.Join(dir, "logs.json")
	if err := os.WriteFile(cfg.ConfigPath, []byte(`{"activation_model":"","activation_requests_per_run":0}`), 0o600); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(cfg, &fakeHost{})
	if err != nil {
		t.Fatalf("invalid persisted config must not block service registration: %v", err)
	}
	if got := service.Config(); got.ActivationModel != cfg.ActivationModel || got.ActivationRequestsPerRun != cfg.ActivationRequestsPerRun {
		t.Fatalf("service did not fall back to runtime config: %+v", got)
	}
}

func TestNewServiceRestoresPersistedConfigBeforeSavingSnapshot(t *testing.T) {
	dir := t.TempDir()
	paths := defaultConfig()
	paths.DataDir = dir
	paths.ConfigPath = filepath.Join(dir, "config.json")
	paths.LogsPath = filepath.Join(dir, "logs.json")
	persisted := paths
	persisted.ActivationEnabled = false
	persisted.ActivationTimesText = "02:15,08:15,14:15,20:15"
	persisted.ActivationTimes, _, _ = parseActivationTimes(persisted.ActivationTimesText)
	persisted.ActivationModel = "gpt-4.1"
	if err := saveConfigFile(paths.ConfigPath, persisted); err != nil {
		t.Fatal(err)
	}

	// Simulate CPA re-install/reconfigure passing only its generic defaults.
	input := paths
	input.ActivationEnabled = true
	input.ActivationTimesText = "08:00,20:00"
	input.ActivationTimes, _, _ = parseActivationTimes(input.ActivationTimesText)
	input.ActivationModel = "gpt-5.5"
	service, err := NewService(input, &fakeHost{})
	if err != nil {
		t.Fatal(err)
	}
	got := service.Config()
	if got.ActivationEnabled || got.ActivationTimesText != persisted.ActivationTimesText || got.ActivationModel != persisted.ActivationModel {
		t.Fatalf("NewService did not restore config.json: %+v", got)
	}
}

func TestDecodeConfigWithRestoreDoesNotReadLegacyStateFile(t *testing.T) {
	dir := t.TempDir()
	legacyState := filepath.Join(dir, "state.json")
	if err := os.WriteFile(legacyState, []byte(`{"config":{"activation_model":"legacy-model"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := decodeConfigWithRestore([]byte("data_dir: " + dir + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActivationModel != defaultConfig().ActivationModel {
		t.Fatalf("legacy state.json must be ignored, got model %q", cfg.ActivationModel)
	}
	if cfg.ConfigPath != filepath.Join(dir, "config.json") || cfg.LogsPath != filepath.Join(dir, "logs.json") {
		t.Fatalf("unexpected split persistence paths: config=%q logs=%q", cfg.ConfigPath, cfg.LogsPath)
	}
}

func TestRegistrationDoesNotClaimScheduler(t *testing.T) {
	registration := pluginRegistration()
	if registration.Capabilities.Scheduler {
		t.Fatal("activation-only plugin must not claim scheduler capability")
	}
	if len(registration.Metadata.ConfigFields) != 0 {
		t.Fatalf("plugin manager should not expose page settings, got %d fields", len(registration.Metadata.ConfigFields))
	}
}
