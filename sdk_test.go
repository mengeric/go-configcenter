package goconfigcenter

import (
	"os"
	"testing"
)

// TestSDK_RegisterAutoIP 测试 Register 传入空 IP 或 0.0.0.0 时的智能探测行为
func TestSDK_RegisterAutoIP(t *testing.T) {
	// 准备包含本地模式的临时配置文件
	yamlContent := `
registry:
  type: "local"
services:
  test-service:
    host: "0.0.0.0"
    port: 8080
    local: []
`
	tmpFile := "test-auto-ip.yaml"
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	sdk, err := Init(tmpFile, "test-service")
	if err != nil {
		t.Fatalf("init sdk failed: %v", err)
	}

	// 1. 测试显式注入环境变量时，Register("") 或 Register("0.0.0.0") 优先采用环境变量 IP
	expectedEnvIP := "10.10.10.10"
	os.Setenv("HOST_IP", expectedEnvIP)
	defer os.Unsetenv("HOST_IP")

	if err := sdk.Register("0.0.0.0", 8080); err != nil {
		t.Fatalf("register with 0.0.0.0 failed: %v", err)
	}
	if sdk.registeredIP != expectedEnvIP {
		t.Errorf("registeredIP = %s, want %s", sdk.registeredIP, expectedEnvIP)
	}

	// 2. 测试 QuickRegister 自动处理 host 为 0.0.0.0 的场景
	if err := sdk.QuickRegister(); err != nil {
		t.Fatalf("quick register failed: %v", err)
	}
	if sdk.registeredIP != expectedEnvIP {
		t.Errorf("quick registeredIP = %s, want %s", sdk.registeredIP, expectedEnvIP)
	}

	// 3. 注销
	if err := sdk.Deregister(); err != nil {
		t.Fatalf("deregister failed: %v", err)
	}
}
