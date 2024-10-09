package sample

import (
	"math/rand"
	"sync/atomic"
	"time"
)

// SamplerType 定义采样器类型
type SamplerType string

const (
	// RateSamplerType 表示基于比率的采样器
	RateSamplerType SamplerType = "rate"
	// JitterSamplerType 表示基于抖动的采样器
	JitterSamplerType SamplerType = "jitter"
)

// Sampler 定义采样器接口
type Sampler interface {
	Sample() bool
	SetRate(rate float64)
	GetRate() float64
}

// RateSampler 实现基于比率的采样
type RateSampler struct {
	// 高效地实现浮点数采样率
	// 采样率的范围是 0.0 到 1.0 的浮点数, 将这个浮点数乘以 2^63（即 1 << 63），然后转换为 uint64。
	// 这样可以将 0.0 到 1.0 的范围映射到 0 到 2^63 的整数范围。
	// 好处是使用整数可以利用原子操作,也更快一些
	rate uint64
}

// NewRateSampler 创建一个新的 RateSampler
func NewRateSampler(rate float64) *RateSampler {
	return &RateSampler{
		rate: uint64(rate * (1 << 63)),
	}
}

// Sample 根据给定的比率进行采样
func (s *RateSampler) Sample() bool {
	return rand.Uint64() < atomic.LoadUint64(&s.rate)
}

// SetRate 设置新的采样率
func (s *RateSampler) SetRate(rate float64) {
	atomic.StoreUint64(&s.rate, uint64(rate*(1<<63)))
}

// GetRate 获取当前采样率
func (s *RateSampler) GetRate() float64 {
	return float64(atomic.LoadUint64(&s.rate)) / (1 << 63)
}

// JitterSampler 实现基于抖动的采样
type JitterSampler struct {
	rate       atomic.Value  // 存储 float64 类型的采样率
	jitter     time.Duration // 定义两次采样之间的最小时间间隔
	lastSample time.Time     // 记录上一次采样成功的时间
}

// NewJitterSampler 创建一个新的 JitterSampler
func NewJitterSampler(rate float64, jitter time.Duration) *JitterSampler {
	s := &JitterSampler{
		jitter: jitter,
	}
	s.rate.Store(rate)
	return s
}

// Sample 根据给定的抖动进行采样
func (s *JitterSampler) Sample() bool {
	now := time.Now()
	if now.Sub(s.lastSample) < s.jitter {
		return false
	}
	if rand.Float64() < s.rate.Load().(float64) {
		s.lastSample = now
		return true
	}
	return false
}

// SetRate 设置新的采样率
func (s *JitterSampler) SetRate(rate float64) {
	s.rate.Store(rate)
}

// GetRate 获取当前采样率
func (s *JitterSampler) GetRate() float64 {
	return s.rate.Load().(float64)
}
