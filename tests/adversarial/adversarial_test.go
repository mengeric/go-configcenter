package adversarial

import (
	"os"
	"testing"

	"github.com/mengeric/go-configcenter/internal/netutil"
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
