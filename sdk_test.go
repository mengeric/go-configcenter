package goconfigcenter

import (
	"os"
	"testing"

	"github.com/mengeric/go-configcenter/adapter"
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

// TestSDK_DynamicDiscoveryAndSubscribe 测试 SDK 服务发现与动态订阅完整生命周期
func TestSDK_DynamicDiscoveryAndSubscribe(t *testing.T) {
	yamlContent := `
registry:
  type: "local"
services:
  gateway:
    host: "127.0.0.1"
    port: 8080
    local: []
  order-service:
    host: "192.168.1.10"
    port: 9001
    local: []
`
	tmpFile := "test-discovery.yaml"
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	sdk, err := Init(tmpFile, "gateway")
	if err != nil {
		t.Fatalf("init sdk failed: %v", err)
	}

	// 1. DiscoverAll
	instances, err := sdk.DiscoverAll("order-service")
	if err != nil {
		t.Fatalf("DiscoverAll failed: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 9001 {
		t.Fatalf("DiscoverAll unexpected instances: %+v", instances)
	}

	// 2. Subscribe 初始快照回放
	var received []adapter.Instance
	callCount := 0
	err = sdk.Subscribe("order-service", func(insts []adapter.Instance) {
		received = insts
		callCount++
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if callCount != 1 || len(received) != 1 {
		t.Fatalf("expected initial callback with 1 instance, got count=%d len=%d", callCount, len(received))
	}

	// 3. Unsubscribe
	if err := sdk.Unsubscribe("order-service"); err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}
}
