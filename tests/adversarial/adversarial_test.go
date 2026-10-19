package adversarial

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/mengeric/go-configcenter/adapter"
	"github.com/mengeric/go-configcenter/internal/netutil"
	"github.com/mengeric/go-configcenter/local"
)

// TestIPDetectionAdversarial 验证极端/恶意环境下的 IP 推导健壮性
func TestIPDetectionAdversarial(t *testing.T) {
	t.Run("非法环境变量注入时必须自动过滤并 fallback", func(t *testing.T) {
		// 注入非法/恶意的环境变量
		maliciousInputs := []string{
			"127.0.0.1",
			"0.0.0.0",
			"999.999.999.999",
			"192.168.1.1/24",
			"http://10.0.0.1",
			"::1",
			"abc.xyz",
			"",
		}

		for _, input := range maliciousInputs {
			os.Setenv("HOST_IP", input)
			if netutil.IsValidIPv4(input) {
				t.Errorf("expected %q to be marked invalid, but got true", input)
			}
		}
		os.Unsetenv("HOST_IP")
	})

	t.Run("不可达/格式异常的目标注册中心地址嗅探兜底", func(t *testing.T) {
		// 测试异常目标地址不会导致 panic 或崩溃
		weirdTargets := []string{
			"invalid-hostname-404-not-found:8848",
			"256.256.256.256:8848",
			"",
			"::::8848",
		}

		for _, target := range weirdTargets {
			// ResolveRegisterIP 应平滑 fallback 至物理网卡或返回错误，绝不允许 panic
			_, _ = netutil.ResolveRegisterIP(target)
		}
	})
}

// TestDynamicDiscoveryAdversarial 验证极端/对抗场景下的动态服务发现健壮性
func TestDynamicDiscoveryAdversarial(t *testing.T) {
	t.Run("下游实例全下线时Discover禁止panic且明确报错", func(t *testing.T) {
		svc := local.NewLocalNamingService(map[string]local.ServiceAddr{
			"dying-service": {Host: "10.0.0.1", Port: 8080},
		})

		// 模拟服务下线
		_, err := svc.Deregister("10.0.0.1", 8080, "dying-service", "DEFAULT_GROUP")
		if err != nil {
			t.Fatalf("Deregister failed: %v", err)
		}

		// 再次 Discover 绝不允许 panic
		addr, err := svc.Discover("dying-service", "DEFAULT_GROUP")
		if err == nil {
			t.Errorf("expected error when all instances down, but got addr: %s", addr)
		}

		// DiscoverAll 绝不允许 panic
		instances, err := svc.DiscoverAll("dying-service", "DEFAULT_GROUP")
		if err == nil {
			t.Errorf("expected error from DiscoverAll when all instances down, got: %+v", instances)
		}
	})

	t.Run("恶意/疏忽传入nil回调时Subscribe禁止panic", func(t *testing.T) {
		svc := local.NewLocalNamingService(map[string]local.ServiceAddr{
			"dummy": {Host: "127.0.0.1", Port: 80},
		})

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Subscribe with nil callback panicked: %v", r)
			}
		}()

		// 传入 nil callback
		if err := svc.Subscribe("dummy", "DEFAULT_GROUP", nil); err != nil {
			t.Fatalf("Subscribe nil callback returned unexpected error: %v", err)
		}

		// 触发更新，验证 notifyListeners 对 nil 的抵抗力
		_, _ = svc.Register("127.0.0.1", 81, "dummy", "DEFAULT_GROUP", 10)
	})

	t.Run("高并发订阅与动态注册注销竞争测试", func(t *testing.T) {
		svc := local.NewLocalNamingService(map[string]local.ServiceAddr{})
		serviceName := "concurrent-service"
		concurrency := 30

		var wg sync.WaitGroup
		wg.Add(concurrency * 3)

		// 1. 并发订阅者
		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				_ = svc.Subscribe(serviceName, "DEFAULT_GROUP", func(insts []adapter.Instance) {
					// 纯读取，验证线程安全性
					_ = len(insts)
				})
			}()
		}

		// 2. 并发注册者
		for i := 0; i < concurrency; i++ {
			idx := i
			go func() {
				defer wg.Done()
				ip := fmt.Sprintf("192.168.1.%d", idx+1)
				_, _ = svc.Register(ip, uint64(8000+idx), serviceName, "DEFAULT_GROUP", 10)
			}()
		}

		// 3. 并发发现者
		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				_, _ = svc.Discover(serviceName, "DEFAULT_GROUP")
				_, _ = svc.DiscoverAll(serviceName, "DEFAULT_GROUP")
			}()
		}

		wg.Wait()
	})
}
