package goconfigcenter

// ============================================================
// SDK Option 模式配置
// ============================================================

// Option SDK 配置选项函数
type Option func(*options)

// options SDK 内部配置
type options struct {
	// outputDir 合并配置输出目录（默认当前目录）
	outputDir string
}

// WithOutputDir 设置合并配置输出目录
// 参数：dir-输出目录路径
// 返回：Option 函数
func WithOutputDir(dir string) Option {
	return func(o *options) {
		o.outputDir = dir
	}
}
