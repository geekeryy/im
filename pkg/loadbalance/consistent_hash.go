package loadbalance

import (
	"hash/fnv"
)

// 一致性哈希负载均衡
type ConsistentHashBalancer struct{}

func NewConsistentHashBalancer() *ConsistentHashBalancer {
	return &ConsistentHashBalancer{}
}

func (c *ConsistentHashBalancer) Select(length int64, key string) int64 {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	index := hash.Sum32() % uint32(length)
	return int64(index)
}
