package main

import (
	"fmt"

	goconfigcenter "github.com/mengeric/go-configcenter"
)

func main() {
	// 初始化 SDK
	s := goconfigcenter.MustInit("global.yaml", "galaxy")

	// 注册服务（本地模式下是空操作）
	if err := s.Register("0.0.0.0", 8888); err != nil {
		panic(err)
	}
	defer s.Deregister()

	// 获取合并后的配置内容
	sub := s.Subscriber()
	configContent, err := sub.Value()
	if err != nil {
		panic(err)
	}

	fmt.Println("Merged config:")
	fmt.Println(configContent)

	// 发现其他服务（本地模式下从 global.yaml 的 services 节读取）
	addr, err := s.Discover("phoenix")
	if err != nil {
		fmt.Printf("discover failed: %v\n", err)
	} else {
		fmt.Printf("phoenix address: %s\n", addr)
	}
}
