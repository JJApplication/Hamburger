/*
Project: Sandwich balance.go
Created: 2021/12/12 by Landers
*/

package balancer

import (
	"Hamburger/internal/structure"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gookit/goutil/mathutil"
)

// 负载均衡
// 基于轮询和随机的混合算法，优化性能

type LoadBalancer struct {
	counter    uint64 // 原子计数器，用于轮询
	ports      []int
	length     int
	lastUpdate time.Time
	mu         sync.RWMutex
}

var (
	// 全局负载均衡器实例
	balancerCache = structure.NewMap[*LoadBalancer](100)
)

// GetBalancer 获取或创建负载均衡器
func GetBalancer(ports []int) *LoadBalancer {
	if len(ports) == 0 {
		return nil
	}

	// 为hosts列表创建唯一key
	key := ""
	for _, port := range ports {
		key += strconv.Itoa(port) + "|"
	}

	balancer, exists := balancerCache.Get(key)
	if exists {
		return balancer
	}

	balancer = &LoadBalancer{
		ports:      make([]int, len(ports)),
		length:     len(ports),
		lastUpdate: time.Now(),
	}
	copy(balancer.ports, ports)
	balancerCache.Put(key, balancer)

	return balancer
}

// PickOne 使用轮询算法选择主机，性能更优
func (lb *LoadBalancer) PickOne() int {
	if lb.length == 1 {
		return lb.ports[0]
	}

	// 使用原子操作的轮询算法
	index := atomic.AddUint64(&lb.counter, 1) % uint64(lb.length)
	return lb.ports[index]
}

// PickOne 保持向后兼容的函数接口
func PickOne(hosts []int) int {
	if len(hosts) == 1 {
		return hosts[0]
	}

	// 对于临时请求，使用简单的随机算法
	i := mathutil.RandomInt(0, len(hosts)-1)
	return hosts[i]
}

// PickOneRoundRobin 高性能轮询选择
func PickOneRoundRobin(ports []int) int {
	balancer := GetBalancer(ports)
	if balancer == nil {
		return 0
	}
	return balancer.PickOne()
}
