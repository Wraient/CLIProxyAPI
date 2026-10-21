package helps

import (
	"net/http"
	"testing"
	"time"
)

func TestParseServiceRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	for _, raw := range []string{"2", now.Add(2 * time.Second).Format(http.TimeFormat)} {
		delay := ParseServiceRetryAfter(http.Header{"Retry-After": []string{raw}}, now)
		if delay == nil || *delay != 2*time.Second {
			t.Fatalf("retry hint %q: got %v", raw, delay)
		}
	}
	if ParseServiceRetryAfter(nil, now) != nil {
		t.Fatal("missing header should not invent a deadline")
	}
	for _, raw := range []string{"-1", "Inf", "NaN", "1e100", "invalid"} {
		if ParseServiceRetryAfter(http.Header{"Retry-After": []string{raw}}, now) != nil {
			t.Fatalf("invalid retry deadline accepted: %q", raw)
		}
	}
}
