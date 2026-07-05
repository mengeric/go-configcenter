package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// ============================================================
// Nacos 配置管理实现
// ============================================================

// ConfigService Nacos 配置服务（实现 adapter.ConfigClient 接口）
type ConfigService struct {
	client *Client
}

// NewConfigService 创建 Nacos 配置服务
// 参数：client-Nacos 客户端
// 返回：配置服务实例
func NewConfigService(client *Client) *ConfigService {
	return &ConfigService{
		client: client,
	}
}

// GetConfig 从 Nacos 拉取配置内容
// 参数：dataId-配置ID, group-分组
// 返回：配置内容字符串，错误信息
func (cs *ConfigService) GetConfig(dataId, group string) (string, error) {
	content, err := cs.client.ConfigClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return "", err
	}
	return content, nil
}

// WatchConfig 监听 Nacos 配置变化
// 参数：dataId-配置ID, group-分组, callback-配置变化回调（返回新内容）
// 返回：错误信息
func (cs *ConfigService) WatchConfig(dataId, group string, callback func(string)) error {
	return cs.client.ConfigClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			callback(data)
		},
	})
}
