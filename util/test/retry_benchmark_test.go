package test

import (
	"errors"
	"testing"
	"time"

	"github.com/omeyang/gokit/util/retry"
)

// BenchmarkNoRetryPolicy 基准测试 NoRetryPolicy
func BenchmarkNoRetryPolicy(b *testing.B) {
	policy := &retry.NoRetryPolicy{}
	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.ShouldRetry(1, err)
		_ = policy.WaitDuration(1)
	}
}

// BenchmarkSimpleRetryPolicy 基准测试 SimpleRetryPolicy
func BenchmarkSimpleRetryPolicy(b *testing.B) {
	waitTime := 2 * time.Second
	policy := &retry.SimpleRetryPolicy{MaxAttempts: 3, WaitTime: waitTime}
	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for attempt := 0; attempt < 5; attempt++ {
			_ = policy.ShouldRetry(attempt, err)
			_ = policy.WaitDuration(attempt)
		}
	}
}
