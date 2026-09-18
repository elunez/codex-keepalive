package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const managementBasePath = "/plugins/" + pluginID

type statusPayload struct {
	PluginID        string    `json:"plugin_id"`
	Version         string    `json:"version"`
	GeneratedAt     time.Time `json:"generated_at"`
	Enabled         bool      `json:"enabled"`
	ActivationTimes string    `json:"activation_times"`
	Timezone        string    `json:"timezone"`
	RequestsPerRun  int       `json:"requests_per_run"`
	RandomDelayMax  int       `json:"random_delay_max_seconds"`
	Concurrency     int       `json:"concurrency"`
	NextScheduledAt time.Time `json:"next_scheduled_at,omitempty"`
	Run             RunStatus `json:"run"`
}

func registerManagement() pluginapi.ManagementRegistrationResponse {
	return pluginapi.ManagementRegistrationResponse{
		Resources: []pluginapi.ResourceRoute{{
			Path:        "/status",
			Menu:        "定时唤醒",
			Description: "按计划唤醒 Codex 账号并查看执行日志。",
		}},
		Routes: []pluginapi.ManagementRoute{
			{Method: http.MethodGet, Path: managementBasePath + "/status", Description: "读取定时唤醒状态。"},
			{Method: http.MethodGet, Path: managementBasePath + "/logs", Description: "分页读取执行日志。"},
			{Method: http.MethodPost, Path: managementBasePath + "/execute", Description: "立即执行一次唤醒任务。"},
			{Method: http.MethodDelete, Path: managementBasePath + "/logs", Description: "清空执行日志。"},
			// /config 已被 CPA 内置插件配置接口占用，这里使用独立路径，
			// 确保请求进入插件处理器并写入持久化的 config.json。
			{Method: http.MethodGet, Path: managementBasePath + "/settings", Description: "读取当前插件配置。"},
			{Method: http.MethodPut, Path: managementBasePath + "/settings", Description: "更新插件配置。"},
		},
	}
}

func handleManagement(raw []byte) ([]byte, error) {
	var request pluginapi.ManagementRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return nil, err
	}
	current := currentService()
	if current == nil {
		return okEnvelope(jsonResponse(http.StatusServiceUnavailable, map[string]string{"error": "plugin service unavailable"}))
	}
	path := normalizeManagementPath(request.Path)
	method := strings.ToUpper(request.Method)
	if isResourcePath(request.Path) {
		if method != http.MethodGet || path != "/status" {
			return okEnvelope(jsonResponse(http.StatusNotFound, map[string]string{"error": "not found"}))
		}
		return okEnvelope(htmlResponse(http.StatusOK, []byte(renderStatusPage())))
	}
	switch {
	case method == http.MethodGet && path == "/status":
		if err := current.ReloadPersistedConfig(); err != nil {
			return okEnvelope(jsonResponse(http.StatusInternalServerError, map[string]string{"error": "读取持久化配置失败: " + err.Error()}))
		}
		return okEnvelope(jsonResponse(http.StatusOK, buildStatus(current)))
	case method == http.MethodGet && path == "/logs":
		query := LogQuery{
			Page:     queryInt(request.Query.Get("page"), 1),
			PageSize: queryInt(request.Query.Get("page_size"), 20),
			Account:  request.Query.Get("account"),
			Result:   request.Query.Get("result"),
			Trigger:  request.Query.Get("trigger"),
		}
		return okEnvelope(jsonResponse(http.StatusOK, current.QueryLogs(query)))
	case method == http.MethodPost && (path == "/execute" || path == "/activation/refresh"):
		runID, err := current.StartManualActivation()
		if err != nil {
			return okEnvelope(jsonResponse(http.StatusConflict, map[string]string{"error": err.Error()}))
		}
		return okEnvelope(jsonResponse(http.StatusAccepted, map[string]any{"ok": true, "run_id": runID}))
	case method == http.MethodDelete && path == "/logs":
		if err := current.ClearLogs(); err != nil {
			return okEnvelope(jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()}))
		}
		return okEnvelope(jsonResponse(http.StatusOK, map[string]bool{"ok": true}))
	case method == http.MethodGet && path == "/settings":
		if err := current.ReloadPersistedConfig(); err != nil {
			return okEnvelope(jsonResponse(http.StatusInternalServerError, map[string]string{"error": "读取持久化配置失败: " + err.Error()}))
		}
		return okEnvelope(jsonResponse(http.StatusOK, buildConfigPayload(current.Config())))
	case method == http.MethodPut && path == "/settings":
		newCfg, err := decodeConfigJSON(request.Body, current.Config())
		if err != nil {
			return okEnvelope(jsonResponse(http.StatusBadRequest, map[string]string{"error": err.Error()}))
		}
		if err := current.UpdateConfig(newCfg); err != nil {
			return okEnvelope(jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()}))
		}
		return okEnvelope(jsonResponse(http.StatusOK, buildConfigPayload(newCfg)))
	default:
		return okEnvelope(jsonResponse(http.StatusNotFound, map[string]string{"error": "not found"}))
	}
}

func queryInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func buildStatus(current *Service) statusPayload {
	now := time.Now()
	cfg, run, next := current.Status(now)
	return statusPayload{
		PluginID:        pluginID,
		Version:         pluginVersion,
		GeneratedAt:     now,
		Enabled:         cfg.ActivationEnabled,
		ActivationTimes: cfg.ActivationTimesText,
		Timezone:        cfg.ActivationTimezone,
		RequestsPerRun:  cfg.ActivationRequestsPerRun,
		RandomDelayMax:  cfg.ActivationRandomDelaySecond,
		Concurrency:     cfg.ActivationConcurrency,
		NextScheduledAt: next,
		Run:             run,
	}
}

func normalizeManagementPath(path string) string {
	for _, prefix := range []string{"/v0/management" + managementBasePath, "/v0/resource" + managementBasePath, managementBasePath} {
		if strings.HasPrefix(path, prefix) {
			trimmed := strings.TrimPrefix(path, prefix)
			if trimmed == "" {
				return "/"
			}
			return trimmed
		}
	}
	return path
}

func isResourcePath(path string) bool {
	return strings.HasPrefix(path, "/v0/resource"+managementBasePath)
}

func jsonResponse(status int, value any) pluginapi.ManagementResponse {
	raw, _ := json.Marshal(value)
	return pluginapi.ManagementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": []string{"application/json; charset=utf-8"}, "Cache-Control": []string{"no-store"}},
		Body:       raw,
	}
}

func htmlResponse(status int, body []byte) pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: status,
		Headers: http.Header{
			"Content-Type":                 []string{"text/html; charset=utf-8"},
			"Cache-Control":                []string{"no-store"},
			"Content-Security-Policy":      []string{"default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'self'; base-uri 'none'; form-action 'none'"},
			"Referrer-Policy":              []string{"no-referrer"},
			"X-Content-Type-Options":       []string{"nosniff"},
			"Cross-Origin-Resource-Policy": []string{"same-origin"},
		},
		Body: body,
	}
}

type configPayload struct {
	ActivationEnabled           bool   `json:"activation_enabled"`
	ActivationTimes             string `json:"activation_times"`
	ActivationTimezone          string `json:"activation_timezone"`
	ActivationModel             string `json:"activation_model"`
	ActivationRequestsPerRun    int    `json:"activation_requests_per_run"`
	ActivationRandomDelaySecond int    `json:"activation_random_delay_seconds"`
	ActivationConcurrency       int    `json:"activation_concurrency"`
}

func buildConfigPayload(cfg Config) configPayload {
	return configPayload{
		ActivationEnabled:           cfg.ActivationEnabled,
		ActivationTimes:             cfg.ActivationTimesText,
		ActivationTimezone:          cfg.ActivationTimezone,
		ActivationModel:             cfg.ActivationModel,
		ActivationRequestsPerRun:    cfg.ActivationRequestsPerRun,
		ActivationRandomDelaySecond: cfg.ActivationRandomDelaySecond,
		ActivationConcurrency:       cfg.ActivationConcurrency,
	}
}

func decodeConfigJSON(raw []byte, base Config) (Config, error) {
	var payload configPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Config{}, fmt.Errorf("无效的配置 JSON: %w", err)
	}
	times, text, err := parseActivationTimes(payload.ActivationTimes)
	if err != nil {
		return Config{}, err
	}
	timezone := strings.TrimSpace(payload.ActivationTimezone)
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Config{}, fmt.Errorf("时区无效，请输入有效的 IANA 时区（如 Asia/Shanghai）")
	}
	model := strings.TrimSpace(payload.ActivationModel)
	if model == "" {
		return Config{}, fmt.Errorf("唤醒模型不能为空")
	}
	if payload.ActivationRequestsPerRun < 1 || payload.ActivationRequestsPerRun > 10 {
		return Config{}, fmt.Errorf("单账号请求次数必须在 1 到 10 之间")
	}
	if payload.ActivationRandomDelaySecond < 0 || payload.ActivationRandomDelaySecond > 3600 {
		return Config{}, fmt.Errorf("随机错峰延迟必须在 0 到 3600 秒之间")
	}
	if payload.ActivationConcurrency < 1 || payload.ActivationConcurrency > 32 {
		return Config{}, fmt.Errorf("并发数必须在 1 到 32 之间")
	}
	cfg := base
	cfg.ActivationEnabled = payload.ActivationEnabled
	cfg.ActivationTimes = times
	cfg.ActivationTimesText = text
	cfg.ActivationTimezone = timezone
	cfg.ActivationModel = model
	cfg.ActivationRequestsPerRun = payload.ActivationRequestsPerRun
	cfg.ActivationRandomDelaySecond = payload.ActivationRandomDelaySecond
	cfg.ActivationConcurrency = payload.ActivationConcurrency
	return cfg, nil
}
