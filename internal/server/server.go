package server

import (
	"fmt"
	"net/http"

	"hugo-api/internal/config"
	"hugo-api/internal/handler"
)

// authMiddleware 对请求进行 API Key 认证
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		receivedKey := r.Header.Get("X-API-Key")
		if receivedKey == "" {
			receivedKey = r.URL.Query().Get("api_key")
		}

		if receivedKey != config.Cfg.APIKey {
			http.Error(w, "无效的API密钥", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RegisterRoutes 注册所有的 API 路由
func RegisterRoutes() {
	// 创建内容
	http.HandleFunc("/api/hugo/create-post", authMiddleware(handler.CreatePostHandler))
	http.HandleFunc("/api/hugo/create-moment", authMiddleware(handler.CreateMomentHandler))

	// 列出内容
	http.HandleFunc("/api/hugo/list-post/post", authMiddleware(handler.ListPostHandler))
	http.HandleFunc("/api/hugo/list-post/moment", authMiddleware(handler.ListMomentHandler))
}

// Start 启动 API 服务
func Start() error {
	fmt.Println("✅ API服务启动成功")
	fmt.Printf("📌 监听地址：%s\n", config.Cfg.ListenAddr)
	fmt.Printf("📌 文章写入路径 (/create-post): %s\n", config.Cfg.HugoContentPath)
	fmt.Printf("📌 动态写入路径 (/create-moment): %s\n", config.Cfg.HugoMomentPath)
	fmt.Printf("📌 文章读取路径 (/list-post/post): %s\n", config.Cfg.HugoContentPath)
	fmt.Printf("📌 动态读取路径 (/list-post/moment): %s\n", config.Cfg.HugoMomentPath)

	return http.ListenAndServe(config.Cfg.ListenAddr, nil)
}
