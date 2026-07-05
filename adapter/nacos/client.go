package nacos

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// ============================================================
// Nacos 客户端封装
// ============================================================

// Client Nacos 客户端（同时包含命名和配置客户端）
type Client struct {
	NamingClient naming_client.INamingClient
	ConfigClient config_client.IConfigClient
}

// Config Nacos 连接配置
type Config struct {
	// Addr Nacos 地址（ip:port）
	Addr string

	// Namespace 命名空间
	Namespace string

	// Group 默认分组
	Group string
}

// NewClient 创建 Nacos 客户端
// 参数：cfg-Nacos 连接配置
// 返回：客户端实例，错误信息
func NewClient(cfg *Config) (*Client, error) {
	// 解析 ip:port
	ip, port, err := parseAddr(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("[nacos] parse addr failed: %w", err)
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

	serverConfigs := []constant.ServerConfig{serverConfig}

	// 创建命名客户端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("[nacos] create naming client failed: %w", err)
	}

	// 创建配置客户端
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("[nacos] create config client failed: %w", err)
	}

	return &Client{
		NamingClient: namingClient,
		ConfigClient: configClient,
	}, nil
}

// parseAddr 解析 ip:port 格式地址
// 参数：addr-地址字符串（如 "192.168.1.1:8848"）
// 返回：IP，端口，错误信息
func parseAddr(addr string) (string, uint64, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid addr format: %s, expected ip:port", addr)
	}
	port, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}
	return parts[0], port, nil
}
