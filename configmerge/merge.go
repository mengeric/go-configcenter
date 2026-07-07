package configmerge

import (
	"fmt"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
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
	// 使用 yamlv2 直接解析合并，保留嵌套结构
	merged := make(map[string]interface{})

	// 1. 加载远程配置（优先级低）
	for _, content := range m.remoteContents {
		var cfg map[string]interface{}
		if err := yamlv2.Unmarshal([]byte(content), &cfg); err != nil {
			return nil, fmt.Errorf("[configmerge] unmarshal remote config failed: %w", err)
		}
		mergeMap(merged, cfg)
	}

	// 2. 加载本地配置文件（优先级高，覆盖远程）
	for _, f := range m.localFiles {
		k := koanf.New(".")
		if err := k.Load(file.Provider(f), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("[configmerge] load local file %s failed: %w", f, err)
		}
		data, err := k.Marshal(yaml.Parser())
		if err != nil {
			return nil, fmt.Errorf("[configmerge] marshal local file %s failed: %w", f, err)
		}
		var cfg map[string]interface{}
		if err := yamlv2.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("[configmerge] unmarshal local file %s failed: %w", f, err)
		}
		mergeMap(merged, cfg)
	}

	return yamlv2.Marshal(merged)
}

// mergeMap 递归合并（原地修改 dst）
func mergeMap(dst, src interface{}) {
	switch srcMap := src.(type) {
	case map[string]interface{}:
		switch dstMap := dst.(type) {
		case map[string]interface{}:
			mergeStrMap(dstMap, srcMap)
		case map[interface{}]interface{}:
			mergeInterfaceMap(dstMap, srcMap)
		}
	case map[interface{}]interface{}:
		switch dstMap := dst.(type) {
		case map[string]interface{}:
			for k, v := range srcMap {
				mergeMap(dstMap[fmt.Sprintf("%v", k)], v)
				dstMap[fmt.Sprintf("%v", k)] = v
			}
		case map[interface{}]interface{}:
			for k, v := range srcMap {
				if dstVal, ok := dstMap[k]; ok {
					if isMap(dstVal) && isMap(v) {
						mergeMap(dstVal, v)
						continue
					}
				}
				dstMap[k] = v
			}
		}
	}
}

// mergeStrMap 合并两个 map[string]interface{}
func mergeStrMap(dst, src map[string]interface{}) {
	for k, v := range src {
		if dstVal, ok := dst[k]; ok {
			if isMap(dstVal) && isMap(v) {
				mergeMap(dstVal, v)
				continue
			}
		}
		dst[k] = v
	}
}

// mergeInterfaceMap 将 src[string] 合并到 dst[interface{}]
func mergeInterfaceMap(dst map[interface{}]interface{}, src map[string]interface{}) {
	for k, v := range src {
		if dstVal, ok := dst[k]; ok {
			if isMap(dstVal) && isMap(v) {
				mergeMap(dstVal, v)
				continue
			}
		}
		dst[k] = v
	}
}

// isMap 检查值是否是 map 类型
func isMap(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, map[interface{}]interface{}:
		return true
	}
	return false
}

// toStrMap 将 map[interface{}]interface{} 转换为 map[string]interface{}
// yaml.v2 解析 YAML 时会返回 interface{} 类型的 key，需要转换为 string
var toStrMap = func(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for key, val := range m {
			result[fmt.Sprintf("%v", key)] = val
		}
		return result, true
	default:
		return nil, false
	}
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
