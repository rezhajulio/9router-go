package executor

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"9router/proxy/internal/proxy"
)

const (
	freebuffSessionPath  = "/api/v1/freebuff/session"
	freebuffSessionTTL   = 60 * time.Minute
	freebuffCLIUserAgent = "codebuff-cli/0.0.138"
)

type freebuffSession struct {
	InstanceID string
	ExpiresAt  time.Time
}

var (
	freebuffSessionMu    sync.RWMutex
	freebuffSessionCache = make(map[string]*freebuffSession) // key: token::model
)

func getFreebuffSession(token, model string) (*freebuffSession, bool) {
	key := token + "::" + model
	freebuffSessionMu.RLock()
	defer freebuffSessionMu.RUnlock()
	sess, ok := freebuffSessionCache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return sess, true
}

func setFreebuffSession(token, model string, sess *freebuffSession) {
	key := token + "::" + model
	freebuffSessionMu.Lock()
	defer freebuffSessionMu.Unlock()
	freebuffSessionCache[key] = sess
}

func clearFreebuffSession(token, model string) {
	key := token + "::" + model
	freebuffSessionMu.Lock()
	defer freebuffSessionMu.Unlock()
	delete(freebuffSessionCache, key)
}

func newModelLockedError(w http.ResponseWriter, currentModel, requestedModel string) *proxy.UpstreamError {
	errMsg := fmt.Sprintf(`Freebuff session is locked to "%s" — it cannot serve %s. Use "%s" or wait for the session to expire (~1h).`, currentModel, requestedModel, currentModel)
	errBody, _ := json.Marshal(map[string]any{
		"error": map[string]any{
			"message":      errMsg,
			"type":         "model_locked",
			"code":         "model_locked",
			"currentModel": currentModel,
		},
	})
	if w != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write(errBody)
	}
	return &proxy.UpstreamError{
		StatusCode: http.StatusConflict,
		Body:       errBody,
	}
}

func requestFreebuffSession(ctx context.Context, client *http.Client, baseURL, token, model string) (*freebuffSession, error) {
	origin := freebuffOrigin(baseURL)
	reqURL := origin + freebuffSessionPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("create freebuff session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", freebuffCLIUserAgent)
	req.Header.Set("x-freebuff-model", model)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("freebuff session request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read freebuff session response: %w", err)
	}

	var data struct {
		Status         string `json:"status"`
		InstanceID     string `json:"instanceId"`
		ExpiresAt      string `json:"expiresAt"`
		CurrentModel   string `json:"currentModel"`
		RequestedModel string `json:"requestedModel"`
		Message        string `json:"message"`
		Error          string `json:"error"`
	}
	_ = json.Unmarshal(respBytes, &data)

	// Handle model_locked response
	if data.Status == "model_locked" || data.Error == "model_locked" || (resp.StatusCode == http.StatusConflict && (data.CurrentModel != "" || strings.Contains(string(respBytes), "model_locked"))) {
		currentModel := data.CurrentModel
		requestedModel := data.RequestedModel
		if requestedModel == "" {
			requestedModel = model
		}

		// If requestedModel matches currentModel and instanceId is present, accept session!
		if requestedModel == currentModel && data.InstanceID != "" {
			expiresAt := time.Now().Add(freebuffSessionTTL)
			if data.ExpiresAt != "" {
				if t, err := time.Parse(time.RFC3339, data.ExpiresAt); err == nil {
					expiresAt = t
				}
			}
			sess := &freebuffSession{
				InstanceID: data.InstanceID,
				ExpiresAt:  expiresAt,
			}
			setFreebuffSession(token, model, sess)
			return sess, nil
		}

		return nil, newModelLockedError(nil, currentModel, model)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &proxy.UpstreamError{
			StatusCode: resp.StatusCode,
			Body:       respBytes,
		}
	}

	expiresAt := time.Now().Add(freebuffSessionTTL)
	if data.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, data.ExpiresAt); err == nil {
			expiresAt = t
		}
	}

	sess := &freebuffSession{
		InstanceID: data.InstanceID,
		ExpiresAt:  expiresAt,
	}
	setFreebuffSession(token, model, sess)
	return sess, nil
}
