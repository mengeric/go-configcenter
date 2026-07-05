package goconfigcenter

import (
	"os"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ============================================================
// config.go 单元测试
// ============================================================

func TestArgoConfig_Parse(t *testing.T) {
	// 创建临时 argo.yaml
	yamlContent := `
registry:
  type: "nacos"
  addr: "192.168.110.164:8848"
  namespace: "public"
  group: "DEFAULT_GROUP"

services:
  galaxy:
    local:
      - "etc/base.yaml"
      - "etc/galaxy.yaml"
    remote:
      - dataId: "shared.yaml"
        group: "SHARED"
      - dataId: "galaxy.yaml"
        group: "APP"
  phoenix:
    local:
      - "etc/base.yaml"
      - "etc/phoenix.yaml"
    remote:
      - dataId: "shared.yaml"
        group: "SHARED"
`
	tmpFile := "test-argo.yaml"
	os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	defer os.Remove(tmpFile)

	// 测试解析
	k := koanf.New(".")
	err := k.Load(file.Provider(tmpFile), yaml.Parser())
	if err != nil {
		t.Fatalf("load argo.yaml failed: %v", err)
	}

	var cfg ArgoConfig
	err = k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "json",
	})
	if err != nil {
		t.Fatalf("unmarshal argo.yaml failed: %v", err)
	}

	// 验证 registry
	if cfg.Registry.Type != "nacos" {
		t.Errorf("registry.type = %s, want nacos", cfg.Registry.Type)
	}
	if cfg.Registry.Addr != "192.168.110.164:8848" {
		t.Errorf("registry.addr = %s, want 192.168.110.164:8848", cfg.Registry.Addr)
	}
	if cfg.Registry.Namespace != "public" {
		t.Errorf("registry.namespace = %s, want public", cfg.Registry.Namespace)
	}

	// 验证 services
	galaxy, ok := cfg.Services["galaxy"]
	if !ok {
		t.Fatal("service galaxy not found")
	}
	if len(galaxy.Local) != 2 {
		t.Errorf("galaxy.local count = %d, want 2", len(galaxy.Local))
	}
	if len(galaxy.Remote) != 2 {
		t.Errorf("galaxy.remote count = %d, want 2", len(galaxy.Remote))
	}
	if galaxy.Remote[0].DataId != "shared.yaml" {
		t.Errorf("galaxy.remote[0].dataId = %s, want shared.yaml", galaxy.Remote[0].DataId)
	}
	if galaxy.Remote[0].Group != "SHARED" {
		t.Errorf("galaxy.remote[0].group = %s, want SHARED", galaxy.Remote[0].Group)
	}

	phoenix, ok := cfg.Services["phoenix"]
	if !ok {
		t.Fatal("service phoenix not found")
	}
	if len(phoenix.Local) != 2 {
		t.Errorf("phoenix.local count = %d, want 2", len(phoenix.Local))
	}
}

func TestArgoConfig_EmptyServices(t *testing.T) {
	yamlContent := `
registry:
  type: "nacos"
  addr: "192.168.110.164:8848"

services:
`
	tmpFile := "test-empty.yaml"
	os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	defer os.Remove(tmpFile)

	k := koanf.New(".")
	err := k.Load(file.Provider(tmpFile), yaml.Parser())
	if err != nil {
		t.Fatalf("load argo.yaml failed: %v", err)
	}

	var cfg ArgoConfig
	err = k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "json",
	})
	if err != nil {
		t.Fatalf("unmarshal argo.yaml failed: %v", err)
	}

	if len(cfg.Services) != 0 {
		t.Errorf("services count = %d, want 0", len(cfg.Services))
	}
}
