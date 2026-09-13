package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type fakeHost struct {
	mu          sync.Mutex
	auths       []pluginapi.HostAuthFileEntry
	documents   map[string]json.RawMessage
	response    pluginapi.HTTPResponse
	doDelay     time.Duration
	requests    int
	active      int
	maxActive   int
	requestsLog []pluginapi.HTTPRequest
}

func (host *fakeHost) ListAuths() ([]pluginapi.HostAuthFileEntry, error) {
	host.mu.Lock()
	defer host.mu.Unlock()
	return append([]pluginapi.HostAuthFileEntry(nil), host.auths...), nil
}

func (host *fakeHost) GetAuth(authIndex string) (pluginapi.HostAuthGetResponse, error) {
	host.mu.Lock()
	defer host.mu.Unlock()
	return pluginapi.HostAuthGetResponse{AuthIndex: authIndex, Name: authIndex + ".json", JSON: host.documents[authIndex]}, nil
}

func (host *fakeHost) Do(request pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
	host.mu.Lock()
	host.requests++
	host.requestsLog = append(host.requestsLog, request)
	host.active++
	if host.active > host.maxActive {
		host.maxActive = host.active
	}
	response := host.response
	delay := host.doDelay
	host.mu.Unlock()
	if delay > 0 {
		time.Sleep(delay)
	}
	host.mu.Lock()
	host.active--
	host.mu.Unlock()
	return response, nil
}

func newTestService(t *testing.T, host HostClient) *Service {
	t.Helper()
	dir := t.TempDir()
	cfg := defaultConfig()
	cfg.DataDir = dir
	cfg.ConfigPath = filepath.Join(dir, "config.json")
	cfg.LogsPath = filepath.Join(dir, "logs.json")
	cfg.ActivationEnabled = false
	cfg.ActivationRandomDelaySecond = 0
	service, err := NewService(cfg, host)
	if err != nil {
		t.Fatal(err)
	}
	service.Start()
	t.Cleanup(func() { service.Stop(false) })
	return service
}

func waitForRun(t *testing.T, service *Service) RunStatus {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, status, _ := service.Status(time.Now())
		if !status.Running && !status.CompletedAt.IsZero() {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("activation run did not complete")
	return RunStatus{}
}

func TestManualActivationExecutesEveryAccountRequest(t *testing.T) {
	host := &fakeHost{
		auths: []pluginapi.HostAuthFileEntry{
			{ID: "a", AuthIndex: "a-index", Provider: "codex", Email: "a@example.com"},
			{ID: "b", AuthIndex: "b-index", Provider: "codex", Email: "b@example.com"},
			{ID: "other", AuthIndex: "other-index", Provider: "gemini"},
		},
		documents: map[string]json.RawMessage{
			"a-index": json.RawMessage(`{"access_token":"token-a","account_id":"account-a"}`),
			"b-index": json.RawMessage(`{"access_token":"token-b","account_id":"account-b"}`),
		},
		response: pluginapi.HTTPResponse{StatusCode: http.StatusOK, Body: []byte(`{"id":"resp_test"}`)},
		doDelay:  20 * time.Millisecond,
	}
	service := newTestService(t, host)
	if _, err := service.StartManualActivation(); err != nil {
		t.Fatal(err)
	}
	status := waitForRun(t, service)
	if status.Successes != 4 || status.Failures != 0 {
		t.Fatalf("unexpected run status: %+v", status)
	}
	page := service.QueryLogs(LogQuery{Page: 1, PageSize: 20})
	if page.Total != 4 {
		t.Fatalf("expected 4 execution logs, got %d", page.Total)
	}
	host.mu.Lock()
	defer host.mu.Unlock()
	if host.requests != 4 {
		t.Fatalf("expected 4 upstream requests, got %d", host.requests)
	}
	if host.maxActive > 2 {
		t.Fatalf("concurrency limit exceeded: %d", host.maxActive)
	}
	for _, request := range host.requestsLog {
		var body codexActivationBody
		if err := json.Unmarshal(request.Body, &body); err != nil {
			t.Fatalf("invalid activation request body: %v", err)
		}
		if !body.Stream {
			t.Fatal("activation request must be streaming")
		}
		if got := request.Headers.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("activation request Accept = %q, want text/event-stream", got)
		}
	}
}

func TestUnavailableAccountIsLoggedWithoutRequest(t *testing.T) {
	host := &fakeHost{
		auths:     []pluginapi.HostAuthFileEntry{{ID: "disabled", Provider: "codex", Email: "disabled@example.com", Disabled: true}},
		documents: map[string]json.RawMessage{},
		response:  pluginapi.HTTPResponse{StatusCode: http.StatusOK, Body: []byte(`{"id":"resp_test"}`)},
	}
	service := newTestService(t, host)
	if _, err := service.StartManualActivation(); err != nil {
		t.Fatal(err)
	}
	status := waitForRun(t, service)
	if status.Skipped != 1 || status.Successes != 0 {
		t.Fatalf("unexpected run status: %+v", status)
	}
	page := service.QueryLogs(LogQuery{Page: 1, PageSize: 20, Result: "skipped"})
	if page.Total != 1 || page.Items[0].Detail != "账号已禁用" {
		t.Fatalf("unexpected skipped log: %+v", page)
	}
}

func TestPlanDelaysUsesUniqueValuesWhenRangeAllows(t *testing.T) {
	service := newTestService(t, &fakeHost{})
	delays := service.planDelays(10, 60)
	seen := make(map[int]bool)
	for _, delay := range delays {
		if delay < 0 || delay > 60 {
			t.Fatalf("delay out of range: %d", delay)
		}
		if seen[delay] {
			t.Fatalf("unexpected duplicate delay: %d", delay)
		}
		seen[delay] = true
	}
}

func TestNextScheduledAtUsesConfiguredTimezone(t *testing.T) {
	service := newTestService(t, &fakeHost{})
	updated := service.Config()
	updated.ActivationTimezone = "Asia/Shanghai"
	updated.ActivationTimes = []clockTime{{Hour: 8}, {Hour: 20}}
	updated.ActivationTimesText = "08:00,20:00"
	if err := service.UpdateConfig(updated); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 12, 9, 30, 0, 0, location)
	next, err := service.nextScheduledAt(now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 12, 20, 0, 0, 0, location)
	if !next.Equal(want) {
		t.Fatalf("next schedule = %s, want %s", next, want)
	}
}

func TestParseActivationAuthMaterial(t *testing.T) {
	material, err := parseActivationAuthMaterial(json.RawMessage(`{"tokens":{"access_token":"secret"},"account_id":"account"}`))
	if err != nil {
		t.Fatal(err)
	}
	if material.AccessToken != "secret" || material.AccountID != "account" {
		t.Fatalf("unexpected material: %+v", material)
	}
}

func TestEvaluateCodexActivationSuccess(t *testing.T) {
	if ok, message := evaluateCodexActivationSuccess(http.StatusOK, []byte(`{"id":"resp_test"}`)); !ok || message != "" {
		t.Fatalf("expected success, got ok=%v message=%q", ok, message)
	}
	if ok, _ := evaluateCodexActivationSuccess(http.StatusTooManyRequests, []byte(`{"error":{}}`)); ok {
		t.Fatal("expected non-2xx response to fail")
	}
}
