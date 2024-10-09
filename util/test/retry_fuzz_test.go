package test

import (
	"testing"
	"time"

	"github.com/omeyang/gokit/util/retry"
)

// FuzzNoRetryPolicy fuzzing测试 NoRetryPolicy
func FuzzNoRetryPolicy(f *testing.F) {
	f.Add(0)
	f.Fuzz(func(t *testing.T, attempt int) {
		policy := &retry.NoRetryPolicy{}
		_ = policy.ShouldRetry(attempt, nil)
		_ = policy.WaitDuration(attempt)
	})
}

// FuzzSimpleRetryPolicy fuzzing测试 SimpleRetryPolicy
func FuzzSimpleRetryPolicy(f *testing.F) {
	f.Add(0, 5, 1)
	f.Fuzz(func(t *testing.T, attempt int, maxAttempts int, waitTimeSeconds int) {
		waitTime := time.Duration(waitTimeSeconds) * time.Second
		policy := &retry.SimpleRetryPolicy{
			MaxAttempts: maxAttempts,
			WaitTime:    waitTime,
		}
		_ = policy.ShouldRetry(attempt, nil)
		_ = policy.WaitDuration(attempt)
	})
}

// FuzzExponentialBackoffRetryPolicy fuzzing测试 ExponentialBackoffRetryPolicy
func FuzzExponentialBackoffRetryPolicy(f *testing.F) {
	f.Add(0, 1, 5)
	f.Fuzz(func(t *testing.T, attempt int, baseWaitTimeSeconds int, maxAttempts int) {
		baseWaitTime := time.Duration(baseWaitTimeSeconds) * time.Second
		policy := &retry.ExponentialBackoffRetryPolicy{
			BaseWaitTime: baseWaitTime,
			MaxAttempts:  maxAttempts,
		}
		_ = policy.ShouldRetry(attempt, nil)
		_ = policy.WaitDuration(attempt)
	})
}
