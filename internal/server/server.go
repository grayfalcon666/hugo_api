// 模块作用:
// 负责所有与 HTTP 服务器直接相关的工作。
// 路由注册：定义每个URL路径应该由哪个处理器函数来处理。
// 中间件 (Middleware)：管理像 authMiddleware 这样的通用功能。所有发往特定路由的请求都必须先通过它。

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

	// 查询列表
	http.HandleFunc("/api/hugo/list-post/post", authMiddleware(handler.ListPostHandler))
	http.HandleFunc("/api/hugo/list-post/moment", authMiddleware(handler.ListMomentHandler))

	// 获取指定内容
	http.HandleFunc("/api/hugo/get-post", authMiddleware(handler.GetPostHandler))
	http.HandleFunc("/api/hugo/get-moment", authMiddleware(handler.GetMomentHandler))
}

// Start 启动 API 服务
func Start() error {
	fmt.Println("✅ API服务启动成功")
	fmt.Printf("📌 监听地址：%s\n", config.Cfg.ListenAddr)
	fmt.Printf("📌 发布内容 (/create-post)\n")
	fmt.Printf("📌 列表查询 (/list-post/post)\n")
	fmt.Printf("📌 内容查询 (/get-post/post)\n")
	fmt.Printf("末尾换成moment即可对动态进行同等操作\n")

	return http.ListenAndServe(config.Cfg.ListenAddr, nil)
}
