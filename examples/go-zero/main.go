package main

import (
	"flag"
	"fmt"

	goconfigcenter "github.com/mengeric/go-configcenter"
)

// ============================================================
// go-zero 服务接入示例
// ============================================================

// Config 服务配置结构体
type Config struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

func main() {
	flag.Parse()

	// 1. 初始化 SDK
	s := goconfigcenter.MustInit("argo.yaml", "galaxy")

	// 2. 注册服务
	if err := s.Register("0.0.0.0", 8888); err != nil {
		panic(err)
	}
	defer s.Deregister()

	// 3. 加载配置（go-zero configcenter 方式）
	// sub := s.Subscriber()
	// cc := configcenter.MustNewConfigCenter[Config](configcenter.Config{
	//     Type: s.ConfigType(),
	// }, sub)
	// c, err := cc.GetConfig()

	// 4. 合并配置文件（不走 go-zero configcenter 的场景）
	configPath := s.MergedConfigPath()
	fmt.Printf("merged config: %s\n", configPath)

	// 5. 服务发现
	addr, err := s.Discover("phoenix")
	if err != nil {
		fmt.Printf("discover phoenix failed: %v\n", err)
	} else {
		fmt.Printf("phoenix addr: %s\n", addr)
	}

	fmt.Printf("Starting galaxy server at :8888...\n")
	// server.Start()
}
