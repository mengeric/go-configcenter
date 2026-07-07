package adapter

// ============================================================
// 注册中心适配器接口定义
// 所有注册中心（Nacos、etcd、consul）必须实现这些接口
// ============================================================

// NamingClient 服务注册与发现接口
type NamingClient interface {
	// Register 注册服务实例
	// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组, weight-权重（0使用默认值10）
	// 返回：成功/失败，错误信息
	Register(ip string, port uint64, serviceName, group string, weight float64) (bool, error)

	// Deregister 注销服务实例
	// 参数：ip-监听地址, port-监听端口, serviceName-服务名, group-分组
	// 返回：成功/失败，错误信息
	Deregister(ip string, port uint64, serviceName, group string) (bool, error)

	// Discover 发现服务实例（返回 ip:port）
	// 参数：serviceName-服务名, group-分组
	// 返回：地址字符串，错误信息
	Discover(serviceName, group string) (string, error)

	// Subscribe 订阅服务实例变化（动态更新）
	// 参数：serviceName-服务名, group-分组, callback-实例变化回调
	// 返回：错误信息
	Subscribe(serviceName, group string, callback func([]Instance)) error

	// Unsubscribe 取消订阅
	// 参数：serviceName-服务名, group-分组
	// 返回：错误信息
	Unsubscribe(serviceName, group string) error
}

// ConfigClient 配置管理接口
type ConfigClient interface {
	// GetConfig 拉取配置内容
	// 参数：dataId-配置ID, group-分组
	// 返回：配置内容字符串，错误信息
	GetConfig(dataId, group string) (string, error)

	// WatchConfig 监听配置变化
	// 参数：dataId-配置ID, group-分组, callback-配置变化回调（返回新内容）
	// 返回：错误信息
	WatchConfig(dataId, group string, callback func(string)) error
}

// RemoteConfig 远程配置项
type RemoteConfig struct {
	// DataId 配置 ID
	DataId string

	// Group 分组
	Group string
}
