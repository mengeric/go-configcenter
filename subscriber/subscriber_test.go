package subscriber

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/xxx/go-configcenter/adapter"
	"github.com/xxx/go-configcenter/configmerge"
)

// ============================================================
// subscriber/subscriber.go 单元测试
// ============================================================

// MockConfigClient 模拟配置客户端
type MockConfigClient struct {
	configContent string
	watchCallback func(string)
}

func (m *MockConfigClient) GetConfig(dataId, group string) (string, error) {
	return m.configContent, nil
}

func (m *MockConfigClient) WatchConfig(dataId, group string, callback func(string)) error {
	m.watchCallback = callback
	return nil
}

func TestConfigSubscriber_Value(t *testing.T) {
	// 创建临时配置文件
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	baseConfig := `
app:
  name: "base"
`
	os.WriteFile("test-config/base.yaml", []byte(baseConfig), 0644)

	// 创建 merger
	merger := configmerge.NewMerger("test", []string{"test-config/base.yaml"})

	// 创建 subscriber
	sub := NewConfigSubscriber(merger, nil, nil)

	// 测试 Value
	value, err := sub.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	if len(value) == 0 {
		t.Fatal("Value returned empty")
	}
}

func TestConfigSubscriber_AddListener(t *testing.T) {
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	baseConfig := `
app:
  name: "base"
`
	os.WriteFile("test-config/base.yaml", []byte(baseConfig), 0644)

	merger := configmerge.NewMerger("test", []string{"test-config/base.yaml"})
	mockClient := &MockConfigClient{configContent: "key: value"}

	sub := NewConfigSubscriber(merger, mockClient, []adapter.RemoteConfig{
		{DataId: "shared.yaml", Group: "SHARED"},
	})

	// 注册 listener
	var mu sync.Mutex
	listenerCalled := false
	err := sub.AddListener(func() {
		mu.Lock()
		listenerCalled = true
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}

	// 验证 listener 被注册
	if len(sub.listeners) != 1 {
		t.Fatalf("listeners count = %d, want 1", len(sub.listeners))
	}

	// 模拟配置变化
	if mockClient.watchCallback != nil {
		mockClient.watchCallback("new: config")
	}

	// 等待 goroutine 完成
	time.Sleep(100 * time.Millisecond)

	// 验证 listener 被调用
	mu.Lock()
	called := listenerCalled
	mu.Unlock()
	if !called {
		t.Error("listener not called")
	}
}

func TestConfigSubscriber_AddListener_NoClient(t *testing.T) {
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	baseConfig := `
app:
  name: "base"
`
	os.WriteFile("test-config/base.yaml", []byte(baseConfig), 0644)

	merger := configmerge.NewMerger("test", []string{"test-config/base.yaml"})

	// 没有 configClient
	sub := NewConfigSubscriber(merger, nil, nil)

	// 注册 listener 应该成功
	err := sub.AddListener(func() {})
	if err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}

	if len(sub.listeners) != 1 {
		t.Fatalf("listeners count = %d, want 1", len(sub.listeners))
	}
}

func TestConfigSubscriber_MultipleListeners(t *testing.T) {
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	baseConfig := `
app:
  name: "base"
`
	os.WriteFile("test-config/base.yaml", []byte(baseConfig), 0644)

	merger := configmerge.NewMerger("test", []string{"test-config/base.yaml"})
	mockClient := &MockConfigClient{configContent: "key: value"}

	sub := NewConfigSubscriber(merger, mockClient, []adapter.RemoteConfig{
		{DataId: "shared.yaml", Group: "SHARED"},
	})

	// 注册多个 listener
	var mu sync.Mutex
	callCount := 0
	for i := 0; i < 3; i++ {
		sub.AddListener(func() {
			mu.Lock()
			callCount++
			mu.Unlock()
		})
	}

	// 模拟配置变化
	if mockClient.watchCallback != nil {
		mockClient.watchCallback("new: config")
	}

	// 等待 goroutine 完成
	time.Sleep(100 * time.Millisecond)

	// 验证所有 listener 被调用
	mu.Lock()
	count := callCount
	mu.Unlock()
	if count != 3 {
		t.Errorf("callCount = %d, want 3", count)
	}
}
