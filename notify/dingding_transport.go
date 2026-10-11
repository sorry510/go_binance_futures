package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	dingTalkRequestTimeout  = 15 * time.Second
	dingTalkDeliveryTimeout = 35 * time.Second
	dingTalkMaxAttempts     = 2
	dingTalkRetryDelay      = 700 * time.Millisecond
)

var dingTalkHTTPClient = &http.Client{Timeout: dingTalkRequestTimeout}

type dingTalkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// sendDingTalkRequest has a finite network budget and retries one transient
// connection error or HTTP 5xx. Application errors such as rate limits, an
// invalid token or invalid message are never blindly retried.
func sendDingTalkRequest(ctx context.Context, client *http.Client, endpoint string, payload []byte) error {
	if client == nil {
		client = dingTalkHTTPClient
	}
	for attempt := 1; attempt <= dingTalkMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("build DingTalk request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("DingTalk delivery timeout: %w", ctx.Err())
			}
			if attempt == dingTalkMaxAttempts {
				return fmt.Errorf("DingTalk network request failed after %d attempts: %w", attempt, err)
			}
		} else {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8192))
			_ = resp.Body.Close()
			if readErr != nil {
				if attempt == dingTalkMaxAttempts {
					return fmt.Errorf("read DingTalk response: %w", readErr)
				}
			} else if resp.StatusCode >= 500 {
				if attempt == dingTalkMaxAttempts {
					return fmt.Errorf("DingTalk HTTP %d after %d attempts", resp.StatusCode, attempt)
				}
			} else if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("DingTalk HTTP %d", resp.StatusCode)
			} else {
				var data dingTalkResponse
				if err := json.Unmarshal(body, &data); err != nil {
					return fmt.Errorf("decode DingTalk response: %w", err)
				}
				if data.ErrCode != 0 {
					return fmt.Errorf("DingTalk rejected message: code=%d reason=%q", data.ErrCode, data.ErrMsg)
				}
				return nil
			}
		}
		if attempt < dingTalkMaxAttempts {
			delay := time.NewTimer(dingTalkRetryDelay)
			select {
			case <-ctx.Done():
				delay.Stop()
				return fmt.Errorf("DingTalk delivery timeout: %w", ctx.Err())
			case <-delay.C:
			}
		}
	}
	return fmt.Errorf("DingTalk delivery exhausted retries")
}
