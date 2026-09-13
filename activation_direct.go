package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	codexActivationURL          = "https://chatgpt.com/backend-api/codex/responses"
	activationUnauthorizedTries = 3
	activationUnauthorizedDelay = 50 * time.Millisecond
)

type activationAuthMaterial struct {
	AccessToken string
	AccountID   string
}

type codexActivationBody struct {
	Model        string                   `json:"model"`
	Instructions string                   `json:"instructions"`
	Input        []codexActivationMessage `json:"input"`
	Store        bool                     `json:"store"`
	Stream       bool                     `json:"stream"`
}

type codexActivationMessage struct {
	Type    string                 `json:"type"`
	Role    string                 `json:"role"`
	Content []codexActivationInput `json:"content"`
}

type codexActivationInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Service) activateAccount(auth pluginapi.HostAuthFileEntry, settings ActivationSettings) (string, error) {
	material, err := s.activationAuthMaterial(auth)
	if err != nil {
		return settings.Model, err
	}
	body, err := json.Marshal(codexActivationBody{
		Model:        settings.Model,
		Instructions: "You are a helpful assistant.",
		Input: []codexActivationMessage{{
			Type: "message",
			Role: "user",
			Content: []codexActivationInput{{
				Type: "input_text",
				Text: settings.Prompt,
			}},
		}},
		Store:  false,
		Stream: true,
	})
	if err != nil {
		return settings.Model, fmt.Errorf("编码唤醒请求失败: %w", err)
	}
	headers := http.Header{
		"Accept":        []string{"text/event-stream"},
		"Authorization": []string{"Bearer " + material.AccessToken},
		"Content-Type":  []string{"application/json"},
		"OpenAI-Beta":   []string{"responses=v1"},
		"Originator":    []string{"codex_cli_rs"},
		"User-Agent":    []string{fmt.Sprintf("codex_cli_rs/0.76.0 (%s; %s)", runtime.GOOS, runtime.GOARCH)},
	}
	if material.AccountID != "" {
		headers.Set("Chatgpt-Account-Id", material.AccountID)
	}
	request := pluginapi.HTTPRequest{
		Method:  http.MethodPost,
		URL:     codexActivationURL,
		Headers: headers,
		Body:    body,
	}

	var response pluginapi.HTTPResponse
	for attempt := 1; attempt <= activationUnauthorizedTries; attempt++ {
		response, err = s.host.Do(request)
		if err != nil {
			return settings.Model, fmt.Errorf("唤醒请求失败: %w", err)
		}
		if response.StatusCode != http.StatusUnauthorized {
			break
		}
		if attempt < activationUnauthorizedTries {
			select {
			case <-s.Context().Done():
				return settings.Model, s.Context().Err()
			case <-time.After(activationUnauthorizedDelay):
			}
		}
	}
	if ok, message := evaluateCodexActivationSuccess(response.StatusCode, response.Body); !ok {
		return settings.Model, errors.New(message)
	}
	return settings.Model, nil
}

func (s *Service) activationAuthMaterial(auth pluginapi.HostAuthFileEntry) (activationAuthMaterial, error) {
	if !strings.EqualFold(auth.Provider, "codex") || strings.TrimSpace(auth.ID) == "" {
		return activationAuthMaterial{}, errors.New("目标 Codex 账号不存在")
	}
	if auth.Disabled || auth.Unavailable {
		return activationAuthMaterial{}, errors.New("目标 Codex 账号当前不可用")
	}
	if strings.TrimSpace(auth.AuthIndex) == "" {
		return activationAuthMaterial{}, errors.New("目标 Codex 账号缺少凭据索引")
	}
	response, err := s.host.GetAuth(auth.AuthIndex)
	if err != nil {
		return activationAuthMaterial{}, fmt.Errorf("读取目标账号凭据失败: %w", err)
	}
	material, err := parseActivationAuthMaterial(response.JSON)
	if err != nil {
		return activationAuthMaterial{}, err
	}
	return material, nil
}

func parseActivationAuthMaterial(raw json.RawMessage) (activationAuthMaterial, error) {
	var root map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &root) != nil {
		return activationAuthMaterial{}, errors.New("目标 Codex 账号凭据无效")
	}
	material := activationAuthMaterial{
		AccessToken: firstActivationString(root, "access_token", "accessToken", "oauth_access_token", "oauthAccessToken", "token", "id_token", "idToken"),
		AccountID:   firstActivationString(root, "account_id", "chatgpt_account_id", "accountId", "chatgptAccountId"),
	}
	for _, key := range []string{"tokens", "credentials", "auth", "oauth", "session"} {
		rawNested, ok := root[key]
		if !ok {
			continue
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(rawNested, &nested) != nil {
			continue
		}
		if material.AccessToken == "" {
			material.AccessToken = firstActivationString(nested, "access_token", "accessToken", "oauth_access_token", "oauthAccessToken", "token", "id_token", "idToken")
		}
		if material.AccountID == "" {
			material.AccountID = firstActivationString(nested, "account_id", "chatgpt_account_id", "accountId", "chatgptAccountId")
		}
	}
	if material.AccessToken == "" {
		return activationAuthMaterial{}, errors.New("目标 Codex 账号凭据缺少访问令牌")
	}
	return material, nil
}

func firstActivationString(document map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		var value string
		if raw, ok := document[key]; ok && json.Unmarshal(raw, &value) == nil {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}

func evaluateCodexActivationSuccess(statusCode int, body []byte) (bool, string) {
	if statusCode < 200 || statusCode >= 300 {
		return false, fmt.Sprintf("Codex 唤醒失败：上游返回 HTTP %d", statusCode)
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return false, "Codex 唤醒失败：响应体为空"
	}
	if strings.Contains(trimmed, "event:") || strings.Contains(trimmed, "data:") {
		return evaluateCodexActivationSSE(trimmed)
	}
	var response map[string]json.RawMessage
	if json.Unmarshal(body, &response) != nil {
		return false, "Codex 唤醒失败：响应不是合法数据"
	}
	return evaluateCodexActivationObject(response)
}

func evaluateCodexActivationSSE(body string) (bool, string) {
	sawCompleted := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var event map[string]json.RawMessage
		if json.Unmarshal([]byte(payload), &event) != nil {
			continue
		}
		var eventType string
		_ = json.Unmarshal(event["type"], &eventType)
		if eventType == "response.failed" || eventType == "error" {
			return false, codexActivationErrorMessage(event)
		}
		if eventType == "response.completed" {
			sawCompleted = true
		}
		if rawResponse, ok := event["response"]; ok {
			var response map[string]json.RawMessage
			if json.Unmarshal(rawResponse, &response) == nil {
				if ok, message := evaluateCodexActivationObject(response); ok {
					return true, ""
				} else if message != "Codex 唤醒失败：响应缺少有效输出结构" {
					return false, message
				}
			}
		}
	}
	if sawCompleted || (strings.Contains(body, `"type":"response.created"`) && strings.Contains(body, `"id":"resp_`)) {
		return true, ""
	}
	return false, "Codex 唤醒失败：响应缺少有效输出结构"
}

func evaluateCodexActivationObject(response map[string]json.RawMessage) (bool, string) {
	if rawError, ok := response["error"]; ok && string(rawError) != "null" {
		return false, codexActivationErrorMessage(response)
	}
	var id string
	if json.Unmarshal(response["id"], &id) == nil && strings.TrimSpace(id) != "" {
		return true, ""
	}
	var output []json.RawMessage
	if json.Unmarshal(response["output"], &output) == nil && len(output) > 0 {
		return true, ""
	}
	if rawNested, ok := response["response"]; ok {
		var nested map[string]json.RawMessage
		if json.Unmarshal(rawNested, &nested) == nil {
			return evaluateCodexActivationObject(nested)
		}
	}
	return false, "Codex 唤醒失败：响应缺少有效输出结构"
}

func codexActivationErrorMessage(response map[string]json.RawMessage) string {
	rawError := response["error"]
	if rawResponse, ok := response["response"]; ok {
		var nested map[string]json.RawMessage
		if json.Unmarshal(rawResponse, &nested) == nil {
			if nestedError, exists := nested["error"]; exists {
				rawError = nestedError
			}
		}
	}
	var payload struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(rawError, &payload)
	combined := strings.ToLower(payload.Type + " " + payload.Code + " " + payload.Message)
	if strings.Contains(combined, "usage_limit") || strings.Contains(combined, "usage limit") {
		return "Codex 唤醒失败：用量额度已耗尽"
	}
	if strings.Contains(combined, "server_is_overloaded") || strings.Contains(combined, "server is overloaded") {
		return "Codex 唤醒失败：server_is_overloaded"
	}
	return "Codex 唤醒失败：上游返回业务错误"
}
