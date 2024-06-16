package util

import "time"

// RetryPolicy 定义重试策略的接口
type RetryPolicy interface {
	// ShouldRetry 决定给定的尝试次数和遇到的错误是否应该重试
	ShouldRetry(attempt int, err error) bool
	// WaitDuration 返回给定尝试次数后，应该等待多久再进行重试
	WaitDuration(attempt int) int
}

// NoRetryPolicy 不重试策略
type NoRetryPolicy struct{}

func (np *NoRetryPolicy) ShouldRetry(_ int, _ error) bool {
	return false // 永远不重试
}

func (np *NoRetryPolicy) WaitDuration(_ int) int {
	return 0 // 等待时间为0，因为不会进行重试
}

// SimpleRetryPolicy 是一个基于固定时间间隔和最大尝试次数的简单重试策略
type SimpleRetryPolicy struct {
	MaxAttempts int           // 最大重试次数
	WaitTime    time.Duration // 两次重试之间的等待时间
}

// ShouldRetry 返回true如果尝试次数小于MaxAttempts
func (p *SimpleRetryPolicy) ShouldRetry(attempt int, _ error) bool {
	return attempt < p.MaxAttempts
}

// WaitDuration 返回等待时间，单位为秒
func (p *SimpleRetryPolicy) WaitDuration(_ int) int {
	return int(p.WaitTime.Seconds())
}

// ExponentialBackoffRetryPolicy 是一个基于指数退避的重试策略
type ExponentialBackoffRetryPolicy struct {
	BaseWaitTime time.Duration // 基础等待时间
	MaxAttempts  int           // 最大重试次数
}

// ShouldRetry 返回true如果尝试次数小于MaxAttempts
func (p *ExponentialBackoffRetryPolicy) ShouldRetry(attempt int, _ error) bool {
	return attempt < p.MaxAttempts
}

// WaitDuration 返回等待时间，单位为秒，随着尝试次数增加而增加
func (p *ExponentialBackoffRetryPolicy) WaitDuration(attempt int) int {
	if attempt >= p.MaxAttempts {
		return 0
	}
	return int(p.BaseWaitTime.Seconds()) * (attempt + 1)
}
