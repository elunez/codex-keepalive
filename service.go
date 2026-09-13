package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const maxPersistedLogs = 5000

type HostClient interface {
	ListAuths() ([]pluginapi.HostAuthFileEntry, error)
	GetAuth(authIndex string) (pluginapi.HostAuthGetResponse, error)
	Do(request pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error)
}

type ExecutionLog struct {
	ID           string    `json:"id"`
	RunID        string    `json:"run_id"`
	ExecutedAt   time.Time `json:"executed_at"`
	Account      string    `json:"account"`
	AuthID       string    `json:"auth_id,omitempty"`
	Trigger      string    `json:"trigger"`
	RequestIndex int       `json:"request_index"`
	RequestTotal int       `json:"request_total"`
	DelaySeconds int       `json:"delay_seconds"`
	Status       string    `json:"status"`
	Detail       string    `json:"detail"`
	Model        string    `json:"model,omitempty"`
}

// persistedConfig 是用于跨插件重新安装保留配置的稳定数据结构。
// 不直接序列化 Config，避免把内部的 ActivationTimes 表示暴露为持久化格式。
// 这里不包含任何账号凭据、token 或管理密钥。
type persistedConfig struct {
	ActivationEnabled           bool   `json:"activation_enabled"`
	ActivationTimesText         string `json:"activation_times"`
	ActivationTimezone          string `json:"activation_timezone"`
	ActivationModel             string `json:"activation_model"`
	ActivationRequestsPerRun    int    `json:"activation_requests_per_run"`
	ActivationRandomDelaySecond int    `json:"activation_random_delay_seconds"`
	ActivationConcurrency       int    `json:"activation_concurrency"`
}

func persistedConfigFromConfig(cfg Config) persistedConfig {
	return persistedConfig{
		ActivationEnabled:           cfg.ActivationEnabled,
		ActivationTimesText:         cfg.ActivationTimesText,
		ActivationTimezone:          cfg.ActivationTimezone,
		ActivationModel:             cfg.ActivationModel,
		ActivationRequestsPerRun:    cfg.ActivationRequestsPerRun,
		ActivationRandomDelaySecond: cfg.ActivationRandomDelaySecond,
		ActivationConcurrency:       cfg.ActivationConcurrency,
	}
}

func (value persistedConfig) toConfig(base Config) (Config, error) {
	times, text, err := parseActivationTimes(value.ActivationTimesText)
	if err != nil {
		return Config{}, fmt.Errorf("decode persisted activation_times: %w", err)
	}
	if _, err := time.LoadLocation(value.ActivationTimezone); err != nil {
		return Config{}, fmt.Errorf("decode persisted activation_timezone: %w", err)
	}
	if strings.TrimSpace(value.ActivationModel) == "" {
		return Config{}, errors.New("decode persisted activation_model: model must not be empty")
	}
	if value.ActivationRequestsPerRun < 1 || value.ActivationRequestsPerRun > 10 {
		return Config{}, errors.New("decode persisted activation_requests_per_run: value must be between 1 and 10")
	}
	if value.ActivationRandomDelaySecond < 0 || value.ActivationRandomDelaySecond > 3600 {
		return Config{}, errors.New("decode persisted activation_random_delay_seconds: value must be between 0 and 3600")
	}
	if value.ActivationConcurrency < 1 || value.ActivationConcurrency > 32 {
		return Config{}, errors.New("decode persisted activation_concurrency: value must be between 1 and 32")
	}
	cfg := base
	cfg.ActivationEnabled = value.ActivationEnabled
	cfg.ActivationTimes = times
	cfg.ActivationTimesText = text
	cfg.ActivationTimezone = value.ActivationTimezone
	cfg.ActivationModel = value.ActivationModel
	cfg.ActivationRequestsPerRun = value.ActivationRequestsPerRun
	cfg.ActivationRandomDelaySecond = value.ActivationRandomDelaySecond
	cfg.ActivationConcurrency = value.ActivationConcurrency
	return cfg, nil
}

type RunStatus struct {
	Running     bool      `json:"running"`
	RunID       string    `json:"run_id,omitempty"`
	Trigger     string    `json:"trigger,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Successes   int       `json:"successes"`
	Failures    int       `json:"failures"`
	Skipped     int       `json:"skipped"`
	LastError   string    `json:"last_error,omitempty"`
}

type LogQuery struct {
	Page     int
	PageSize int
	Account  string
	Result   string
	Trigger  string
}

type LogPage struct {
	Items      []ExecutionLog `json:"items"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	Total      int            `json:"total"`
	TotalPages int            `json:"total_pages"`
}

type activationJob struct {
	Auth         pluginapi.HostAuthFileEntry
	Account      string
	RequestIndex int
	DelaySeconds int
}

type Service struct {
	cfg  Config
	host HostClient

	mu     sync.RWMutex
	logs   []ExecutionLog
	status RunStatus

	ctx        context.Context
	stop       context.CancelFunc
	done       chan struct{}
	reschedule chan struct{}
	workers    sync.WaitGroup

	randomMu sync.Mutex
	random   *rand.Rand
	logSeq   atomic.Uint64
	runSeq   atomic.Uint64

	sleep func(context.Context, time.Duration) error
}

func NewService(cfg Config, host HostClient) (*Service, error) {
	if cfg.ConfigPath == "" || cfg.LogsPath == "" {
		dir, cfgPath, logPath := resolvePaths(cfg.DataDir)
		cfg.DataDir = dir
		cfg.ConfigPath = cfgPath
		cfg.LogsPath = logPath
	}
	// 业务配置由插件页面维护。宿主在重新安装/重载时通常只会传入
	// enabled、priority 等通用字段，因此初始化阶段必须以 config.json
	// 中的快照作为最终业务配置来源，避免默认值覆盖已保存设置。
	if persisted, err := loadConfigFile(cfg.ConfigPath); err != nil {
		return nil, err
	} else if persisted != nil {
		// 配置文件属于可恢复的持久化数据。升级过程中如果文件来自
		// 不兼容的旧格式或包含无效值，使用宿主本次传入的配置继续
		// 注册，避免非关键配置阻断整个插件启动。
		if restored, restoreErr := persisted.toConfig(cfg); restoreErr == nil {
			cfg = restored
		}
	}
	logs, err := loadLogsFile(cfg.LogsPath)
	if err != nil {
		return nil, err
	}
	if err := saveConfigFile(cfg.ConfigPath, cfg); err != nil {
		return nil, err
	}
	return &Service{
		cfg:        cfg,
		host:       host,
		logs:       logs,
		done:       make(chan struct{}),
		reschedule: make(chan struct{}, 1),
		random:     rand.New(rand.NewSource(time.Now().UnixNano())),
		sleep:      sleepContext,
	}, nil
}

func loadConfigFile(path string) (*persistedConfig, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg persistedConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}
	return &cfg, nil
}

func saveConfigFile(path string, cfg Config) error {
	raw, err := json.MarshalIndent(persistedConfigFromConfig(cfg), "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, raw)
}

func loadLogsFile(path string) ([]ExecutionLog, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read logs file: %w", err)
	}
	var logs []ExecutionLog
	if err := json.Unmarshal(raw, &logs); err != nil {
		// 日志是非关键运行数据。旧版本升级、容器异常退出或手工清空
		// 文件时可能留下空文件/不完整 JSON；不能因此阻断插件注册。
		// 下一次产生日志时会通过原子写入覆盖该文件。
		return nil, nil
	}
	if len(logs) > maxPersistedLogs {
		logs = append([]ExecutionLog(nil), logs[len(logs)-maxPersistedLogs:]...)
	}
	return logs, nil
}

func saveLogsFile(path string, logs []ExecutionLog) error {
	if logs == nil {
		logs = []ExecutionLog{}
	}
	raw, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, raw)
}

func atomicWriteFile(destination string, data []byte) error {
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	pattern := fmt.Sprintf(".%s-*.tmp", filepath.Base(destination))
	temporary, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempName := temporary.Name()
	defer os.Remove(tempName)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, destination); err != nil {
		return fmt.Errorf("replace file %s: %w", destination, err)
	}
	return nil
}

func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.ctx = ctx
	s.stop = cancel
	s.mu.Unlock()
	go s.runScheduler(ctx)
}

func (s *Service) Context() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *Service) configSnapshot() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Service) Stop(_ bool) {
	if s == nil {
		return
	}
	s.mu.RLock()
	cancel := s.stop
	s.mu.RUnlock()
	if cancel == nil {
		return
	}
	cancel()
	<-s.done
	s.workers.Wait()
}

func (s *Service) runScheduler(ctx context.Context) {
	defer close(s.done)
	for {
		cfg := s.configSnapshot()
		enabled := cfg.ActivationEnabled
		if !enabled {
			select {
			case <-ctx.Done():
				return
			case <-s.reschedule:
				continue
			}
		}
		next, err := nextScheduledAtForConfig(cfg, time.Now())
		if err != nil {
			s.setRunError(err.Error())
			select {
			case <-ctx.Done():
				return
			case <-s.reschedule:
				continue
			}
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-s.reschedule:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			continue
		case <-timer.C:
			_, _ = s.startActivation("scheduled")
		}
	}
}

func (s *Service) nextScheduledAt(now time.Time) (time.Time, error) {
	return nextScheduledAtForConfig(s.configSnapshot(), now)
}

func nextScheduledAtForConfig(cfg Config, now time.Time) (time.Time, error) {
	location, err := time.LoadLocation(cfg.ActivationTimezone)
	if err != nil {
		return time.Time{}, err
	}
	localNow := now.In(location)
	for dayOffset := 0; dayOffset <= 1; dayOffset++ {
		day := localNow.AddDate(0, 0, dayOffset)
		for _, configured := range cfg.ActivationTimes {
			candidate := time.Date(day.Year(), day.Month(), day.Day(), configured.Hour, configured.Minute, 0, 0, location)
			if candidate.After(localNow) {
				return candidate, nil
			}
		}
	}
	return time.Time{}, errors.New("no activation time configured")
}

func (s *Service) StartManualActivation() (string, error) {
	return s.startActivation("manual")
}

func (s *Service) startActivation(trigger string) (string, error) {
	if trigger != "scheduled" && trigger != "manual" {
		return "", errors.New("invalid activation trigger")
	}
	now := time.Now()
	runID := fmt.Sprintf("%s:%d:%d", trigger, now.UnixNano(), s.runSeq.Add(1))
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return "", errors.New("唤醒任务正在执行，请稍后再试")
	}
	s.status = RunStatus{Running: true, RunID: runID, Trigger: trigger, StartedAt: now}
	s.mu.Unlock()

	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		s.runActivationRound(s.Context(), runID, trigger)
	}()
	return runID, nil
}

func (s *Service) runActivationRound(ctx context.Context, runID, trigger string) {
	cfg := s.configSnapshot()
	auths, err := s.host.ListAuths()
	if err != nil {
		s.appendLog(ExecutionLog{
			RunID: runID, ExecutedAt: time.Now(), Account: "系统", Trigger: trigger,
			Status: "failed", Detail: "读取 Codex 账号失败: " + err.Error(),
		})
		s.finishRun(runID, 0, 1, 0, err.Error())
		return
	}
	sort.Slice(auths, func(i, j int) bool {
		return strings.ToLower(accountDisplayName(auths[i])) < strings.ToLower(accountDisplayName(auths[j]))
	})

	jobs := make([]activationJob, 0)
	skipped := 0
	found := 0
	for _, auth := range auths {
		if !strings.EqualFold(auth.Provider, "codex") || strings.TrimSpace(auth.ID) == "" {
			continue
		}
		found++
		account := accountDisplayName(auth)
		if reason := unavailableReason(auth); reason != "" {
			skipped++
			s.appendLog(ExecutionLog{
				RunID: runID, ExecutedAt: time.Now(), Account: account, AuthID: auth.ID,
				Trigger: trigger, Status: "skipped", Detail: reason,
			})
			continue
		}
		for requestIndex := 1; requestIndex <= cfg.ActivationRequestsPerRun; requestIndex++ {
			jobs = append(jobs, activationJob{Auth: auth, Account: account, RequestIndex: requestIndex})
		}
	}
	if found == 0 {
		skipped++
		s.appendLog(ExecutionLog{
			RunID: runID, ExecutedAt: time.Now(), Account: "—", Trigger: trigger,
			Status: "skipped", Detail: "未发现 Codex 账号",
		})
	}

	delays := s.planDelays(len(jobs), cfg.ActivationRandomDelaySecond)
	requestsPerAccount := cfg.ActivationRequestsPerRun
	for start := 0; start < len(jobs); start += requestsPerAccount {
		end := start + requestsPerAccount
		if end > len(jobs) {
			end = len(jobs)
		}
		sort.Ints(delays[start:end])
		for index := start; index < end; index++ {
			jobs[index].DelaySeconds = delays[index]
		}
	}
	semaphore := make(chan struct{}, cfg.ActivationConcurrency)
	var group sync.WaitGroup
	var successes atomic.Int64
	var failures atomic.Int64
	for _, job := range jobs {
		job := job
		group.Add(1)
		go func() {
			defer group.Done()
			if err := s.sleep(ctx, time.Duration(job.DelaySeconds)*time.Second); err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
			}
			defer func() { <-semaphore }()

			executedAt := time.Now()
			model, activationErr := s.activateAccount(job.Auth, activationSettingsFromConfig(cfg))
			status := "success"
			detail := "唤醒请求已发送"
			if activationErr != nil {
				status = "failed"
				detail = activationErr.Error()
				failures.Add(1)
			} else {
				successes.Add(1)
			}
			s.appendLog(ExecutionLog{
				RunID: runID, ExecutedAt: executedAt, Account: job.Account, AuthID: job.Auth.ID,
				Trigger: trigger, RequestIndex: job.RequestIndex, RequestTotal: cfg.ActivationRequestsPerRun,
				DelaySeconds: job.DelaySeconds, Status: status, Detail: detail, Model: model,
			})
		}()
	}
	group.Wait()
	lastError := ""
	if ctx.Err() != nil {
		lastError = "任务已取消"
	}
	s.finishRun(runID, int(successes.Load()), int(failures.Load()), skipped, lastError)
}

func unavailableReason(auth pluginapi.HostAuthFileEntry) string {
	switch {
	case auth.Disabled:
		return "账号已禁用"
	case auth.Unavailable:
		return "账号当前不可用"
	case strings.TrimSpace(auth.AuthIndex) == "":
		return "账号缺少凭据索引"
	default:
		return ""
	}
}

func accountDisplayName(auth pluginapi.HostAuthFileEntry) string {
	for _, value := range []string{auth.Email, auth.Label, auth.Name, auth.ID} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "未命名账号"
}

func (s *Service) planDelays(count, maximum int) []int {
	delays := make([]int, count)
	if count == 0 || maximum <= 0 {
		return delays
	}
	s.randomMu.Lock()
	defer s.randomMu.Unlock()
	if count <= maximum+1 {
		permutation := s.random.Perm(maximum + 1)
		copy(delays, permutation[:count])
		return delays
	}
	for index := range delays {
		delays[index] = s.random.Intn(maximum + 1)
	}
	return delays
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Service) finishRun(runID string, successes, failures, skipped int, lastError string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.RunID != runID {
		return
	}
	s.status.Running = false
	s.status.CompletedAt = time.Now()
	s.status.Successes = successes
	s.status.Failures = failures
	s.status.Skipped = skipped
	s.status.LastError = lastError
}

func (s *Service) setRunError(message string) {
	s.mu.Lock()
	s.status.LastError = message
	s.mu.Unlock()
}

func (s *Service) Config() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// ReloadPersistedConfig refreshes the effective activation settings from
// config.json. It is used by the management page so a plugin reinstall or
// host reload immediately reflects the durable configuration in both the UI
// and the scheduler.
func (s *Service) ReloadPersistedConfig() error {
	s.mu.RLock()
	path := s.cfg.ConfigPath
	base := s.cfg
	s.mu.RUnlock()
	persisted, err := loadConfigFile(path)
	if err != nil {
		return err
	}
	if persisted == nil {
		return nil
	}
	restored, err := persisted.toConfig(base)
	if err != nil {
		// 与启动阶段保持一致：语义无效的持久化配置不阻断服务，
		// 用当前有效配置修复文件，避免管理页面反复返回 500。
		s.mu.Lock()
		saveErr := s.persistConfigLocked()
		s.mu.Unlock()
		if saveErr != nil {
			return fmt.Errorf("修复持久化配置失败: %w", saveErr)
		}
		return nil
	}
	s.mu.Lock()
	s.cfg = restored
	s.mu.Unlock()
	if s.reschedule != nil {
		select {
		case s.reschedule <- struct{}{}:
		default:
		}
	}
	return nil
}

func (s *Service) persistConfigLocked() error {
	return saveConfigFile(s.cfg.ConfigPath, s.cfg)
}

func (s *Service) persistLogsLocked() error {
	return saveLogsFile(s.cfg.LogsPath, s.logs)
}

func (s *Service) UpdateConfig(newCfg Config) error {
	s.mu.Lock()
	err := saveConfigFile(newCfg.ConfigPath, newCfg)
	if err == nil {
		s.cfg = newCfg
	}
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	if s.reschedule != nil {
		select {
		case s.reschedule <- struct{}{}:
		default:
		}
	}
	return nil
}

func (s *Service) Status(now time.Time) (Config, RunStatus, time.Time) {
	cfg := s.configSnapshot()
	s.mu.RLock()
	status := s.status
	s.mu.RUnlock()
	var next time.Time
	if cfg.ActivationEnabled {
		next, _ = nextScheduledAtForConfig(cfg, now)
	}
	return cfg, status, next
}

func (s *Service) appendLog(entry ExecutionLog) {
	entry.ID = fmt.Sprintf("%s:%d", entry.RunID, s.logSeq.Add(1))
	s.mu.Lock()
	s.logs = append(s.logs, entry)
	if len(s.logs) > maxPersistedLogs {
		s.logs = append([]ExecutionLog(nil), s.logs[len(s.logs)-maxPersistedLogs:]...)
	}
	if err := s.persistLogsLocked(); err != nil {
		s.status.LastError = "保存执行日志失败: " + err.Error()
	}
	s.mu.Unlock()
}

func (s *Service) QueryLogs(query LogQuery) LogPage {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	accountFilter := strings.ToLower(strings.TrimSpace(query.Account))
	resultFilter := strings.ToLower(strings.TrimSpace(query.Result))
	triggerFilter := strings.ToLower(strings.TrimSpace(query.Trigger))

	s.mu.RLock()
	filtered := make([]ExecutionLog, 0, len(s.logs))
	for _, entry := range s.logs {
		if accountFilter != "" && !strings.Contains(strings.ToLower(entry.Account), accountFilter) {
			continue
		}
		if resultFilter != "" && resultFilter != "all" && entry.Status != resultFilter {
			continue
		}
		if triggerFilter != "" && triggerFilter != "all" && entry.Trigger != triggerFilter {
			continue
		}
		filtered = append(filtered, entry)
	}
	s.mu.RUnlock()
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].ExecutedAt.Equal(filtered[j].ExecutedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].ExecutedAt.After(filtered[j].ExecutedAt)
	})

	total := len(filtered)
	totalPages := 0
	if total > 0 {
		totalPages = (total + query.PageSize - 1) / query.PageSize
		if query.Page > totalPages {
			query.Page = totalPages
		}
	} else {
		query.Page = 1
	}
	start := (query.Page - 1) * query.PageSize
	if start > total {
		start = total
	}
	end := start + query.PageSize
	if end > total {
		end = total
	}
	items := append([]ExecutionLog(nil), filtered[start:end]...)
	return LogPage{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total, TotalPages: totalPages}
}

func (s *Service) ClearLogs() error {
	s.mu.Lock()
	previous := s.logs
	s.logs = nil
	if err := s.persistLogsLocked(); err != nil {
		s.logs = previous
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) refreshLogsAndPersistConfig() error {
	cfg := s.configSnapshot()
	logs, err := loadLogsFile(cfg.LogsPath)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.logs = logs
	err = s.persistConfigLocked()
	s.mu.Unlock()
	return err
}
