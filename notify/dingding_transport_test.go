package notify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDingTalkDeliveryRetriesTemporaryServerError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("unexpected request method %s", r.Method)
		}
		if requests.Load() == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := sendDingTalkRequest(ctx, server.Client(), server.URL, []byte(`{"msgtype":"markdown"}`)); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("expected one retry, got %d requests", requests.Load())
	}
}

func TestDingTalkRejectDoesNotRetry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"errcode":310000,"errmsg":"keyword mismatch"}`))
	}))
	defer server.Close()

	err := sendDingTalkRequest(context.Background(), server.Client(), server.URL, []byte("{}"))
	if err == nil || !strings.Contains(err.Error(), "310000") {
		t.Fatalf("expected DingTalk application error, got %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("application error must not retry, got %d requests", requests.Load())
	}
}

func TestDingTalkDeliveryHonorsCallerDeadline(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		// A genuinely slow response, without depending on server-side
		// request cancellation behavior in the test HTTP transport.
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	err := sendDingTalkRequest(ctx, server.Client(), server.URL, []byte("{}"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected a bounded deadline, got %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("should stop once deadline expires, got %d", requests.Load())
	}
}
