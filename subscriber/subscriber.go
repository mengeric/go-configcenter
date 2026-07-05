package subscriber

import (
	"sync"

	"github.com/mengeric/go-configcenter/adapter"
	"github.com/mengeric/go-configcenter/configmerge"
)

// ============================================================
// go-zero Subscriber 实现
// 实现 go-zero configcenter.Subscriber 接口（Value + AddListener）
// ============================================================

// ConfigSubscriber 配置订阅器（实现 go-zero Subscriber 接口）
type ConfigSubscriber struct {
	// merger 配置合并器
	merger *configmerge.Merger
	// configClient 配置客户端（Nacos 或本地）
	configClient adapter.ConfigClient
	// remoteConfigs 远程配置列表
	remoteConfigs []adapter.RemoteConfig
	// listeners 变更监听回调列表
	listeners []func()
	// lock 互斥锁
	lock sync.Mutex
}

// NewConfigSubscriber 创建配置订阅器
// 参数：merger-配置合并器, configClient-配置客户端, remoteConfigs-远程配置列表
// 返回：配置订阅器实例
func NewConfigSubscriber(
	merger *configmerge.Merger,
	configClient adapter.ConfigClient,
	remoteConfigs []adapter.RemoteConfig,
) *ConfigSubscriber {
	return &ConfigSubscriber{
		merger:        merger,
		configClient:  configClient,
		remoteConfigs: remoteConfigs,
	}
}

// Value 返回当前合并后的配置内容（go-zero Subscriber 接口方法）
// 返回：配置内容字符串，错误信息
func (cs *ConfigSubscriber) Value() (string, error) {
	// 重新合并配置
	data, err := cs.merger.MergeToBytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AddListener 注册配置变更监听（go-zero Subscriber 接口方法）
// 参数：listener-变更回调函数
// 返回：错误信息
func (cs *ConfigSubscriber) AddListener(listener func()) error {
	cs.lock.Lock()
	cs.listeners = append(cs.listeners, listener)
	cs.lock.Unlock()

	// 如果有配置客户端，监听所有远程配置变化
	if cs.configClient != nil {
		for _, rc := range cs.remoteConfigs {
			cs.configClient.WatchConfig(rc.DataId, rc.Group, func(data string) {
				// 远程配置变化，重新拉取并添加到 merger
				cs.merger.AddRemote(data)
				// 通知所有 listener
				cs.lock.Lock()
				for _, l := range cs.listeners {
					go l()
				}
				cs.lock.Unlock()
			})
		}
	}

	return nil
}
