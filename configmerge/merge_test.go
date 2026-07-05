package configmerge

import (
	"os"
	"testing"
)

// ============================================================
// configmerge/merge.go 单元测试
// ============================================================

func TestMerger_Merge(t *testing.T) {
	// 创建临时配置文件
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

	// 测试合并
	merger := NewMerger("galaxy", []string{
		"test-config/base.yaml",
		"test-config/galaxy.yaml",
	})

	outputPath, err := merger.Merge(".")
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	defer os.Remove(outputPath)

	// 验证输出文件存在
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("merged file not found: %s", outputPath)
	}

	// 验证内容（后面的配置覆盖前面的）
	content, _ := os.ReadFile(outputPath)
	contentStr := string(content)

	// app.name 应该是 galaxy（被覆盖）
	if !contains(contentStr, "name: galaxy") {
		t.Errorf("merged config should have name: galaxy, got:\n%s", contentStr)
	}

	// redis 应该保留
	if !contains(contentStr, "host: 192.168.1.1") {
		t.Errorf("merged config should have redis host, got:\n%s", contentStr)
	}

	// mysql 应该新增
	if !contains(contentStr, "host: 192.168.1.2") {
		t.Errorf("merged config should have mysql host, got:\n%s", contentStr)
	}
}

func TestMerger_MergeToBytes(t *testing.T) {
	os.MkdirAll("test-config", 0755)
	defer os.RemoveAll("test-config")

	baseConfig := `
app:
  name: "base"
`
	os.WriteFile("test-config/base.yaml", []byte(baseConfig), 0644)

	merger := NewMerger("test", []string{"test-config/base.yaml"})

	data, err := merger.MergeToBytes()
	if err != nil {
		t.Fatalf("MergeToBytes failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("MergeToBytes returned empty")
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

	// 空文件列表应该返回空 YAML map（yaml.Marshal 会加换行符）
	result := string(data)
	if result != "{}\n" && result != "{}" {
		t.Errorf("MergeToBytes with empty files should return {}, got: %q", result)
	}
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
