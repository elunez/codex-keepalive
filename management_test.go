package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func installCurrentService(t *testing.T, current *Service) {
	t.Helper()
	serviceMu.Lock()
	previous := service
	service = current
	serviceMu.Unlock()
	t.Cleanup(func() {
		serviceMu.Lock()
		service = previous
		serviceMu.Unlock()
	})
}

func callManagement(t *testing.T, request pluginapi.ManagementRequest) pluginapi.ManagementResponse {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := handleManagement(raw)
	if err != nil {
		t.Fatal(err)
	}
	var result envelope
	if err := json.Unmarshal(wrapped, &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("unexpected envelope error: %+v", result.Error)
	}
	var response pluginapi.ManagementResponse
	if err := json.Unmarshal(result.Result, &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestManagementStatusAndPaginatedLogs(t *testing.T) {
	current := newTestService(t, &fakeHost{})
	installCurrentService(t, current)
	base := time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)
	for index := 0; index < 25; index++ {
		current.appendLog(ExecutionLog{
			RunID: "manual:test", ExecutedAt: base.Add(time.Duration(index) * time.Second),
			Account: "user@example.com", Trigger: "manual", RequestIndex: 1, RequestTotal: 2,
			Status: "success", Detail: "唤醒请求已发送",
		})
	}

	statusResponse := callManagement(t, pluginapi.ManagementRequest{Method: http.MethodGet, Path: managementBasePath + "/status"})
	if statusResponse.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", statusResponse.StatusCode)
	}
	var status statusPayload
	if err := json.Unmarshal(statusResponse.Body, &status); err != nil {
		t.Fatal(err)
	}
	if status.Version != pluginVersion || status.RequestsPerRun != 2 {
		t.Fatalf("unexpected status payload: %+v", status)
	}

	logsResponse := callManagement(t, pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   managementBasePath + "/logs",
		Query:  url.Values{"page": {"2"}, "page_size": {"10"}, "trigger": {"manual"}},
	})
	var page LogPage
	if err := json.Unmarshal(logsResponse.Body, &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 25 || page.TotalPages != 3 || page.Page != 2 || len(page.Items) != 10 {
		t.Fatalf("unexpected log page: %+v", page)
	}
}

func TestManagementClearLogs(t *testing.T) {
	current := newTestService(t, &fakeHost{})
	installCurrentService(t, current)
	current.appendLog(ExecutionLog{RunID: "test", ExecutedAt: time.Now(), Account: "a", Trigger: "manual", Status: "success"})
	response := callManagement(t, pluginapi.ManagementRequest{Method: http.MethodDelete, Path: managementBasePath + "/logs"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}
	if page := current.QueryLogs(LogQuery{Page: 1, PageSize: 20}); page.Total != 0 {
		t.Fatalf("expected logs to be empty, got %d", page.Total)
	}
}

func TestManagementConfigGetAndPut(t *testing.T) {
	current := newTestService(t, &fakeHost{})
	installCurrentService(t, current)

	getResponse := callManagement(t, pluginapi.ManagementRequest{Method: http.MethodGet, Path: managementBasePath + "/config"})
	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("unexpected GET /config status: %d", getResponse.StatusCode)
	}
	var currentCfg configPayload
	if err := json.Unmarshal(getResponse.Body, &currentCfg); err != nil {
		t.Fatal(err)
	}
	if currentCfg.ActivationTimes != "08:00,20:00" || currentCfg.ActivationRequestsPerRun != 2 {
		t.Fatalf("unexpected current config: %+v", currentCfg)
	}

	updatedPayload := configPayload{
		ActivationEnabled:           false,
		ActivationTimes:             "09:30,21:30",
		ActivationTimezone:          "Asia/Shanghai",
		ActivationModel:             "gpt-5.5",
		ActivationRequestsPerRun:    3,
		ActivationRandomDelaySecond: 45,
		ActivationConcurrency:       4,
	}
	rawUpdate, _ := json.Marshal(updatedPayload)
	putResponse := callManagement(t, pluginapi.ManagementRequest{
		Method: http.MethodPut,
		Path:   managementBasePath + "/config",
		Body:   rawUpdate,
	})
	if putResponse.StatusCode != http.StatusOK {
		t.Fatalf("unexpected PUT /config status: %d, body: %s", putResponse.StatusCode, string(putResponse.Body))
	}

	statusResponse := callManagement(t, pluginapi.ManagementRequest{Method: http.MethodGet, Path: managementBasePath + "/status"})
	var status statusPayload
	if err := json.Unmarshal(statusResponse.Body, &status); err != nil {
		t.Fatal(err)
	}
	if status.Enabled != false || status.ActivationTimes != "09:30,21:30" || status.RequestsPerRun != 3 || status.Concurrency != 4 {
		t.Fatalf("status did not reflect updated config: %+v", status)
	}
}

func TestManagementConfigGetReloadsPersistedConfig(t *testing.T) {
	current := newTestService(t, &fakeHost{})
	installCurrentService(t, current)
	persisted := current.Config()
	persisted.ActivationTimesText = "02:15,08:15,14:15,20:15"
	persisted.ActivationTimes, _, _ = parseActivationTimes(persisted.ActivationTimesText)
	persisted.ActivationModel = "gpt-4.1"
	if err := saveConfigFile(persisted.ConfigPath, persisted); err != nil {
		t.Fatal(err)
	}

	response := callManagement(t, pluginapi.ManagementRequest{Method: http.MethodGet, Path: managementBasePath + "/config"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected GET /config status: %d", response.StatusCode)
	}
	var payload configPayload
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ActivationTimes != "02:15,08:15,14:15,20:15" || payload.ActivationModel != "gpt-4.1" {
		t.Fatalf("persisted config was not reloaded for UI: %+v", payload)
	}
	if cfg := current.Config(); cfg.ActivationTimesText != payload.ActivationTimes || cfg.ActivationModel != payload.ActivationModel {
		t.Fatalf("service did not adopt reloaded config: %+v", cfg)
	}
}

func TestManagementStatusReloadsPersistedConfig(t *testing.T) {
	current := newTestService(t, &fakeHost{})
	installCurrentService(t, current)
	persisted := current.Config()
	persisted.ActivationEnabled = false
	persisted.ActivationTimesText = "02:15,08:15,14:15,20:15"
	persisted.ActivationTimes, _, _ = parseActivationTimes(persisted.ActivationTimesText)
	if err := saveConfigFile(persisted.ConfigPath, persisted); err != nil {
		t.Fatal(err)
	}

	response := callManagement(t, pluginapi.ManagementRequest{Method: http.MethodGet, Path: managementBasePath + "/status"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected GET /status status: %d", response.StatusCode)
	}
	var payload statusPayload
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Enabled || payload.ActivationTimes != persisted.ActivationTimesText {
		t.Fatalf("persisted config was not reflected by status: %+v", payload)
	}
}

func TestManagementPageMatchesSelectedActivationOnlyUI(t *testing.T) {
	checks := []string{"Codex 定时唤醒", "data-tooltip=\"立即执行\"", "data-tooltip=\"插件设置\"", "settings-modal", "save-settings", "执行日志", "序号", "page-size", "随机延迟", "max-height:377px", "overflow:auto", "nth-child(4){width:10%}", "nth-child(8){width:18%}", "02:15,08:15,14:15,20:15"}
	for _, expected := range checks {
		if !strings.Contains(statusPageHTML, expected) {
			t.Fatalf("page missing %q", expected)
		}
	}
	for _, removed := range []string{"读取额度", "调度规则", "keeper_db_path", ">配置<"} {
		if strings.Contains(statusPageHTML, removed) {
			t.Fatalf("page still contains removed UI %q", removed)
		}
	}
}

func TestLegacySchedulerCallAlwaysFallsThrough(t *testing.T) {
	raw, err := handleMethod("scheduler.pick", nil)
	if err != nil {
		t.Fatal(err)
	}
	var result envelope
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	var response pluginapi.SchedulerPickResponse
	if err := json.Unmarshal(result.Result, &response); err != nil {
		t.Fatal(err)
	}
	if response.Handled {
		t.Fatal("scheduler call must not be handled")
	}
}
