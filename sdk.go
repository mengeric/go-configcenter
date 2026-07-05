package argosdk

import (
	"fmt"
	"os"

	"github.com/knadh/koanf/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
)

// SDK 统一入口
type SDK struct {
	configPath string            // argo.yaml 路径
	config     *ArgoConfig       // 解析后的配置
	nacos      *NacosClient      // Nacos 客户端（nil = 纯本地模式）
	koanf      *koanf.Koanf      // koanf 实例
	service    *ServiceInfo       // 当前服务信息
}

// MustInit 初始化 SDK（失败直接 panic）
func MustInit(configPath, serviceName string) *SDK {
	s, err := Init(configPath, serviceName)
	if err != nil {
		panic(fmt.Sprintf("[argo-sdk] init failed: %v", err))
	}
	return s
}

// Init 初始化 SDK
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
	svc, ok := cfg.Services[serviceName]
	if !ok {
		// 如果 services 节为空，使用 service 节（单服务模式）
		if cfg.Service.Name == serviceName {
			svc = ServiceAddr{
				Host: cfg.Service.Host,
				Port: cfg.Service.Port,
			}
		} else {
			return nil, fmt.Errorf("service %s not found in argo.yaml", serviceName)
		}
	}

	// 4. 初始化 Nacos 客户端（如果有配置）
	var nacosClient *NacosClient
	if cfg.Nacos != nil && cfg.Nacos.Addr != "" {
		var err error
		nacosClient, err = NewNacosClient(cfg.Nacos)
		if err != nil {
			return nil, fmt.Errorf("init nacos failed: %w", err)
		}
	}

	sdk := &SDK{
		configPath: configPath,
		config:     &cfg,
		nacos:      nacosClient,
		koanf:      koanf.New("."),
		service: &ServiceInfo{
			Name: serviceName,
			Host: svc.Host,
			Port: svc.Port,
		},
	}

	// 5. 加载配置文件（本地 merge）
	if err := sdk.loadLocalConfigs(); err != nil {
		return nil, fmt.Errorf("load local configs failed: %w", err)
	}

	// 6. 如果有 Nacos，拉远程配置 merge
	if nacosClient != nil {
		if err := sdk.loadRemoteConfigs(); err != nil {
			return nil, fmt.Errorf("load remote configs failed: %w", err)
		}
	}

	return sdk, nil
}

// loadLocalConfigs 加载本地配置文件（按顺序 merge）
func (s *SDK) loadLocalConfigs() error {
	for _, f := range s.config.Local {
		if err := s.koanf.Load(file.Provider(f), yaml.Parser()); err != nil {
			return fmt.Errorf("load %s failed: %w", f, err)
		}
	}
	return nil
}

// loadRemoteConfigs 从 Nacos 拉取远程配置并 merge
func (s *SDK) loadRemoteConfigs() error {
	for _, rc := range s.config.Remote {
		content, err := s.nacos.GetConfig(rc.DataId, rc.Group)
		if err != nil {
			return fmt.Errorf("get nacos config %s/%s failed: %w", rc.Group, rc.DataId, err)
		}
		if content == "" {
			continue
		}
		if err := s.koanf.Load(rawbytes.Provider([]byte(content)), yaml.Parser()); err != nil {
			return fmt.Errorf("load nacos config %s failed: %w", rc.DataId, err)
		}
	}
	return nil
}

// MergedConfigPath 输出合并后的配置文件路径（go-zero 用这个加载）
func (s *SDK) MergedConfigPath() string {
	data := s.koanf.Slices()
	merged := fmt.Sprintf("merged-%s.yaml", s.service.Name)
	// 写入当前目录
	os.WriteFile(merged, marshalYaml(data), 0644)
	return merged
}

// Get 获取配置值
func (s *SDK) Get(key string) string {
	return s.koanf.String(key)
}

// GetInt 获取整数配置值
func (s *SDK) GetInt(key string) int {
	return s.koanf.Int(key)
}

// Has 判断配置是否存在
func (s *SDK) Has(key string) bool {
	return s.koanf.Exists(key)
}

// Koanf 返回 koanf 实例（高级用法）
func (s *SDK) Koanf() *koanf.Koanf {
	return s.koanf
}
