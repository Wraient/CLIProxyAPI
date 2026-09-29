package auth

import (
	"context"
	"testing"
	"time"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

// With disable-cooling=true, stock CLIProxyAPI ignores 429 cooldowns. Free-usage
// exhaustion still must park the auth for the explicit RetryAfter (24h).
func TestMarkResult_HonorsRetryAfterWithDisableCooling(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfigSnapshot(&internalconfig.Config{DisableCooling: true})

	auth := &Auth{ID: "xai-free-exhausted", Provider: "xai", Status: StatusActive}
	if _, err := manager.Register(WithSkipPersist(context.Background()), auth); err != nil {
		t.Fatalf("Register: %v", err)
	}

	retryAfter := 24 * time.Hour
	manager.MarkResult(context.Background(), Result{
		AuthID:     auth.ID,
		Provider:   auth.Provider,
		Model:      "grok-4.5",
		Success:    false,
		RetryAfter: &retryAfter,
		Error:      &Error{Message: "free-usage-exhausted", HTTPStatus: 429},
	})

	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatal("auth missing after MarkResult")
	}
	state := updated.ModelStates["grok-4.5"]
	if state == nil {
		t.Fatal("expected model state for grok-4.5")
	}
	if state.NextRetryAfter.IsZero() {
		t.Fatal("NextRetryAfter is zero; free-usage auth would be retried immediately")
	}
	remaining := time.Until(state.NextRetryAfter)
	if remaining < 23*time.Hour || remaining > 24*time.Hour+time.Minute {
		t.Fatalf("NextRetryAfter remaining = %v, want ~24h", remaining)
	}
	if !state.Unavailable {
		t.Fatal("model state should be Unavailable during free-usage cooldown")
	}
	if blocked, _, _ := isAuthBlockedForModel(updated, "grok-4.5", time.Now()); !blocked {
		t.Fatal("auth must be blocked for model while free-usage cooldown is active")
	}
	if blocked, _, _ := isAuthBlockedForModel(updated, "grok-4.5", state.NextRetryAfter.Add(time.Second)); blocked {
		t.Fatal("auth should unblock after free-usage RetryAfter elapses")
	}
}

func TestApplyAuthFailureState_HonorsRetryAfterWithDisableCooling(t *testing.T) {
	auth := &Auth{ID: "xai-no-model", Provider: "xai", Status: StatusActive}
	retryAfter := 24 * time.Hour
	now := time.Now()
	applyAuthFailureState(auth, &Error{Message: "free-usage-exhausted", HTTPStatus: 429}, &retryAfter, now, true)
	if auth.NextRetryAfter.IsZero() {
		t.Fatal("NextRetryAfter is zero under disable-cooling")
	}
	if !auth.NextRetryAfter.Equal(now.Add(retryAfter)) {
		t.Fatalf("NextRetryAfter = %v, want %v", auth.NextRetryAfter, now.Add(retryAfter))
	}
	if !auth.Unavailable {
		t.Fatal("auth should remain Unavailable when explicit RetryAfter is set")
	}
}
