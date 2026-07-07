package configmerge

import (
	"os"
	"testing"
)

// ============================================================
// configmerge/merge.go 单元测试
// ============================================================

func TestMerger_MergeToBytes(t *testing.T) {
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	baseConfig := `
app:
  name: "base"
  version: "1.0"
redis:
  host: "192.168.1.1"
  port: 6379
`
	serviceConfig := `
app:
  name: "galaxy"
  port: 8888
mysql:
  host: "192.168.1.2"
  port: 3306
`
	os.WriteFile("test-config/base.yaml", []byte(baseConfig), 0644)
	os.WriteFile("test-config/galaxy.yaml", []byte(serviceConfig), 0644)

	merger := NewMerger("galaxy", []string{
		"test-config/base.yaml",
		"test-config/galaxy.yaml",
	})

	data, err := merger.MergeToBytes()
	if err != nil {
		t.Fatalf("MergeToBytes failed: %v", err)
	}

	content := string(data)

	// app.name 应该是 galaxy（被覆盖）
	if !contains(content, "name: galaxy") {
		t.Errorf("merged config should have name: galaxy, got:\n%s", content)
	}

	// redis 应该保留
	if !contains(content, "host: 192.168.1.1") {
		t.Errorf("merged config should have redis host, got:\n%s", content)
	}

	// mysql 应该新增
	if !contains(content, "host: 192.168.1.2") {
		t.Errorf("merged config should have mysql host, got:\n%s", content)
	}
}

func TestMerger_AddRemote(t *testing.T) {
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	baseConfig := `
app:
  name: "base"
`
	remoteConfig := `
redis:
  host: "192.168.110.164"
`
	os.WriteFile("test-config/base.yaml", []byte(baseConfig), 0644)

	merger := NewMerger("test", []string{"test-config/base.yaml"})
	merger.AddRemote(remoteConfig)

	data, err := merger.MergeToBytes()
	if err != nil {
		t.Fatalf("MergeToBytes failed: %v", err)
	}

	content := string(data)
	if !contains(content, "host: 192.168.110.164") {
		t.Errorf("merged config should have remote redis host, got:\n%s", content)
	}
}

func TestMerger_ConfigType(t *testing.T) {
	tests := []struct {
		files    []string
		expected string
	}{
		{[]string{"etc/base.yaml"}, "yaml"},
		{[]string{"etc/base.yml"}, "yaml"},
		{[]string{"etc/base.json"}, "json"},
		{[]string{"etc/base.toml"}, "toml"},
		{[]string{}, "yaml"},
	}

	for _, tt := range tests {
		merger := NewMerger("test", tt.files)
		result := merger.ConfigType()
		if result != tt.expected {
			t.Errorf("ConfigType(%v) = %s, want %s", tt.files, result, tt.expected)
		}
	}
}

func TestMerger_EmptyFiles(t *testing.T) {
	merger := NewMerger("test", []string{})

	data, err := merger.MergeToBytes()
	if err != nil {
		t.Fatalf("MergeToBytes failed: %v", err)
	}

	// 空文件列表应该返回空 YAML map
	result := string(data)
	if result != "{}\n" && result != "{}" {
		t.Errorf("MergeToBytes with empty files should return {}, got: %q", result)
	}
}

func TestMerger_PartialOverride(t *testing.T) {
	// 模拟 basic.yaml (SHARE) 和 galaxy.yaml (DEFAULT_GROUP) 的合并
	// basic.yaml 有完整的 Redis 配置（DB: 0），galaxy.yaml 只覆盖 DB: 1
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	basicConfig := `
Redis:
  Host: "192.168.110.164:6379"
  Pass: ""
  Type: "node"
  DB: 0
PhoenixDB:
  Type: "mysql"
  Host: "192.168.110.164"
  Port: 3306
  Schema: "phoenix"
  User: "root"
  Password: "Root@123"
`

	// galaxy.yaml 只覆盖 Redis.DB
	galaxyConfig := `
Name: galaxy
Host: 0.0.0.0
Port: 8888
Redis:
  DB: 1
`

	os.WriteFile("test-config/basic.yaml", []byte(basicConfig), 0644)
	os.WriteFile("test-config/galaxy.yaml", []byte(galaxyConfig), 0644)

	merger := NewMerger("galaxy", []string{
		"test-config/basic.yaml",
		"test-config/galaxy.yaml",
	})

	data, err := merger.MergeToBytes()
	if err != nil {
		t.Fatalf("MergeToBytes failed: %v", err)
	}

	content := string(data)

	// Redis.Host 应该保留（basic.yaml 的值）
	if !contains(content, "192.168.110.164:6379") {
		t.Errorf("merged config should preserve Redis.Host from basic.yaml, got:\n%s", content)
	}

	// Redis.DB 应该被覆盖为 1
	if !contains(content, "DB: 1") {
		t.Errorf("merged config should have Redis.DB: 1, got:\n%s", content)
	}

	// Redis.Pass 应该保留
	if !contains(content, `Pass: ""`) {
		t.Errorf("merged config should preserve Redis.Pass, got:\n%s", content)
	}

	// PhoenixDB 应该完整保留
	if !contains(content, "Schema: phoenix") {
		t.Errorf("merged config should preserve PhoenixDB, got:\n%s", content)
	}

	// Name 应该是 galaxy
	if !contains(content, "Name: galaxy") {
		t.Errorf("merged config should have Name: galaxy, got:\n%s", content)
	}

	t.Logf("Merged config:\n%s", content)
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
