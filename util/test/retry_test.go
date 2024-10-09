package test

import (
	"errors"
	"testing"
	"time"

	"github.com/omeyang/gokit/util/retry"
)

// TestNoRetryPolicy 测试 NoRetryPolicy
func TestNoRetryPolicy(t *testing.T) {
	policy := &retry.NoRetryPolicy{}

	if policy.ShouldRetry(1, errors.New("test error")) {
		t.Errorf("NoRetryPolicy.ShouldRetry() = true, want false")
	}

	if policy.WaitDuration(1) != 0 {
		t.Errorf("NoRetryPolicy.WaitDuration() = %d, want 0", policy.WaitDuration(1))
	}
}

// TestSimpleRetryPolicy 测试 SimpleRetryPolicy
func TestSimpleRetryPolicy(t *testing.T) {
	waitTime := 2 * time.Second
	policy := &retry.SimpleRetryPolicy{MaxAttempts: 3, WaitTime: waitTime}

	tests := []struct {
		attempt   int
		wantRetry bool
	}{
		{0, true},
		{1, true},
		{2, true},
		{3, false},
		{4, false},
	}

	for _, tt := range tests {
		if policy.ShouldRetry(tt.attempt, errors.New("test error")) != tt.wantRetry {
			t.Errorf("SimpleRetryPolicy.ShouldRetry(%d) = %v, want %v", tt.attempt, !tt.wantRetry, tt.wantRetry)
		}
	}

	if policy.WaitDuration(1) != int(waitTime.Seconds()) {
		t.Errorf("SimpleRetryPolicy.WaitDuration() = %d, want %d", policy.WaitDuration(1), int(waitTime.Seconds()))
	}

	// 测试最大尝试次数为0的情况
	policy = &retry.SimpleRetryPolicy{MaxAttempts: 0, WaitTime: waitTime}
	if policy.ShouldRetry(0, errors.New("test error")) {
		t.Errorf("SimpleRetryPolicy.ShouldRetry(0) = true, want false")
	}

	if policy.WaitDuration(0) != int(waitTime.Seconds()) {
		t.Errorf("SimpleRetryPolicy.WaitDuration() = %d, want %d", policy.WaitDuration(0), int(waitTime.Seconds()))
	}
}

func TestExponentialBackoffRetryPolicy(t *testing.T) {
	policy := &retry.ExponentialBackoffRetryPolicy{
		BaseWaitTime: 2 * time.Second,
		MaxAttempts:  3,
	}

	tests := []struct {
		attempt       int
		expectedWait  int
		expectedRetry bool
	}{
		{0, 2, true},
		{1, 4, true},
		{2, 6, true},
		{3, 0, false},
	}

	for _, tt := range tests {
		if policy.ShouldRetry(tt.attempt, nil) != tt.expectedRetry {
			t.Errorf("ShouldRetry(%d) = %v; want %v", tt.attempt, !tt.expectedRetry, tt.expectedRetry)
		}
		if policy.WaitDuration(tt.attempt) != tt.expectedWait {
			t.Errorf("WaitDuration(%d) = %v; want %v", tt.attempt, policy.WaitDuration(tt.attempt), tt.expectedWait)
		}
	}
}
