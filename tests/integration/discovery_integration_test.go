package integration

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	goconfigcenter "github.com/mengeric/go-configcenter"
	"github.com/mengeric/go-configcenter/adapter"
)

// ============================================================
// 端到端服务注册、动态发现与配置共存集成测试
// ============================================================

// createTempConfigFile 辅助函数：快速生成集成测试所需的 argo.yaml 配置文件
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "argo.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	return path
}

// TestIntegration_ServiceDiscoveryLifecycle 测试端到端动态注册、发现、订阅与注销完整生命周期
func TestIntegration_ServiceDiscoveryLifecycle(t *testing.T) {
	configYAML := `
registry:
  type: "local"
  scheme: "http"
services:
  phoenix:
    host: "10.0.1.10"
    port: 9001
    weight: 20
    local: []
`
	configPath := createTempConfigFile(t, configYAML)

	// 1. 初始化 phoenix 服务 SDK
	sdk, err := goconfigcenter.Init(configPath, "phoenix")
	if err != nil {
		t.Fatalf("init phoenix SDK failed: %v", err)
	}

	// 2. 验证首次 Discover 发现自身或预配置节点
	addr, err := sdk.Discover("phoenix")
	if err != nil {
		t.Fatalf("discover phoenix failed: %v", err)
	}
	if addr != "http://10.0.1.10:9001" {
		t.Errorf("discover addr = %s, want http://10.0.1.10:9001", addr)
	}

	// 3. 订阅动态变更（验证初始快照是否立即回放）
	var received []adapter.Instance
	var notifyCount int
	var lock sync.Mutex

	err = sdk.Subscribe("phoenix", func(instances []adapter.Instance) {
		lock.Lock()
		defer lock.Unlock()
		received = instances
		notifyCount++
	})
	if err != nil {
		t.Fatalf("subscribe phoenix failed: %v", err)
	}

	lock.Lock()
	if notifyCount != 1 || len(received) != 1 || received[0].Port != 9001 {
		t.Fatalf("expected initial snapshot callback: count=%d, instances=%+v", notifyCount, received)
	}
	lock.Unlock()

	// 4. 服务动态重启上报新地址 (10.0.1.20:9002)
	if err := sdk.Register("10.0.1.20", 9002); err != nil {
		t.Fatalf("re-register failed: %v", err)
	}

	// 5. 验证端到端自动收到动态变更通知
	lock.Lock()
	if notifyCount != 2 || len(received) != 1 || received[0].Port != 9002 {
		t.Fatalf("expected dynamic update callback after re-register: count=%d, instances=%+v", notifyCount, received)
	}
	lock.Unlock()

	// 6. 验证再次 Discover 实时返回最新地址
	addr, err = sdk.Discover("phoenix")
	if err != nil {
		t.Fatalf("discover updated phoenix failed: %v", err)
	}
	if addr != "http://10.0.1.20:9002" {
		t.Errorf("discover updated addr = %s, want http://10.0.1.20:9002", addr)
	}

	// 7. 服务优雅注销
	if err := sdk.Deregister(); err != nil {
		t.Fatalf("deregister failed: %v", err)
	}

	// 8. 验证收到注销空切片通知，且再次 Discover 明确报错（全下线防护）
	lock.Lock()
	if notifyCount != 3 || len(received) != 0 {
		t.Fatalf("expected empty instances notification after deregister: count=%d, instances=%+v", notifyCount, received)
	}
	lock.Unlock()

	_, err = sdk.Discover("phoenix")
	if err == nil {
		t.Error("expected error when discovering deregistered service, got nil")
	}

	// 9. 取消订阅
	if err := sdk.Unsubscribe("phoenix"); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
}

// TestIntegration_DiscoveryAndConfigCoexistence 测试动态服务发现与动态配置热更新并存时的协同安全性
func TestIntegration_DiscoveryAndConfigCoexistence(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "app.yaml")
	appYAML := `
server:
  name: "composite-test"
  timeout: 5s
`
	if err := os.WriteFile(appPath, []byte(appYAML), 0644); err != nil {
		t.Fatalf("create app.yaml failed: %v", err)
	}

	configYAML := fmt.Sprintf(`
registry:
  type: "local"
services:
  composite-service:
    host: "127.0.0.1"
    port: 8888
    local:
      - "%s"
  remote-peer:
    host: "10.0.2.1"
    port: 7001
    local: []
`, appPath)

	configPath := filepath.Join(dir, "argo.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("create argo.yaml failed: %v", err)
	}

	sdk, err := goconfigcenter.Init(configPath, "composite-service")
	if err != nil {
		t.Fatalf("init SDK failed: %v", err)
	}

	// 1. 测试配置订阅器读取
	sub := sdk.Subscriber()
	val, err := sub.Value()
	if err != nil {
		t.Fatalf("subscriber Value failed: %v", err)
	}
	if len(val) == 0 {
		t.Fatal("config value is empty")
	}

	// 2. 并发进行配置监听注册与服务发现
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			addr, dErr := sdk.Discover("remote-peer")
			if dErr != nil || addr != "http://10.0.2.1:7001" {
				t.Errorf("discover failed in goroutine: %v, addr=%s", dErr, addr)
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = sub.AddListener(func() {})
			_, _ = sub.Value()
		}
	}()

	wg.Wait()
}

// TestIntegration_RealNacosEnvironment 真实 Nacos 环境联调集成用例（按需执行）
// 当设置环境变量 INTEGRATION_NACOS_ADDR (如 "127.0.0.1:8848") 时触发真实网络集成，否则跳过
func TestIntegration_RealNacosEnvironment(t *testing.T) {
	nacosAddr := os.Getenv("INTEGRATION_NACOS_ADDR")
	if nacosAddr == "" {
		t.Skip("skipping real Nacos integration test (set INTEGRATION_NACOS_ADDR=ip:port to enable)")
	}

	serviceName := fmt.Sprintf("nacos-integ-%d", time.Now().UnixNano()%10000)
	configYAML := fmt.Sprintf(`
registry:
  type: "nacos"
  addr: "%s"
  namespace: ""
  group: "DEFAULT_GROUP"
services:
  %s:
    host: "127.0.0.1"
    port: 19876
    weight: 20
    local: []
`, nacosAddr, serviceName)

	configPath := createTempConfigFile(t, configYAML)
	sdk, err := goconfigcenter.Init(configPath, serviceName)
	if err != nil {
		t.Fatalf("init real nacos SDK failed: %v", err)
	}

	// 真实注册
	if err := sdk.Register("127.0.0.1", 19876); err != nil {
		t.Fatalf("real nacos register failed: %v", err)
	}
	defer sdk.Deregister()

	// 真实自动订阅与发现
	addr, err := sdk.Discover(serviceName)
	if err != nil {
		t.Fatalf("real nacos discover failed: %v", err)
	}
	expected := "http://127.0.0.1:19876"
	if addr != expected {
		t.Errorf("real nacos discover addr = %s, want %s", addr, expected)
	}
}

// TestIntegration_MultipleServicesDiscovery 测试多个服务同时注册并相互发现的拓扑网络
func TestIntegration_MultipleServicesDiscovery(t *testing.T) {
	configYAML := `
registry:
  type: "local"
services:
  service-a:
    host: "10.0.0.1"
    port: 8001
    local: []
  service-b:
    host: "10.0.0.2"
    port: 8002
    local: []
  service-c:
    host: "10.0.0.3"
    port: 8003
    local: []
`
	configPath := createTempConfigFile(t, configYAML)

	sdkA, err := goconfigcenter.Init(configPath, "service-a")
	if err != nil {
		t.Fatalf("init sdkA failed: %v", err)
	}

	// A 发现 B 和 C
	addrB, err := sdkA.Discover("service-b")
	if err != nil || addrB != "http://10.0.0.2:8002" {
		t.Fatalf("sdkA discover B failed: %v, addr=%s", err, addrB)
	}

	addrC, err := sdkA.Discover("service-c")
	if err != nil || addrC != "http://10.0.0.3:8003" {
		t.Fatalf("sdkA discover C failed: %v, addr=%s", err, addrC)
	}

	// 验证 DiscoverAll 数量与端口对应
	instsB, err := sdkA.DiscoverAll("service-b")
	if err != nil || len(instsB) != 1 || instsB[0].Port != 8002 {
		t.Fatalf("sdkA discoverAll B failed: %v, insts=%+v", err, instsB)
	}
}

// 确保测试引入 math 库不告警
var _ = math.Abs
