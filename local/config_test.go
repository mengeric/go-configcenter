package local

import (
	"os"
	"testing"
)

// ============================================================
// local/config.go 单元测试
// ============================================================

func TestLocalConfigService_GetConfig(t *testing.T) {
	// 创建临时配置文件
	configDir := "test-config"
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	configContent := `
redis:
  host: "192.168.110.164"
  port: 6379
mysql:
  host: "192.168.110.164"
  port: 3306
`
	os.WriteFile(configDir+"/shared.yaml", []byte(configContent), 0644)

	// 测试读取
	svc := NewLocalConfigService(configDir)
	content, err := svc.GetConfig("shared.yaml", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if content != configContent {
		t.Errorf("GetConfig content mismatch:\ngot: %s\nwant: %s", content, configContent)
	}
}

func TestLocalConfigService_GetConfig_NotFound(t *testing.T) {
	configDir := "test-config-nonexistent"
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	svc := NewLocalConfigService(configDir)
	_, err := svc.GetConfig("nonexistent.yaml", "DEFAULT_GROUP")
	if err == nil {
		t.Fatal("GetConfig should failed for nonexistent file")
	}
}

func TestLocalConfigService_WatchConfig(t *testing.T) {
	svc := NewLocalConfigService(".")

	// 本地模式 WatchConfig 应该返回 nil
	err := svc.WatchConfig("shared.yaml", "DEFAULT_GROUP", func(data string) {
		t.Fatal("WatchConfig callback should not be called")
	})
	if err != nil {
		t.Fatalf("WatchConfig failed: %v", err)
	}
}
