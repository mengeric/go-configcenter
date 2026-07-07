package local

import (
	"fmt"
	"sync"

	"github.com/mengeric/go-configcenter/adapter"
)

// ============================================================
// 本地模式实现（无注册中心时使用）
// ============================================================

// ServiceAddr 本地服务地址配置
type ServiceAddr struct {
	// Host 服务地址
	Host string `yaml:"host" json:"host"`

	// Port 服务端口
	Port uint64 `yaml:"port" json:"port"`
}

// LocalNamingService 本地命名服务（从配置文件读地址）
type LocalNamingService struct {
	// services 本地服务地址表
	services map[string]ServiceAddr
	// cache 服务发现缓存
	cache map[string][]adapter.Instance
	lock  sync.RWMutex
}

// NewLocalNamingService 创建本地命名服务
// 参数：services-服务地址表（从 argo.yaml 的 services 节读取）
// 返回：本地命名服务实例
func NewLocalNamingService(services map[string]ServiceAddr) *LocalNamingService {
	// 初始化缓存
	cache := make(map[string][]adapter.Instance)
	for name, addr := range services {
		cache[name] = []adapter.Instance{{
			Ip:      addr.Host,
			Port:    addr.Port,
			Weight:  10,
			Healthy: true,
			Enable:  true,
		}}
	}

	return &LocalNamingService{
		services: services,
		cache:    cache,
	}
}

// Register 本地模式注册（记录到内存，不做网络调用）
// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组（本地模式忽略）, weight-权重
// 返回：成功/失败，错误信息
func (ln *LocalNamingService) Register(ip string, port uint64, serviceName, group string, weight float64) (bool, error) {
	ln.lock.Lock()
	defer ln.lock.Unlock()

	if weight <= 0 {
		weight = 10
	}

	ln.cache[serviceName] = []adapter.Instance{{
		Ip:      ip,
		Port:    port,
		Weight:  weight,
		Healthy: true,
		Enable:  true,
	}}
	return true, nil
}

// Deregister 本地模式注销（从内存移除）
// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组（本地模式忽略）
// 返回：成功/失败，错误信息
func (ln *LocalNamingService) Deregister(ip string, port uint64, serviceName, group string) (bool, error) {
	ln.lock.Lock()
	defer ln.lock.Unlock()

	delete(ln.cache, serviceName)
	return true, nil
}

// Discover 本地模式服务发现（从配置文件读地址）
// 参数：serviceName-服务名, group-分组（本地模式忽略）
// 返回：地址字符串（ip:port），错误信息
func (ln *LocalNamingService) Discover(serviceName, group string) (string, error) {
	ln.lock.RLock()
	instances, ok := ln.cache[serviceName]
	ln.lock.RUnlock()

	if !ok || len(instances) == 0 {
		return "", fmt.Errorf("[local] service %s not found", serviceName)
	}
	inst := instances[0]
	return fmt.Sprintf("%s:%d", inst.Ip, inst.Port), nil
}

// Subscribe 本地模式订阅（地址不会变化，直接回调一次）
// 参数：serviceName-服务名, group-分组（本地模式忽略）, callback-实例变化回调
// 返回：错误信息
func (ln *LocalNamingService) Subscribe(serviceName, group string, callback func([]adapter.Instance)) error {
	ln.lock.RLock()
	instances := ln.cache[serviceName]
	ln.lock.RUnlock()

	// 本地模式立即回调一次
	if len(instances) > 0 {
		callback(instances)
	}
	return nil
}

// Unsubscribe 本地模式取消订阅（无操作）
// 参数：serviceName-服务名, group-分组（本地模式忽略）
// 返回：错误信息
func (ln *LocalNamingService) Unsubscribe(serviceName, group string) error {
	return nil
}
