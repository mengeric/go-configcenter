package nacos

import (
	"errors"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/model"

	"github.com/mengeric/go-configcenter/adapter"
)

// TestNamingService_WeightedSelect 验证加权轮询算法的数学有效性与边界防御
func TestNamingService_WeightedSelect(t *testing.T) {
	ns := &NamingService{}

	t.Run("空切片边界绝不允许越界panic", func(t *testing.T) {
		var instances []adapter.Instance
		_, err := ns.weightedSelect(instances)
		if !errors.Is(err, ErrNoHealthyInstance) {
			t.Errorf("expected ErrNoHealthyInstance on empty slice, got: %v", err)
		}
	})

	t.Run("全部不健康节点绝不允许分配给上游", func(t *testing.T) {
		instances := []adapter.Instance{
			{Ip: "10.0.0.1", Port: 8080, Weight: 10, Healthy: false, Enable: true},
			{Ip: "10.0.0.2", Port: 8080, Weight: 10, Healthy: false, Enable: true},
		}
		_, err := ns.weightedSelect(instances)
		if !errors.Is(err, ErrNoHealthyInstance) {
			t.Errorf("expected ErrNoHealthyInstance when all unhealthy, got: %v", err)
		}
	})

	t.Run("被禁用Enable=false节点必须被安全剔除", func(t *testing.T) {
		instances := []adapter.Instance{
			{Ip: "10.0.0.1", Port: 8080, Weight: 10, Healthy: true, Enable: false},
		}
		_, err := ns.weightedSelect(instances)
		if !errors.Is(err, ErrNoHealthyInstance) {
			t.Errorf("expected ErrNoHealthyInstance when disabled, got: %v", err)
		}
	})

	t.Run("单一健康实例直接命中", func(t *testing.T) {
		instances := []adapter.Instance{
			{Ip: "10.0.0.1", Port: 8080, Weight: 10, Healthy: true, Enable: true},
		}
		inst, err := ns.weightedSelect(instances)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inst.Ip != "10.0.0.1" || inst.Port != 8080 {
			t.Errorf("unexpected selected instance: %+v", inst)
		}
	})

	t.Run("多实例加权选取分布合理", func(t *testing.T) {
		instances := []adapter.Instance{
			{Ip: "10.0.0.1", Port: 8080, Weight: 10, Healthy: true, Enable: true},
			{Ip: "10.0.0.2", Port: 8080, Weight: 90, Healthy: true, Enable: true},
		}

		counts := make(map[string]int)
		for i := 0; i < 1000; i++ {
			inst, err := ns.weightedSelect(instances)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			counts[inst.Ip]++
		}

		// 权重 90 的节点被命中概率应当显著高于权重 10 的节点
		if counts["10.0.0.2"] <= counts["10.0.0.1"] {
			t.Errorf("weighted selection distribution abnormal: %+v", counts)
		}
	})
}

// TestNamingService_ToAdapterInstances 验证 Nacos model.Instance 转换过滤机制
func TestNamingService_ToAdapterInstances(t *testing.T) {
	raw := []model.Instance{
		{Ip: "10.0.0.1", Port: 8080, Weight: 10, Healthy: true, Enable: true},
		{Ip: "10.0.0.2", Port: 8080, Weight: 10, Healthy: false, Enable: true}, // 不健康，应被过滤
		{Ip: "10.0.0.3", Port: 8080, Weight: 10, Healthy: true, Enable: false}, // 未启用，应被过滤
	}

	converted := toAdapterInstances(raw)
	if len(converted) != 1 {
		t.Fatalf("expected 1 healthy&enabled instance, got %d", len(converted))
	}
	if converted[0].Ip != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", converted[0].Ip)
	}
}

// TestNamingService_ServiceKey 验证缓存 Key 格式
func TestNamingService_ServiceKey(t *testing.T) {
	key := serviceKey("order-service", "DEFAULT_GROUP")
	if key != "order-service|DEFAULT_GROUP" {
		t.Errorf("serviceKey = %s, want order-service|DEFAULT_GROUP", key)
	}
}
