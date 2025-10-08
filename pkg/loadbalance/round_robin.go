package loadbalance

import "sync/atomic"

// 轮询负载均衡
type RoundRobinBalancer struct {
	counter uint64
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

func (r *RoundRobinBalancer) Select(length int64, _ string) int64 {
	index := atomic.AddUint64(&r.counter, 1) % uint64(length)
	return int64(index)
}
