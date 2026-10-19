package local

import (
	"testing"

	"github.com/mengeric/go-configcenter/adapter"
)

// ============================================================
// local/naming.go 单元测试
// ============================================================

func TestLocalNamingService_Register(t *testing.T) {
	services := map[string]ServiceAddr{
		"phoenix": {Host: "192.168.1.1", Port: 10011},
	}
	svc := NewLocalNamingService(services)

	// 注册新服务
	ok, err := svc.Register("192.168.1.2", 9999, "galaxy", "DEFAULT_GROUP", 10)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !ok {
		t.Fatal("Register returned false")
	}

	// 验证注册成功
	addr, err := svc.Discover("galaxy", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if addr != "192.168.1.2:9999" {
		t.Errorf("Discover = %s, want 192.168.1.2:9999", addr)
	}
}

func TestLocalNamingService_Deregister(t *testing.T) {
	services := map[string]ServiceAddr{
		"phoenix": {Host: "192.168.1.1", Port: 10011},
	}
	svc := NewLocalNamingService(services)

	// 注册
	svc.Register("192.168.1.2", 9999, "galaxy", "DEFAULT_GROUP", 10)

	// 注销
	ok, err := svc.Deregister("192.168.1.2", 9999, "galaxy", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Deregister failed: %v", err)
	}
	if !ok {
		t.Fatal("Deregister returned false")
	}

	// 验证注销成功
	_, err = svc.Discover("galaxy", "DEFAULT_GROUP")
	if err == nil {
		t.Fatal("Discover should failed after Deregister")
	}
}

func TestLocalNamingService_Discover(t *testing.T) {
	services := map[string]ServiceAddr{
		"phoenix":  {Host: "192.168.1.1", Port: 10011},
		"stargate": {Host: "192.168.1.2", Port: 8800},
	}
	svc := NewLocalNamingService(services)

	// 发现 phoenix
	addr, err := svc.Discover("phoenix", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Discover phoenix failed: %v", err)
	}
	if addr != "192.168.1.1:10011" {
		t.Errorf("Discover phoenix = %s, want 192.168.1.1:10011", addr)
	}

	// 发现 stargate
	addr, err = svc.Discover("stargate", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Discover stargate failed: %v", err)
	}
	if addr != "192.168.1.2:8800" {
		t.Errorf("Discover stargate = %s, want 192.168.1.2:8800", addr)
	}

	// 发现不存在的服务
	_, err = svc.Discover("nonexistent", "DEFAULT_GROUP")
	if err == nil {
		t.Fatal("Discover nonexistent should failed")
	}

	// 测试 DiscoverAll
	instances, err := svc.DiscoverAll("phoenix", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("DiscoverAll phoenix failed: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 10011 {
		t.Errorf("DiscoverAll phoenix mismatch: %+v", instances)
	}
}

func TestLocalNamingService_Subscribe(t *testing.T) {
	services := map[string]ServiceAddr{
		"phoenix": {Host: "192.168.1.1", Port: 10011},
	}
	svc := NewLocalNamingService(services)

	// 订阅
	var callbackInstances []adapter.Instance
	err := svc.Subscribe("phoenix", "DEFAULT_GROUP", func(instances []adapter.Instance) {
		callbackInstances = instances
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// 本地模式立即回调
	if len(callbackInstances) != 1 {
		t.Fatalf("callback instances count = %d, want 1", len(callbackInstances))
	}
	if callbackInstances[0].Ip != "192.168.1.1" {
		t.Errorf("callback instance ip = %s, want 192.168.1.1", callbackInstances[0].Ip)
	}
	if callbackInstances[0].Port != 10011 {
		t.Errorf("callback instance port = %d, want 10011", callbackInstances[0].Port)
	}
}

func TestLocalNamingService_Unsubscribe(t *testing.T) {
	svc := NewLocalNamingService(map[string]ServiceAddr{})

	// 取消订阅
	err := svc.Unsubscribe("phoenix", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}
}

func TestLocalNamingService_DynamicNotification(t *testing.T) {
	svc := NewLocalNamingService(map[string]ServiceAddr{})

	var received []adapter.Instance
	callCount := 0
	err := svc.Subscribe("dynamic-svc", "DEFAULT_GROUP", func(instances []adapter.Instance) {
		received = instances
		callCount++
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// 此时无实例，回调未触发
	if callCount != 0 {
		t.Fatalf("callCount = %d, want 0", callCount)
	}

	// 动态注册新实例
	_, err = svc.Register("192.168.10.5", 7001, "dynamic-svc", "DEFAULT_GROUP", 20)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if callCount != 1 || len(received) != 1 || received[0].Port != 7001 {
		t.Fatalf("expected 1 instance after register, got: %+v (callCount=%d)", received, callCount)
	}

	// 动态注销
	_, err = svc.Deregister("192.168.10.5", 7001, "dynamic-svc", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Deregister failed: %v", err)
	}

	if callCount != 2 || len(received) != 0 {
		t.Fatalf("expected 0 instance after deregister, got: %+v (callCount=%d)", received, callCount)
	}
}
