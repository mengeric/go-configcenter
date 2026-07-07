package goconfigcenter

import (
	"fmt"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/mengeric/go-configcenter/adapter"
	"github.com/mengeric/go-configcenter/adapter/nacos"
	"github.com/mengeric/go-configcenter/configmerge"
	"github.com/mengeric/go-configcenter/local"
	"github.com/mengeric/go-configcenter/subscriber"
)

// ============================================================
// SDK 统一入口
// ============================================================

// SDK 配置中心 SDK 实例
type SDK struct {
	// configPath argo.yaml 路径
	configPath string
	// config 解析后的配置
	config *ArgoConfig
	// serviceName 当前服务名
	serviceName string
	// namingClient 命名服务客户端（Nacos 或本地）
	namingClient adapter.NamingClient
	// configClient 配置客户端（Nacos 或本地）
	configClient adapter.ConfigClient
	// merger 配置合并器
	merger *configmerge.Merger
	// registeredIP 注册的 IP（用于注销）
	registeredIP string
	// registeredPort 注册的端口（用于注销）
	registeredPort uint64
	// group 默认分组
	group string
}

// MustInit 初始化 SDK（失败直接 panic）
// 参数：configPath-argo.yaml 路径, serviceName-当前服务名
// 返回：SDK 实例
func MustInit(configPath, serviceName string) *SDK {
	s, err := Init(configPath, serviceName)
	if err != nil {
		panic(fmt.Sprintf("[goconfigcenter] init failed: %v", err))
	}
	return s
}

// Init 初始化 SDK
// 参数：configPath-argo.yaml 路径, serviceName-当前服务名
// 返回：SDK 实例，错误信息
func Init(configPath, serviceName string) (*SDK, error) {
	// 1. 加载 argo.yaml
	k := koanf.New(".")
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("load argo.yaml failed: %w", err)
	}

	// 2. 解析配置
	var cfg ArgoConfig
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		return nil, fmt.Errorf("unmarshal argo.yaml failed: %w", err)
	}

	// 3. 查找目标服务配置
	svcConfig, ok := cfg.Services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found in argo.yaml", serviceName)
	}

	// 4. 初始化注册中心客户端
	var namingClient adapter.NamingClient
	var configClient adapter.ConfigClient

	if cfg.Registry.Type == "nacos" && cfg.Registry.Addr != "" {
		// Nacos 模式
		nacosClient, err := nacos.NewClient(&nacos.Config{
			Addr:      cfg.Registry.Addr,
			Namespace: cfg.Registry.Namespace,
			Group:     cfg.Registry.Group,
			Username:  cfg.Registry.Username,
			Password:  cfg.Registry.Password,
		})
		if err != nil {
			return nil, fmt.Errorf("init nacos client failed: %w", err)
		}
		namingClient = nacos.NewNamingService(nacosClient)
		configClient = nacos.NewConfigService(nacosClient)
	} else {
		// 本地模式（无注册中心）
		localServices := make(map[string]local.ServiceAddr)
		for name, svc := range cfg.Services {
			if svc.Host != "" && svc.Port > 0 {
				localServices[name] = local.ServiceAddr{
					Host: svc.Host,
					Port: svc.Port,
				}
			}
		}
		namingClient = local.NewLocalNamingService(localServices)
		configClient = local.NewLocalConfigService(filepath.Dir(configPath))
	}

	// 5. 创建配置合并器
	merger := configmerge.NewMerger(serviceName, svcConfig.Local)

	// 6. 加载远程配置（如果有 Nacos）
	if configClient != nil && cfg.Registry.Type == "nacos" {
		for _, rc := range svcConfig.Remote {
			content, err := configClient.GetConfig(rc.DataId, rc.Group)
			if err != nil {
				return nil, fmt.Errorf("get remote config %s/%s failed: %w", rc.Group, rc.DataId, err)
			}
			merger.AddRemote(content)
		}
	}

	return &SDK{
		configPath: configPath,
		config:     &cfg,
		serviceName: serviceName,
		namingClient: namingClient,
		configClient: configClient,
		merger:      merger,
		group:       cfg.Registry.Group,
	}, nil
}

// Register 注册服务到注册中心
// 参数：ip-监听地址, port-监听端口
// 返回：错误信息
func (s *SDK) Register(ip string, port uint64) error {
	ok, err := s.namingClient.Register(ip, port, s.serviceName, s.group)
	if err != nil {
		return fmt.Errorf("register service %s failed: %w", s.serviceName, err)
	}
	if !ok {
		return fmt.Errorf("register service %s returned false", s.serviceName)
	}

	// 记录注册信息（用于注销）
	s.registeredIP = ip
	s.registeredPort = port
	return nil
}

// Deregister 从注册中心注销服务
// 返回：错误信息
func (s *SDK) Deregister() error {
	ok, err := s.namingClient.Deregister(s.registeredIP, s.registeredPort, s.serviceName, s.group)
	if err != nil {
		return fmt.Errorf("deregister service %s failed: %w", s.serviceName, err)
	}
	if !ok {
		return fmt.Errorf("deregister service %s returned false", s.serviceName)
	}
	return nil
}

// Discover 发现服务实例（返回 ip:port）
// 参数：serviceName-要发现的服务名
// 返回：地址字符串，错误信息
func (s *SDK) Discover(serviceName string) (string, error) {
	return s.namingClient.Discover(serviceName, s.group)
}

// Subscriber 返回 go-zero configcenter.Subscriber 实现
// 返回：配置订阅器实例
func (s *SDK) Subscriber() *subscriber.ConfigSubscriber {
	// 转换 remote 配置
	var remoteConfigs []adapter.RemoteConfig
	for _, rc := range s.config.Services[s.serviceName].Remote {
		remoteConfigs = append(remoteConfigs, adapter.RemoteConfig{
			DataId: rc.DataId,
			Group:  rc.Group,
		})
	}

	return subscriber.NewConfigSubscriber(
		s.merger,
		s.configClient,
		remoteConfigs,
	)
}

// ConfigType 根据配置文件扩展名判断配置类型
// 返回："yaml"、"json"、"toml"
func (s *SDK) ConfigType() string {
	return s.merger.ConfigType()
}



// ServiceName 获取当前服务名
// 返回：服务名
func (s *SDK) ServiceName() string {
	return s.serviceName
}

// Config 获取解析后的 argo.yaml 配置
// 返回：配置结构体指针
func (s *SDK) Config() *ArgoConfig {
	return s.config
}
