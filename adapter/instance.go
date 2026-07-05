package adapter

// ============================================================
// 服务实例结构体
// ============================================================

// Instance 服务实例信息
type Instance struct {
	// Ip 服务实例 IP 地址
	Ip string `json:"ip"`

	// Port 服务实例端口
	Port uint64 `json:"port"`

	// Weight 实例权重（用于负载均衡）
	Weight float64 `json:"weight"`

	// Healthy 实例是否健康
	Healthy bool `json:"healthy"`

	// Enable 实例是否启用
	Enable bool `json:"enable"`

	// Metadata 实例元数据
	Metadata map[string]string `json:"metadata,omitempty"`
}
