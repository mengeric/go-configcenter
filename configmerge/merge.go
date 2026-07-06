package configmerge

import (
	"fmt"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	yamlv2 "gopkg.in/yaml.v2"
)

// ============================================================
// 配置合并模块
// 合并本地配置文件 + Nacos 远程配置，输出合并后的临时文件
// ============================================================

// Merger 配置合并器
type Merger struct {
	// serviceName 服务名（用于生成临时文件名）
	serviceName string
	// localFiles 本地配置文件列表
	localFiles []string
	// remoteContents 远程配置内容列表
	remoteContents []string
}

// NewMerger 创建配置合并器
// 参数：serviceName-服务名, localFiles-本地配置文件路径列表
// 返回：合并器实例
func NewMerger(serviceName string, localFiles []string) *Merger {
	return &Merger{
		serviceName: serviceName,
		localFiles:  localFiles,
	}
}

// AddRemote 添加远程配置内容
// 参数：content-远程配置内容字符串
func (m *Merger) AddRemote(content string) {
	if content != "" {
		m.remoteContents = append(m.remoteContents, content)
	}
}

// MergeToBytes 合并所有配置并返回字节内容（不写文件）
// 合并顺序：远程配置 → 本地配置（本地覆盖远程）
// 返回：合并后的 YAML 字节内容，错误信息
func (m *Merger) MergeToBytes() ([]byte, error) {
	k := koanf.New(".")

	// 1. 加载远程配置内容（优先级低）
	for _, content := range m.remoteContents {
		if err := k.Load(rawbytes.Provider([]byte(content)), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("[configmerge] load remote config failed: %w", err)
		}
	}

	// 2. 加载本地配置文件（优先级高，覆盖远程）
	for _, f := range m.localFiles {
		if err := k.Load(file.Provider(f), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("[configmerge] load local file %s failed: %w", f, err)
		}
	}

	return yamlv2.Marshal(k.All())
}

// ConfigType 根据本地配置文件扩展名判断配置类型
// 返回："yaml"、"json"、"toml"
func (m *Merger) ConfigType() string {
	if len(m.localFiles) == 0 {
		return "yaml"
	}
	ext := filepath.Ext(m.localFiles[0])
	switch ext {
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	default:
		return "yaml"
	}
}
