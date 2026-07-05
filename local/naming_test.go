package local

import (
	"testing"

	"github.com/xxx/go-configcenter/adapter"
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
	ok, err := svc.Register("192.168.1.2", 9999, "galaxy", "DEFAULT_GROUP")
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
	svc.Register("192.168.1.2", 9999, "galaxy", "DEFAULT_GROUP")

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
		"phoenix": {Host: "192.168.1.1", Port: 10011},
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

	// 取消订阅（本地模式无操作）
	err := svc.Unsubscribe("phoenix", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}
}
