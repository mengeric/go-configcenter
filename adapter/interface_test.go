package adapter

import (
	"testing"
)

// ============================================================
// adapter/interface.go 接口合规性测试
// ============================================================

// MockNamingClient 模拟命名客户端
type MockNamingClient struct {
	registered   bool
	deregistered bool
	discoverAddr string
}

func (m *MockNamingClient) Register(ip string, port uint64, serviceName, group string, weight float64) (bool, error) {
	m.registered = true
	return true, nil
}

func (m *MockNamingClient) Deregister(ip string, port uint64, serviceName, group string) (bool, error) {
	m.deregistered = true
	return true, nil
}

func (m *MockNamingClient) Discover(serviceName, group string) (string, error) {
	return m.discoverAddr, nil
}

func (m *MockNamingClient) Subscribe(serviceName, group string, callback func([]Instance)) error {
	return nil
}

func (m *MockNamingClient) Unsubscribe(serviceName, group string) error {
	return nil
}

// MockConfigClient 模拟配置客户端
type MockConfigClient struct {
	configContent string
	watchCalled   bool
}

func (m *MockConfigClient) GetConfig(dataId, group string) (string, error) {
	return m.configContent, nil
}

func (m *MockConfigClient) WatchConfig(dataId, group string, callback func(string)) error {
	m.watchCalled = true
	return nil
}

func TestInterfaceCompliance(t *testing.T) {
	// 验证 MockNamingClient 实现了 NamingClient 接口
	var _ NamingClient = (*MockNamingClient)(nil)

	// 验证 MockConfigClient 实现了 ConfigClient 接口
	var _ ConfigClient = (*MockConfigClient)(nil)
}

func TestMockNamingClient(t *testing.T) {
	mock := &MockNamingClient{discoverAddr: "192.168.1.1:8888"}

	// 测试 Register
	ok, err := mock.Register("0.0.0.0", 8888, "galaxy", "DEFAULT_GROUP", 10)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !ok {
		t.Fatal("Register returned false")
	}
	if !mock.registered {
		t.Fatal("Register not called")
	}

	// 测试 Deregister
	ok, err = mock.Deregister("0.0.0.0", 8888, "galaxy", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Deregister failed: %v", err)
	}
	if !ok {
		t.Fatal("Deregister returned false")
	}
	if !mock.deregistered {
		t.Fatal("Deregister not called")
	}

	// 测试 Discover
	addr, err := mock.Discover("phoenix", "DEFAULT_GROUP")
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if addr != "192.168.1.1:8888" {
		t.Errorf("Discover = %s, want 192.168.1.1:8888", addr)
	}
}

func TestMockConfigClient(t *testing.T) {
	mock := &MockConfigClient{configContent: "key: value"}

	// 测试 GetConfig
	content, err := mock.GetConfig("shared.yaml", "SHARED")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if content != "key: value" {
		t.Errorf("GetConfig = %s, want key: value", content)
	}

	// 测试 WatchConfig
	err = mock.WatchConfig("shared.yaml", "SHARED", func(data string) {})
	if err != nil {
		t.Fatalf("WatchConfig failed: %v", err)
	}
	if !mock.watchCalled {
		t.Fatal("WatchConfig not called")
	}
}

func TestInstance(t *testing.T) {
	inst := Instance{
		Ip:       "192.168.1.1",
		Port:     8888,
		Weight:   10,
		Healthy:  true,
		Enable:   true,
		Metadata: map[string]string{"idc": "shanghai"},
	}

	if inst.Ip != "192.168.1.1" {
		t.Errorf("Ip = %s, want 192.168.1.1", inst.Ip)
	}
	if inst.Port != 8888 {
		t.Errorf("Port = %d, want 8888", inst.Port)
	}
	if inst.Weight != 10 {
		t.Errorf("Weight = %f, want 10", inst.Weight)
	}
	if !inst.Healthy {
		t.Error("Healthy should be true")
	}
	if !inst.Enable {
		t.Error("Enable should be true")
	}
	if inst.Metadata["idc"] != "shanghai" {
		t.Errorf("Metadata[idc] = %s, want shanghai", inst.Metadata["idc"])
	}
}
