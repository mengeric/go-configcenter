package netutil

import (
	"os"
	"testing"
)

// TestIsValidIPv4 测试 IPv4 有效性校验
func TestIsValidIPv4(t *testing.T) {
	cases := []struct {
		ip    string
		valid bool
	}{
		{"192.168.1.100", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"0.0.0.0", false},
		{"127.0.0.1", false},
		{"::1", false},
		{"invalid-ip", false},
		{"", false},
	}

	for _, c := range cases {
		got := IsValidIPv4(c.ip)
		if got != c.valid {
			t.Errorf("IsValidIPv4(%q) = %v; want %v", c.ip, got, c.valid)
		}
	}
}

// TestResolveRegisterIPWithEnv 测试环境变量优先级探测
func TestResolveRegisterIPWithEnv(t *testing.T) {
	expectedIP := "10.200.1.20"
	os.Setenv("HOST_IP", expectedIP)
	defer os.Unsetenv("HOST_IP")

	ip, err := ResolveRegisterIP("127.0.0.1:8848")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != expectedIP {
		t.Fatalf("got ip %s, want %s", ip, expectedIP)
	}
}

// TestResolveRegisterIPDefault 测试默认网卡/目标探测兜底
func TestResolveRegisterIPDefault(t *testing.T) {
	os.Unsetenv("HOST_IP")
	os.Unsetenv("POD_IP")
	os.Unsetenv("REGISTER_IP")

	ip, err := ResolveRegisterIP("8.8.8.8:53")
	if err != nil {
		t.Logf("resolve ip warning (may have no active network interface): %v", err)
		return
	}
	if !IsValidIPv4(ip) {
		t.Fatalf("resolved IP %q is invalid IPv4", ip)
	}
}
