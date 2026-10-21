package executor

import (
	"net/http"
	"testing"
	"time"
)

func TestClaudeServiceRetryAfterHasNoQuotaGrace(t *testing.T) {
	for _, status := range []int{408, 500, 502, 503, 504, 529} {
		err := classifyClaudeUpstreamError(status, http.Header{"Retry-After": []string{"2"}, "Set-Cookie": []string{"private-fixture"}}, []byte(`{"error":{"type":"api_error","message":"network unavailable"}}`))
		retry, ok := err.(interface{ RetryAfter() *time.Duration })
		if !ok || retry.RetryAfter() == nil || *retry.RetryAfter() != 2*time.Second {
			t.Fatalf("status %d: service deadline was discarded or quota fuzz added", status)
		}
		carrier, ok := err.(interface{ Headers() http.Header })
		if !ok || carrier.Headers().Get("Retry-After") != "2" || carrier.Headers().Get("Set-Cookie") != "" {
			t.Fatalf("status %d: retry header lost or unrelated headers exposed", status)
		}
		copy := carrier.Headers()
		copy.Set("Retry-After", "100")
		if carrier.Headers().Get("Retry-After") != "2" {
			t.Fatal("error header state was externally mutated")
		}
	}
}

func TestClaudeServiceMissingOrInvalidRetryAfterKeepsDefault(t *testing.T) {
	for _, raw := range []string{"", "invalid", "-5"} {
		err := classifyClaudeUpstreamError(503, http.Header{"Retry-After": []string{raw}}, []byte(`service unavailable`))
		if err.(interface{ RetryAfter() *time.Duration }).RetryAfter() != nil {
			t.Fatalf("invalid service retry hint accepted: %q", raw)
		}
	}
}
