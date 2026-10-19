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

// LocalNamingService 本地命名服务（从配置文件读地址并支持内存动态事件总线）
type LocalNamingService struct {
	// services 本地服务地址表
	services map[string]ServiceAddr
	// cache 服务发现缓存
	cache map[string][]adapter.Instance
	// listeners 变更订阅监听列表
	listeners map[string][]func([]adapter.Instance)
	lock      sync.RWMutex
}

// NewLocalNamingService 创建本地命名服务
// 参数：services-服务地址表（从 argo.yaml 的 services 节读取）
// 返回：本地命名服务实例
func NewLocalNamingService(services map[string]ServiceAddr) *LocalNamingService {
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
		services:  services,
		cache:     cache,
		listeners: make(map[string][]func([]adapter.Instance)),
	}
}

// Register 本地模式注册（记录到内存并广播变更通知）
// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组（本地模式忽略）, weight-权重
// 返回：成功/失败，错误信息
func (ln *LocalNamingService) Register(ip string, port uint64, serviceName, group string, weight float64) (bool, error) {
	if weight <= 0 {
		weight = 10
	}

	inst := adapter.Instance{
		Ip:      ip,
		Port:    port,
		Weight:  weight,
		Healthy: true,
		Enable:  true,
	}
	instances := []adapter.Instance{inst}

	ln.lock.Lock()
	ln.cache[serviceName] = instances
	ln.lock.Unlock()

	ln.notifyListeners(serviceName, instances)
	return true, nil
}

// Deregister 本地模式注销（从内存移除并广播空实例通知）
// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组（本地模式忽略）
// 返回：成功/失败，错误信息
func (ln *LocalNamingService) Deregister(ip string, port uint64, serviceName, group string) (bool, error) {
	ln.lock.Lock()
	delete(ln.cache, serviceName)
	ln.lock.Unlock()

	ln.notifyListeners(serviceName, nil)
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

// DiscoverAll 本地模式发现所有实例
// 参数：serviceName-服务名, group-分组（本地模式忽略）
// 返回：实例切片，错误信息
func (ln *LocalNamingService) DiscoverAll(serviceName, group string) ([]adapter.Instance, error) {
	ln.lock.RLock()
	instances, ok := ln.cache[serviceName]
	ln.lock.RUnlock()

	if !ok || len(instances) == 0 {
		return nil, fmt.Errorf("[local] service %s not found", serviceName)
	}

	result := make([]adapter.Instance, len(instances))
	copy(result, instances)
	return result, nil
}

// Subscribe 本地模式订阅（保存回调并在初始时立即触发一次）
// 参数：serviceName-服务名, group-分组（本地模式忽略）, callback-实例变化回调
// 返回：错误信息
func (ln *LocalNamingService) Subscribe(serviceName, group string, callback func([]adapter.Instance)) error {
	if callback == nil {
		return nil
	}

	ln.lock.Lock()
	ln.listeners[serviceName] = append(ln.listeners[serviceName], callback)
	instances := ln.cache[serviceName]
	ln.lock.Unlock()

	if len(instances) > 0 {
		callback(instances)
	}
	return nil
}

// Unsubscribe 本地模式取消订阅（清理该服务所有已注册的回调）
// 参数：serviceName-服务名, group-分组（本地模式忽略）
// 返回：错误信息
func (ln *LocalNamingService) Unsubscribe(serviceName, group string) error {
	ln.lock.Lock()
	delete(ln.listeners, serviceName)
	ln.lock.Unlock()
	return nil
}

// ============================================================
// 私有辅助方法 (Step-down 下沉)
// ============================================================

// notifyListeners 向指定服务的所有订阅监听者广播最新实例切片
func (ln *LocalNamingService) notifyListeners(serviceName string, instances []adapter.Instance) {
	ln.lock.RLock()
	cbs := make([]func([]adapter.Instance), len(ln.listeners[serviceName]))
	copy(cbs, ln.listeners[serviceName])
	ln.lock.RUnlock()

	for _, cb := range cbs {
		if cb != nil {
			cb(instances)
		}
	}
}
