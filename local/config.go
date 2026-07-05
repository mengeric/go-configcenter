package local

import (
	"fmt"
	"os"
)

// ============================================================
// 本地配置管理实现（无 Nacos 时使用）
// ============================================================

// LocalConfigService 本地配置服务
type LocalConfigService struct {
	// configDir 配置文件目录
	configDir string
}

// NewLocalConfigService 创建本地配置服务
// 参数：configDir-配置文件所在目录
// 返回：本地配置服务实例
func NewLocalConfigService(configDir string) *LocalConfigService {
	return &LocalConfigService{
		configDir: configDir,
	}
}

// GetConfig 读取本地配置文件
// 参数：dataId-文件名（如 "shared.yaml"）, group-分组（本地模式忽略）
// 返回：文件内容字符串，错误信息
func (lc *LocalConfigService) GetConfig(dataId, group string) (string, error) {
	content, err := os.ReadFile(lc.configDir + "/" + dataId)
	if err != nil {
		return "", fmt.Errorf("[local] read config %s failed: %w", dataId, err)
	}
	return string(content), nil
}

// WatchConfig 本地模式不支持配置监听（返回 nil）
// 参数：dataId-配置ID, group-分组, callback-回调（不会触发）
// 返回：nil
func (lc *LocalConfigService) WatchConfig(dataId, group string, callback func(string)) error {
	// 本地文件不会自动变化，不需要监听
	return nil
}
