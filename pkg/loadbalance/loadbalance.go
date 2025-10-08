package loadbalance

// 负载均衡算法接口
type LoadBalancer interface {
	Select(length int64, key string) int64
}
