package main

import (
	"fmt"
	"net/http"

	goconfigcenter "github.com/mengeric/go-configcenter"
)

func main() {
	// 初始化 SDK
	s := goconfigcenter.MustInit("global.yaml", "phoenix")

	// 注册服务（本地模式下是空操作）
	if err := s.Register("0.0.0.0", 10011); err != nil {
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

	// 普通 HTTP 服务
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from phoenix"))
	})

	fmt.Println("Starting server on :10011")
	http.ListenAndServe(":10011", nil)
}
