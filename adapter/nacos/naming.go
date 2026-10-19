package nacos

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/mengeric/go-configcenter/adapter"
)

// ============================================================
// Nacos 服务注册与发现实现
// ============================================================

// ErrNoHealthyInstance 表示无可用健康实例错误
var ErrNoHealthyInstance = errors.New("[nacos] no healthy instance available")

// NamingService Nacos 命名服务（实现 adapter.NamingClient 接口）
type NamingService struct {
	client *Client
	// 订阅缓存：key=serviceName|group，value=健康可用实例列表
	cache map[string][]adapter.Instance
	// 已启动订阅的 serviceKey 集合
	subscribed map[string]bool
	// 监听回调列表：key=serviceName|group，value=回调函数切片
	listeners map[string][]func([]adapter.Instance)
	lock      sync.RWMutex
	subLock   sync.Mutex
}

// NewNamingService 创建 Nacos 命名服务
// 参数：client-Nacos 客户端
// 返回：命名服务实例
func NewNamingService(client *Client) *NamingService {
	return &NamingService{
		client:     client,
		cache:      make(map[string][]adapter.Instance),
		subscribed: make(map[string]bool),
		listeners:  make(map[string][]func([]adapter.Instance)),
	}
}

// Register 注册服务实例到 Nacos
// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组, weight-权重（0使用默认值10）
// 返回：成功/失败，错误信息
func (ns *NamingService) Register(ip string, port uint64, serviceName, group string, weight float64) (bool, error) {
	if weight <= 0 {
		weight = 10
	}
	return ns.client.NamingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: serviceName,
		GroupName:   group,
		Weight:      weight,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})
}

// Deregister 从 Nacos 注销服务实例
// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组
// 返回：成功/失败，错误信息
func (ns *NamingService) Deregister(ip string, port uint64, serviceName, group string) (bool, error) {
	return ns.client.NamingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: serviceName,
		GroupName:   group,
		Ephemeral:   true,
	})
}

// Discover 发现服务实例（返回 ip:port，加权轮询）
// 参数：serviceName-服务名, group-分组
// 返回：地址字符串（ip:port），错误信息
func (ns *NamingService) Discover(serviceName, group string) (string, error) {
	if err := ns.ensureSubscribed(serviceName, group); err != nil {
		// 自动订阅拉取失败时，单次查询兜底
		instance, qErr := ns.client.NamingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
			ServiceName: serviceName,
			GroupName:   group,
		})
		if qErr != nil {
			return "", fmt.Errorf("[nacos] discover service %s failed: %w", serviceName, err)
		}
		return fmt.Sprintf("%s:%d", instance.Ip, instance.Port), nil
	}

	ns.lock.RLock()
	instances, ok := ns.cache[serviceKey(serviceName, group)]
	ns.lock.RUnlock()

	if !ok || len(instances) == 0 {
		return "", fmt.Errorf("[nacos] no available instance found for service %s: %w", serviceName, ErrNoHealthyInstance)
	}

	inst, err := ns.weightedSelect(instances)
	if err != nil {
		return "", fmt.Errorf("[nacos] select instance for %s failed: %w", serviceName, err)
	}
	return fmt.Sprintf("%s:%d", inst.Ip, inst.Port), nil
}

// DiscoverAll 发现所有可用健康服务实例
// 参数：serviceName-服务名, group-分组
// 返回：实例切片，错误信息
func (ns *NamingService) DiscoverAll(serviceName, group string) ([]adapter.Instance, error) {
	if err := ns.ensureSubscribed(serviceName, group); err != nil {
		instances, qErr := ns.client.NamingClient.SelectInstances(vo.SelectInstancesParam{
			ServiceName: serviceName,
			GroupName:   group,
			HealthyOnly: true,
		})
		if qErr != nil {
			return nil, fmt.Errorf("[nacos] discover all instances for %s failed: %w", serviceName, err)
		}
		return toAdapterInstances(instances), nil
	}

	ns.lock.RLock()
	instances := ns.cache[serviceKey(serviceName, group)]
	ns.lock.RUnlock()

	result := make([]adapter.Instance, len(instances))
	copy(result, instances)
	return result, nil
}

// Subscribe 订阅服务实例变化（动态更新本地缓存并追加变更监听）
// 参数：serviceName-服务名, group-分组, callback-实例变化回调（传 nil 仅开启自动缓存同步）
// 返回：错误信息
func (ns *NamingService) Subscribe(serviceName, group string, callback func([]adapter.Instance)) error {
	if err := ns.ensureSubscribed(serviceName, group); err != nil {
		return err
	}

	if callback == nil {
		return nil
	}

	key := serviceKey(serviceName, group)
	ns.lock.Lock()
	ns.listeners[key] = append(ns.listeners[key], callback)
	instances := ns.cache[key]
	ns.lock.Unlock()

	// 初始快照即时回放
	if len(instances) > 0 {
		callback(instances)
	}

	return nil
}

// Unsubscribe 取消订阅服务实例变化
// 参数：serviceName-服务名, group-分组
// 返回：错误信息
func (ns *NamingService) Unsubscribe(serviceName, group string) error {
	key := serviceKey(serviceName, group)

	ns.lock.Lock()
	delete(ns.listeners, key)
	delete(ns.cache, key)
	ns.subscribed[key] = false
	ns.lock.Unlock()

	return ns.client.NamingClient.Unsubscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   group,
		SubscribeCallback: func(services []model.Instance, err error) {
			// 空回调
		},
	})
}

// ============================================================
// 私有辅助方法 (Step-down 下沉)
// ============================================================

// ensureSubscribed 保证该服务已建立动态长连接订阅并完成本地缓存初始化
func (ns *NamingService) ensureSubscribed(serviceName, group string) error {
	key := serviceKey(serviceName, group)

	ns.lock.RLock()
	alreadySubscribed := ns.subscribed[key]
	ns.lock.RUnlock()
	if alreadySubscribed {
		return nil
	}

	ns.subLock.Lock()
	defer ns.subLock.Unlock()

	ns.lock.RLock()
	if ns.subscribed[key] {
		ns.lock.RUnlock()
		return nil
	}
	ns.lock.RUnlock()

	// 1. 同步拉取当前可用健康实例
	instances, err := ns.client.NamingClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   group,
		HealthyOnly: true,
	})
	if err != nil {
		return fmt.Errorf("[nacos] initial fetch instances for service %s failed: %w", serviceName, err)
	}

	// 2. 注册 Nacos 动态订阅变更监听
	err = ns.client.NamingClient.Subscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   group,
		SubscribeCallback: func(nacosInstances []model.Instance, cbErr error) {
			if cbErr != nil {
				return
			}
			ns.handleInstanceUpdate(serviceName, group, nacosInstances)
		},
	})
	if err != nil {
		return fmt.Errorf("[nacos] register subscribe for service %s failed: %w", serviceName, err)
	}

	ns.lock.Lock()
	ns.cache[key] = toAdapterInstances(instances)
	ns.subscribed[key] = true
	ns.lock.Unlock()

	return nil
}

// handleInstanceUpdate 统一处理 Nacos 实例推送并广播给所有订阅者
func (ns *NamingService) handleInstanceUpdate(serviceName, group string, nacosInstances []model.Instance) {
	key := serviceKey(serviceName, group)
	list := toAdapterInstances(nacosInstances)

	ns.lock.Lock()
	ns.cache[key] = list
	listenersCopy := make([]func([]adapter.Instance), len(ns.listeners[key]))
	copy(listenersCopy, ns.listeners[key])
	ns.lock.Unlock()

	for _, cb := range listenersCopy {
		if cb != nil {
			cb(list)
		}
	}
}

// weightedSelect 加权轮询选择实例
// 根据实例权重随机选择，权重越高被选中的概率越大，若无可用实例返回明确错误
func (ns *NamingService) weightedSelect(instances []adapter.Instance) (adapter.Instance, error) {
	var healthy []adapter.Instance
	var totalWeight float64
	for _, inst := range instances {
		if inst.Healthy && inst.Enable {
			w := inst.Weight
			if w <= 0 {
				w = 1
			}
			totalWeight += w
			healthy = append(healthy, inst)
		}
	}

	if len(healthy) == 0 {
		return adapter.Instance{}, ErrNoHealthyInstance
	}

	if len(healthy) == 1 {
		return healthy[0], nil
	}

	randVal := rand.Float64() * totalWeight
	var cumulative float64
	for _, inst := range healthy {
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		cumulative += w
		if randVal <= cumulative {
			return inst, nil
		}
	}

	return healthy[len(healthy)-1], nil
}

// toAdapterInstances 将 Nacos SDK 的 model.Instance 转换为 adapter.Instance
func toAdapterInstances(instances []model.Instance) []adapter.Instance {
	var list []adapter.Instance
	for _, inst := range instances {
		if inst.Healthy && inst.Enable {
			list = append(list, adapter.Instance{
				Ip:       inst.Ip,
				Port:     inst.Port,
				Weight:   inst.Weight,
				Healthy:  inst.Healthy,
				Enable:   inst.Enable,
				Metadata: inst.Metadata,
			})
		}
	}
	return list
}

// serviceKey 生成缓存 key
func serviceKey(serviceName, group string) string {
	return serviceName + "|" + group
}
