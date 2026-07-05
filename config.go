package goconfigcenter

// ============================================================
// argo.yaml 配置结构定义
// ============================================================

// ArgoConfig 是 argo.yaml 的完整结构
type ArgoConfig struct {
	// Nacos 连接配置（可选，为空则纯本地模式）
	Nacos *NacosConfig `yaml:"nacos" json:"nacos"`

	// 当前服务注册信息
	Service ServiceInfo `yaml:"service" json:"service"`

	// 本地配置文件列表（按顺序 merge，后面的覆盖前面的）
	Local []string `yaml:"local" json:"local"`

	// Nacos 远程配置列表（按顺序 merge）
	Remote []RemoteConfig `yaml:"remote" json:"remote"`

	// 无 Nacos 时的本地服务地址表
	Services map[string]ServiceAddr `yaml:"services" json:"services"`
}

// NacosConfig Nacos 连接配置
type NacosConfig struct {
	// Nacos 地址（ip:port）
	Addr string `yaml:"addr" json:"addr"`

	// 命名空间（public 传空字符串）
	Namespace string `yaml:"namespace" json:"namespace"`

	// 分组
	Group string `yaml:"group" json:"group"`
}

// ServiceInfo 当前服务信息（用于注册）
type ServiceInfo struct {
	// 服务名称（如 galaxy、phoenix）
	Name string `yaml:"name" json:"name"`

	// 监听地址
	Host string `yaml:"host" json:"host"`

	// 监听端口
	Port uint64 `yaml:"port" json:"port"`
}

// RemoteConfig Nacos 远程配置项
type RemoteConfig struct {
	// Nacos DataId
	DataId string `yaml:"dataId" json:"dataId"`

	// Nacos Group
	Group string `yaml:"group" json:"group"`
}

// ServiceAddr 本地服务地址（无 Nacos 时使用）
type ServiceAddr struct {
	// 服务地址
	Host string `yaml:"host" json:"host"`

	// 服务端口
	Port uint64 `yaml:"port" json:"port"`
}
