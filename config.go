package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	pluginID = "codex-keepalive"
)

// pluginVersion 会在发布构建时由 -ldflags 注入；默认值用于本地开发和测试。
var pluginVersion = "0.1.0"

type clockTime struct {
	Hour   int
	Minute int
}

func (value clockTime) String() string {
	return fmt.Sprintf("%02d:%02d", value.Hour, value.Minute)
}

type Config struct {
	DataDir                     string
	ConfigPath                  string
	LogsPath                    string
	ActivationEnabled           bool
	ActivationTimes             []clockTime
	ActivationTimesText         string
	ActivationTimezone          string
	ActivationModel             string
	ActivationRequestsPerRun    int
	ActivationRandomDelaySecond int
	ActivationConcurrency       int
}

func resolvePaths(dataDir string) (dir, cfgPath, logPath string) {
	home, _ := os.UserHomeDir()
	dir = filepath.Join(home, ".cli-proxy-api", "plugins", pluginID)
	if trimmed := strings.TrimSpace(dataDir); trimmed != "" {
		dir = filepath.Clean(trimmed)
	}
	cfgPath = filepath.Join(dir, "config.json")
	logPath = filepath.Join(dir, "logs.json")
	return dir, cfgPath, logPath
}

type registration struct {
	SchemaVersion uint32                   `json:"schema_version"`
	Metadata      pluginapi.Metadata       `json:"metadata"`
	Capabilities  registrationCapabilities `json:"capabilities"`
}

type registrationCapabilities struct {
	Scheduler     bool `json:"scheduler"`
	ManagementAPI bool `json:"management_api"`
}

func defaultConfig() Config {
	dir, cfgPath, logPath := resolvePaths("")
	times, text, _ := parseActivationTimes("07:00,12:15,17:30")
	return Config{
		DataDir:                     dir,
		ConfigPath:                  cfgPath,
		LogsPath:                    logPath,
		ActivationEnabled:           true,
		ActivationTimes:             times,
		ActivationTimesText:         text,
		ActivationTimezone:          "Asia/Shanghai",
		ActivationModel:             "gpt-5.6-sol",
		ActivationRequestsPerRun:    2,
		ActivationRandomDelaySecond: 60,
		ActivationConcurrency:       2,
	}
}

func decodeConfig(raw []byte) (Config, error) {
	return decodeConfigWithBase(raw, defaultConfig())
}

// decodeConfigWithBase 在给定的基础配置上应用 CPA 本次传入的显式字段。
// 基础配置来自上一次持久化快照时，空配置或缺少字段即可自然恢复旧值。
func decodeConfigWithBase(raw []byte, base Config) (Config, error) {
	cfg := base
	if len(raw) == 0 {
		if _, err := time.LoadLocation(cfg.ActivationTimezone); err != nil {
			return Config{}, fmt.Errorf("activation_timezone must be a valid IANA timezone")
		}
		if cfg.ActivationModel == "" {
			return Config{}, fmt.Errorf("activation_model must not be empty")
		}
		return cfg, nil
	}
	values := parseFlatYAML(raw)
	dataDir := values["data_dir"]
	if dataDir != "" {
		dir, cfgPath, logPath := resolvePaths(dataDir)
		cfg.DataDir = dir
		cfg.ConfigPath = cfgPath
		cfg.LogsPath = logPath
	}
	if value := values["activation_enabled"]; value != "" {
		enabled, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return Config{}, fmt.Errorf("activation_enabled must be true or false")
		}
		cfg.ActivationEnabled = enabled
	}
	if value := values["activation_times"]; value != "" {
		times, text, err := parseActivationTimes(value)
		if err != nil {
			return Config{}, err
		}
		cfg.ActivationTimes = times
		cfg.ActivationTimesText = text
	}
	if value := strings.TrimSpace(values["activation_timezone"]); value != "" {
		cfg.ActivationTimezone = value
	}
	if _, err := time.LoadLocation(cfg.ActivationTimezone); err != nil {
		return Config{}, fmt.Errorf("activation_timezone must be a valid IANA timezone")
	}
	if value := strings.TrimSpace(values["activation_model"]); value != "" {
		cfg.ActivationModel = value
	}
	if cfg.ActivationModel == "" {
		return Config{}, fmt.Errorf("activation_model must not be empty")
	}
	requestsValue := values["activation_requests_per_run"]
	if requestsValue != "" {
		count, err := strconv.Atoi(strings.TrimSpace(requestsValue))
		if err != nil || count < 1 || count > 10 {
			return Config{}, fmt.Errorf("activation_requests_per_run must be between 1 and 10")
		}
		cfg.ActivationRequestsPerRun = count
	}
	if value := values["activation_random_delay_seconds"]; value != "" {
		seconds, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || seconds < 0 || seconds > 3600 {
			return Config{}, fmt.Errorf("activation_random_delay_seconds must be between 0 and 3600")
		}
		cfg.ActivationRandomDelaySecond = seconds
	}
	if value := values["activation_concurrency"]; value != "" {
		count, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || count < 1 || count > 32 {
			return Config{}, fmt.Errorf("activation_concurrency must be between 1 and 32")
		}
		cfg.ActivationConcurrency = count
	}
	return cfg, nil
}

func parseActivationTimes(raw string) ([]clockTime, string, error) {
	raw = strings.ReplaceAll(raw, "，", ",")
	parts := strings.Split(raw, ",")
	seen := make(map[int]struct{})
	times := make([]clockTime, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := time.Parse("15:04", part)
		if err != nil {
			return nil, "", fmt.Errorf("activation_times must use comma-separated HH:mm values")
		}
		minutes := parsed.Hour()*60 + parsed.Minute()
		if _, exists := seen[minutes]; exists {
			continue
		}
		seen[minutes] = struct{}{}
		times = append(times, clockTime{Hour: parsed.Hour(), Minute: parsed.Minute()})
	}
	if len(times) == 0 {
		return nil, "", fmt.Errorf("activation_times must contain at least one HH:mm value")
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i].Hour*60+times[i].Minute < times[j].Hour*60+times[j].Minute
	})
	text := make([]string, len(times))
	for index, value := range times {
		text[index] = value.String()
	}
	return times, strings.Join(text, ","), nil
}

// parseFlatYAML 只读取本插件公开的标量配置。宿主注入的 enabled、priority 等字段会被忽略。
func parseFlatYAML(raw []byte) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			if unquoted, err := strconv.Unquote(value); err == nil {
				value = unquoted
			} else {
				value = value[1 : len(value)-1]
			}
		}
		result[key] = value
	}
	return result
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             "Codex 定时唤醒",
			Version:          pluginVersion,
			Author:           "Jie",
			GitHubRepository: "https://github.com/elunez/codex-keepalive",
			Logo:             "https://raw.githubusercontent.com/router-for-me/CLIProxyAPI/main/docs/logo.png",
		},
		Capabilities: registrationCapabilities{Scheduler: false, ManagementAPI: true},
	}
}
