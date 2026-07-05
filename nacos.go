package argosdk

import (
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// NacosClient Nacos 客户端封装
type NacosClient struct {
	namingClient naming_client.INamingClient
	configClient config_client.IConfigClient
}

// NewNacosClient 创建 Nacos 客户端
func NewNacosClient(cfg *NacosConfig) (*NacosClient, error) {
	// 解析 addr → ip:port
	ip, port, err := parseAddr(cfg.Addr)
	if err != nil {
		return nil, err
	}

	// 客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:         cfg.Namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "info",
	}

	// 服务端配置
	serverConfig := constant.ServerConfig{
		IpAddr: ip,
		Port:   port,
	}

	// 创建命名客户端（服务注册与发现）
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: []constant.ServerConfig{serverConfig},
		},
	)
	if err != nil {
		return nil, err
	}

	// 创建配置客户端（配置管理）
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: []constant.ServerConfig{serverConfig},
		},
	)
	if err != nil {
		return nil, err
	}

	return &NacosClient{
		namingClient: namingClient,
		configClient: configClient,
	}, nil
}

// Register 注册服务到 Nacos
func (nc *NacosClient) Register(serviceName, ip string, port uint64, group string) (bool, error) {
	return nc.namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		GroupName:   group,
	})
}

// Deregister 从 Nacos 注销服务
func (nc *NacosClient) Deregister(serviceName, ip string, port uint64, group string) (bool, error) {
	return nc.namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: serviceName,
		Ephemeral:   true,
		GroupName:   group,
	})
}

// Discover 发现服务实例（返回 ip:port）
func (nc *NacosClient) Discover(serviceName string, group string) (string, error) {
	instance, err := nc.namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   group,
	})
	if err != nil {
		return "", err
	}
	return instance.Ip + ":" + strconv.FormatUint(instance.Port, 10), nil
}

// DiscoverAll 获取服务所有实例
func (nc *NacosClient) DiscoverAll(serviceName string, group string) ([]NacosInstance, error) {
	instances, err := nc.namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   group,
	})
	if err != nil {
		return nil, err
	}
	var result []NacosInstance
	for _, inst := range instances {
		result = append(result, NacosInstance{
			Ip:      inst.Ip,
			Port:    inst.Port,
			Weight:  inst.Weight,
			Healthy: inst.Healthy,
		})
	}
	return result, nil
}

// WatchService 监听服务实例变化
func (nc *NacosClient) WatchService(serviceName, group string, fn func([]NacosInstance)) error {
	return nc.namingClient.Subscribe(vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   group,
		SubscribeCallback: func(instances []model.Instance, err error) {
			if err != nil {
				return
			}
			var result []NacosInstance
			for _, inst := range instances {
				result = append(result, NacosInstance{
					Ip:      inst.Ip,
					Port:    inst.Port,
					Weight:  inst.Weight,
					Healthy: inst.Healthy,
				})
			}
			fn(result)
		},
	})
}

// GetConfig 拉取 Nacos 配置
func (nc *NacosClient) GetConfig(dataId, group string) (string, error) {
	return nc.configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
}

// WatchConfig 监听 Nacos 配置变化
func (nc *NacosClient) WatchConfig(dataId, group string, fn func(data string)) error {
	return nc.configClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			fn(data)
		},
	})
}

// NacosInstance Nacos 服务实例
type NacosInstance struct {
	Ip      string
	Port    uint64
	Weight  float64
	Healthy bool
}

// parseAddr 解析 ip:port 格式的地址
func parseAddr(addr string) (string, uint64, error) {
	parts := strings.Split(addr, ":")
	port, _ := strconv.ParseUint(parts[1], 10, 64)
	return parts[0], port, nil
}
