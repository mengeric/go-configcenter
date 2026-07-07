package goconfigcenter

// ============================================================
// argo.yaml 配置结构定义
// ============================================================

// ArgoConfig 是 argo.yaml 的完整结构
type ArgoConfig struct {
	// Registry 注册中心配置
	Registry RegistryConfig `yaml:"registry" json:"registry"`

	// Services 各服务配置（按服务名取）
	Services map[string]ServiceConfig `yaml:"services" json:"services"`
}

// RegistryConfig 注册中心配置
type RegistryConfig struct {
	// Type 注册中心类型（nacos / etcd / consul）
	Type string `yaml:"type" json:"type"`

	// Addr 注册中心地址（ip:port）
	Addr string `yaml:"addr" json:"addr"`

	// Namespace 命名空间（nacos 用，public 传空字符串）
	Namespace string `yaml:"namespace" json:"namespace"`

	// Group 分组
	Group string `yaml:"group" json:"group"`

	// Username Nacos 认证用户名（可选，为空则不认证）
	Username string `yaml:"username" json:"username"`

	// Password Nacos 认证密码
	Password string `yaml:"password" json:"password"`
}

// ServiceConfig 单个服务配置
type ServiceConfig struct {
	// Host 服务地址（本地模式用于服务发现）
	Host string `yaml:"host" json:"host"`

	// Port 服务端口（本地模式用于服务发现）
	Port uint64 `yaml:"port" json:"port"`

	// Weight 服务权重（默认10，范围0-100）
	Weight float64 `yaml:"weight" json:"weight"`

	// Local 本地配置文件列表（按顺序 merge，后面的覆盖前面的）
	Local []string `yaml:"local" json:"local"`

	// Remote Nacos 远程配置列表（按顺序 merge）
	Remote []RemoteConfig `yaml:"remote" json:"remote"`
}

// RemoteConfig Nacos 远程配置项
type RemoteConfig struct {
	// DataId Nacos 配置 ID
	DataId string `yaml:"dataId" json:"dataId"`

	// Group Nacos 分组
	Group string `yaml:"group" json:"group"`
}
