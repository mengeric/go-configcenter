package configmerge

import (
	"fmt"
	"os"
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

// Merge 合并所有配置并输出到临时文件
// 参数：outputDir-输出目录（默认当前目录）
// 返回：合并后的文件路径，错误信息
func (m *Merger) Merge(outputDir string) (string, error) {
	k := koanf.New(".")

	// 1. 加载本地配置文件（按顺序 merge，后面的覆盖前面的）
	for _, f := range m.localFiles {
		if err := k.Load(file.Provider(f), yaml.Parser()); err != nil {
			return "", fmt.Errorf("[configmerge] load local file %s failed: %w", f, err)
		}
	}

	// 2. 加载远程配置内容（按顺序 merge）
	for _, content := range m.remoteContents {
		if err := k.Load(rawbytes.Provider([]byte(content)), yaml.Parser()); err != nil {
			return "", fmt.Errorf("[configmerge] load remote config failed: %w", err)
		}
	}

	// 3. 序列化为 YAML
	data := k.All()
	yamlBytes, err := yamlv2.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("[configmerge] marshal yaml failed: %w", err)
	}

	// 4. 输出到文件
	if outputDir == "" {
		outputDir = "."
	}
	outputPath := filepath.Join(outputDir, fmt.Sprintf("merged-%s.yaml", m.serviceName))
	if err := os.WriteFile(outputPath, yamlBytes, 0644); err != nil {
		return "", fmt.Errorf("[configmerge] write merged file failed: %w", err)
	}

	return outputPath, nil
}

// MergeToBytes 合并所有配置并返回字节内容（不写文件）
// 返回：合并后的 YAML 字节内容，错误信息
func (m *Merger) MergeToBytes() ([]byte, error) {
	k := koanf.New(".")

	// 1. 加载本地配置文件
	for _, f := range m.localFiles {
		if err := k.Load(file.Provider(f), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("[configmerge] load local file %s failed: %w", f, err)
		}
	}

	// 2. 加载远程配置内容
	for _, content := range m.remoteContents {
		if err := k.Load(rawbytes.Provider([]byte(content)), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("[configmerge] load remote config failed: %w", err)
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
