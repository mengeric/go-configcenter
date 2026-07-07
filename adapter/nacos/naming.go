package nacos

import (
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

// NamingService Nacos 命名服务（实现 adapter.NamingClient 接口）
type NamingService struct {
	client *Client
	// 订阅缓存：key=serviceName+group，value=实例列表
	cache map[string][]adapter.Instance
	lock  sync.RWMutex
}

// NewNamingService 创建 Nacos 命名服务
// 参数：client-Nacos 客户端
// 返回：命名服务实例
func NewNamingService(client *Client) *NamingService {
	return &NamingService{
		client: client,
		cache:  make(map[string][]adapter.Instance),
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
	// 优先从缓存取
	ns.lock.RLock()
	instances, ok := ns.cache[serviceKey(serviceName, group)]
	ns.lock.RUnlock()

	if ok && len(instances) > 0 {
		// 加权轮询（WRR）
		inst := ns.weightedSelect(instances)
		return fmt.Sprintf("%s:%d", inst.Ip, inst.Port), nil
	}

	// 缓存没有，从 Nacos 实时查询
	instance, err := ns.client.NamingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   group,
	})
	if err != nil {
		return "", fmt.Errorf("[nacos] discover service %s failed: %w", serviceName, err)
	}
	return fmt.Sprintf("%s:%d", instance.Ip, instance.Port), nil
}

// weightedSelect 加权轮询选择实例
// 根据实例权重随机选择，权重越高被选中的概率越大
func (ns *NamingService) weightedSelect(instances []adapter.Instance) adapter.Instance {
	// 过滤健康实例
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
		return instances[0]
	}

	if len(healthy) == 1 {
		return healthy[0]
	}

	// 加权随机
	rand := rand.Float64() * totalWeight
	var cumulative float64
	for _, inst := range healthy {
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		cumulative += w
		if rand <= cumulative {
			return inst
		}
	}

	return healthy[len(healthy)-1]
}

// Subscribe 订阅服务实例变化（动态更新本地缓存）
// 参数：serviceName-服务名, group-分组, callback-实例变化回调
// 返回：错误信息
func (ns *NamingService) Subscribe(serviceName, group string, callback func([]adapter.Instance)) error {
	return ns.client.NamingClient.Subscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   group,
		SubscribeCallback: func(instances []model.Instance, err error) {
			if err != nil {
				return
			}
			// 转换为 adapter.Instance
			var list []adapter.Instance
			for _, inst := range instances {
				list = append(list, adapter.Instance{
					Ip:       inst.Ip,
					Port:     inst.Port,
					Weight:   inst.Weight,
					Healthy:  inst.Healthy,
					Enable:   inst.Enable,
					Metadata: inst.Metadata,
				})
			}
			// 更新缓存
			ns.lock.Lock()
			ns.cache[serviceKey(serviceName, group)] = list
			ns.lock.Unlock()

			// 回调通知
			callback(list)
		},
	})
}

// Unsubscribe 取消订阅服务实例变化
// 参数：serviceName-服务名, group-分组
// 返回：错误信息
func (ns *NamingService) Unsubscribe(serviceName, group string) error {
	return ns.client.NamingClient.Unsubscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   group,
		SubscribeCallback: func(services []model.Instance, err error) {
			// 空回调，Unsubscribe 不需要
		},
	})
}

// serviceKey 生成缓存 key
func serviceKey(serviceName, group string) string {
	return serviceName + "|" + group
}
