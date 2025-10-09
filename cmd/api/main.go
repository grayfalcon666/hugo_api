package main

import (
	"log"

	"hugo-api/internal/config"
	"hugo-api/internal/server"
)

func main() {
	// 1. 加载配置
	if err := config.Load("config.json"); err != nil {
		log.Fatalf("❌ 配置加载失败：%v\n", err)
	}

	// 2. 注册路由
	server.RegisterRoutes()

	// 3. 启动服务
	if err := server.Start(); err != nil {
		log.Fatalf("❌ API启动失败：%v\n", err)
	}
}
