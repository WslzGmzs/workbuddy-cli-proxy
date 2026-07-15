package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUpstreamHTTPError_QuotaExhausted429(t *testing.T) {
	t.Parallel()
	body := []byte(`{"error":{"data":{"code":14018,"msg":"额度已用尽，请访问以下链接，购买加量包以获取更多额度：https://www.codebuddy.cn/profile/usage ","requestId":"x"}}}`)
	err := upstreamHTTPError(429, body, nil)
	se, ok := err.(*statusError)
	if !ok {
		t.Fatalf("type %T, want *statusError", err)
	}
	if se.StatusCode() != 429 {
		t.Fatalf("status=%d", se.StatusCode())
	}
	if se.RetryAfter() == nil || *se.RetryAfter() < 29*time.Minute {
		t.Fatalf("RetryAfter=%v, want ~30m for quota exhausted", se.RetryAfter())
	}
	if !strings.Contains(se.Error(), "upstream 429") {
		t.Fatalf("message=%q", se.Error())
	}
}

func TestUpstreamHTTPError_RetryAfterHeader(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Retry-After", "120")
	err := upstreamHTTPError(429, []byte(`rate limit`), h)
	se := err.(*statusError)
	if se.RetryAfter() == nil || *se.RetryAfter() != 120*time.Second {
		t.Fatalf("RetryAfter=%v, want 120s", se.RetryAfter())
	}
}

func TestErrorEnvelopeFromErr_PreservesHTTPStatus(t *testing.T) {
	t.Parallel()
	raw := errorEnvelopeFromErr(&statusError{
		Message:    "upstream 429: quota",
		Code:       "upstream_error",
		HTTPStatus: 429,
		Retryable:  true,
	})
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env.OK {
		t.Fatal("expected ok=false")
	}
	if env.Error == nil || env.Error.HTTPStatus != 429 {
		t.Fatalf("error=%#v", env.Error)
	}
	if env.Error.Code != "upstream_error" {
		t.Fatalf("code=%q", env.Error.Code)
	}
	if !env.Error.Retryable {
		t.Fatal("expected retryable")
	}
}

func TestIsQuotaExhaustedBody(t *testing.T) {
	t.Parallel()
	if !isQuotaExhaustedBody([]byte(`{"code":14018}`)) {
		t.Fatal("14018")
	}
	if !isQuotaExhaustedBody([]byte(`额度已用尽`)) {
		t.Fatal("chinese")
	}
	if isQuotaExhaustedBody([]byte(`ok`)) {
		t.Fatal("false positive")
	}
}
