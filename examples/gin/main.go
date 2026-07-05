package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	goconfigcenter "github.com/mengeric/go-configcenter"
)

// ============================================================
// Gin 服务接入示例
// ============================================================

func main() {
	flag.Parse()

	// 1. 初始化 SDK
	s := goconfigcenter.MustInit("argo.yaml", "phoenix")

	// 2. 注册服务
	if err := s.Register("0.0.0.0", 10011); err != nil {
		log.Fatal(err)
	}
	defer s.Deregister()

	// 3. 合并配置文件
	configPath := s.MergedConfigPath()
	fmt.Printf("merged config: %s\n", configPath)

	// 4. 服务发现（调其他服务）
	addr, err := s.Discover("galaxy")
	if err != nil {
		fmt.Printf("discover galaxy failed: %v\n", err)
	} else {
		fmt.Printf("galaxy addr: %s\n", addr)
	}

	// 5. 启动 Gin 服务
	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from phoenix"))
	})
	fmt.Printf("Starting phoenix server at :10011...\n")
	http.ListenAndServe(":10011", nil)
}
