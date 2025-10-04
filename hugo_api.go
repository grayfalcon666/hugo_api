package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ======================== 1. 配置结构体（新增 hugo_moment_path 字段） ========================
type Config struct {
	APIKey          string `json:"api_key"`           // 从config.json读取密钥
	HugoContentPath string `json:"hugo_content_path"` // 原文章路径（对应/create-post）
	HugoMomentPath  string `json:"hugo_moment_path"`  // 新增：Moment路径（对应/create-moment）
	HugoProjectPath string `json:"hugo_project_path"` // Hugo根路径（不变）
	HugoExecPath    string `json:"hugo_exec_path"`    // Hugo执行路径（不变）
	ListenAddr      string `json:"listen_addr"`       // 监听地址（不变）
}

// 全局配置变量（程序启动时加载config.json）
var config Config

// ======================== 2. 请求参数结构体（完全复用，无需修改） ========================
type PostRequest struct {
	Title      string   `json:"title"`      // 是：文章标题（支持中文/特殊字符）
	Content    string   `json:"content"`    // 是：文章正文（支持Markdown，表单提交时直接复制）
	Tags       []string `json:"tags"`       // 否：标签数组（表单用逗号分隔，如"Go,编程"）
	Categories []string `json:"categories"` // 否：分类数组（表单用逗号分隔，如"技术教程,Go语言"）
	Filename   string   `json:"filename"`   // 否：自定义文件名（无需带.md）
	Draft      *bool    `json:"draft"`      // 否：是否草稿（表单填"true"/"false"，默认false）
	Date       string   `json:"date"`       // 否：自定义时间（格式2006-01-02 15:04:05）
}

// ======================== 3. 响应结构体（完全复用，无需修改） ========================
type Response struct {
	Status   string `json:"status"`             // success/error
	Message  string `json:"message"`            // 结果描述
	Filename string `json:"filename,omitempty"` // 成功时返回文件名
	Error    string `json:"error,omitempty"`    // 失败时返回错误日志
}

// ======================== 4. 加载配置文件（新增 hugo_moment_path 校验） ========================
func loadConfig(filePath string) error {
	// 1. 读取config.json文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("找不到config.json：%w（请确保文件在程序同级目录）", err)
	}

	// 2. 解析JSON到Config结构体（自动识别新增的hugo_moment_path）
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("config.json格式错误：%w（请检查JSON语法，确保新增hugo_moment_path字段）", err)
	}

	// 3. 校验必填配置（新增hugo_moment_path校验，与原有路径同级）
	if config.APIKey == "" {
		return fmt.Errorf("config.json中api_key不能为空")
	}
	if config.HugoContentPath == "" {
		return fmt.Errorf("config.json中hugo_content_path不能为空（对应/create-post路由）")
	}
	if config.HugoMomentPath == "" { // 新增：校验Moment路径
		return fmt.Errorf("config.json中hugo_moment_path不能为空（对应/create-moment路由）")
	}
	if config.HugoProjectPath == "" {
		return fmt.Errorf("config.json中hugo_project_path不能为空")
	}
	return nil
}

// ======================== 5. 主函数（新增 /api/hugo/create-moment 路由注册） ========================
func main() {
	// 第一步：加载config.json（自动读取新增的hugo_moment_path）
	if err := loadConfig("config.json"); err != nil {
		fmt.Printf("❌ 配置加载失败：%v\n", err)
		os.Exit(1)
	}

	// 第二步：注册路由（新增/create-moment路由，复用认证中间件）
	http.HandleFunc("/api/hugo/create-post", authMiddleware(createPostHandler))     // 原有路由
	http.HandleFunc("/api/hugo/create-moment", authMiddleware(createMomentHandler)) // 新增路由

	// 第三步：启动API服务（日志新增Moment路径提示）
	fmt.Printf("✅ API服务启动成功\n")
	fmt.Printf("📌 监听地址：%s\n", config.ListenAddr)
	fmt.Printf("📌 /create-post 文章路径：%s\n", config.HugoContentPath)
	fmt.Printf("📌 /create-moment 文章路径：%s\n", config.HugoMomentPath) // 新增日志
	if err := http.ListenAndServe(config.ListenAddr, nil); err != nil {
		fmt.Printf("❌ API启动失败：%v\n", err)
		os.Exit(1)
	}
}

// ======================== 6. 密钥认证中间件（完全复用，无需修改） ========================
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从请求头或URL参数获取密钥
		receivedKey := r.Header.Get("X-API-Key")
		if receivedKey == "" {
			receivedKey = r.URL.Query().Get("api_key")
		}

		// 用config里的密钥校验（非硬编码）
		if receivedKey != config.APIKey {
			sendResponse(w, http.StatusUnauthorized, Response{
				Status:  "error",
				Message: "无效的API密钥（请检查config.json中的api_key）",
			})
			return
		}

		next.ServeHTTP(w, r)
	}
}

// ======================== 7. 原有路由处理函数：/api/hugo/create-post（完全不变） ========================
func createPostHandler(w http.ResponseWriter, r *http.Request) {
	// 仅支持POST请求
	if r.Method != http.MethodPost {
		sendResponse(w, http.StatusMethodNotAllowed, Response{
			Status:  "error",
			Message: "仅支持POST请求（支持：表单格式/JSON格式）",
		})
		return
	}

	// 解析请求参数（表单/JSON双格式）
	var req PostRequest
	var err error
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		err = r.ParseMultipartForm(10 << 20)
		if err == nil {
			req = parseFormData(r)
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		err = r.ParseForm()
		if err == nil {
			req = parseFormData(r)
		}
	} else if strings.Contains(contentType, "application/json") {
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		err = decoder.Decode(&req)
	} else {
		sendResponse(w, http.StatusUnsupportedMediaType, Response{
			Status:  "error",
			Message: "不支持的请求格式（仅支持：multipart/form-data、x-www-form-urlencoded、application/json）",
		})
		return
	}

	// 解析错误处理
	if err != nil {
		sendResponse(w, http.StatusBadRequest, Response{
			Status:  "error",
			Message: "请求参数解析失败",
			Error:   fmt.Sprintf("错误原因：%v（表单提交时直接复制Markdown即可，无需修改）", err),
		})
		return
	}

	// 校验必填参数
	if req.Title == "" {
		sendResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "必填参数缺失：title（文章标题）"})
		return
	}
	if req.Content == "" {
		sendResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "必填参数缺失：content（文章正文）"})
		return
	}

	// 处理时间（中国时区）
	cstZone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status: "error", Message: "加载北京时间时区失败", Error: fmt.Sprintf("错误原因：%v", err),
		})
		return
	}
	var postDate time.Time
	if req.Date != "" {
		postDate, err = time.ParseInLocation("2006-01-02 15:04:05", req.Date, cstZone)
		if err != nil {
			sendResponse(w, http.StatusBadRequest, Response{
				Status: "error", Message: "date参数格式错误", Error: fmt.Sprintf("正确格式：2006-01-02 15:04:05，错误原因：%v", err),
			})
			return
		}
	} else {
		postDate = time.Now().In(cstZone)
	}

	// 处理草稿状态
	draft := false
	if req.Draft != nil {
		draft = *req.Draft
	}

	// 处理文件名（核心：使用原有路径 hugo_content_path）
	var filename string
	if req.Filename != "" {
		if filepath.Ext(req.Filename) != ".md" {
			req.Filename += ".md"
		}
		filename = req.Filename
	} else {
		timestamp := postDate.Format("20060102150405")
		cleanTitle := sanitizeFilename(req.Title)
		filename = fmt.Sprintf("%s-%s.md", timestamp, cleanTitle)
	}
	savePath := filepath.Join(config.HugoContentPath, filename) // 原有路径：hugo_content_path

	// 生成Front Matter
	frontMatter := fmt.Sprintf(`---
title: "%s"
date: %s
draft: %t
`, escapeQuotes(req.Title), postDate.Format(time.RFC3339), draft)
	if len(req.Tags) > 0 {
		frontMatter += "tags: ["
		for i, tag := range req.Tags {
			if i > 0 {
				frontMatter += ", "
			}
			frontMatter += fmt.Sprintf("\"%s\"", escapeQuotes(tag))
		}
		frontMatter += "]\n"
	}
	if len(req.Categories) > 0 {
		frontMatter += "categories: ["
		for i, cat := range req.Categories {
			if i > 0 {
				frontMatter += ", "
			}
			frontMatter += fmt.Sprintf("\"%s\"", escapeQuotes(cat))
		}
		frontMatter += "]\n"
	}
	frontMatter += "---\n\n"

	// 保存文章
	fullContent := frontMatter + req.Content
	if err := os.WriteFile(savePath, []byte(fullContent), 0644); err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status: "error", Message: "保存文章失败", Error: fmt.Sprintf("错误原因：%v（检查%s权限）", err, config.HugoContentPath),
		})
		return
	}

	// 执行Hugo构建
	fmt.Printf("🔨 开始Hugo构建：%s\n", config.HugoProjectPath)
	hugoCmd := exec.Command(config.HugoExecPath)
	hugoCmd.Dir = config.HugoProjectPath
	hugoOutput, err := hugoCmd.CombinedOutput()
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status: "error", Message: "Hugo构建失败", Error: fmt.Sprintf("构建日志：%s，错误原因：%v", string(hugoOutput), err),
		})
		return
	}
	fmt.Printf("✅ Hugo构建成功：%s\n", string(hugoOutput))

	// 返回成功响应
	sendResponse(w, http.StatusOK, Response{
		Status: "success", Message: "文章创建并发布成功（/create-post）", Filename: filename,
	})
}

// ======================== 8. 新增路由处理函数：/api/hugo/create-moment（仅改保存路径） ========================
func createMomentHandler(w http.ResponseWriter, r *http.Request) {
	// ------------ 以下逻辑与createPostHandler完全一致，仅最后保存路径改为 hugo_moment_path ------------
	if r.Method != http.MethodPost {
		sendResponse(w, http.StatusMethodNotAllowed, Response{
			Status:  "error",
			Message: "仅支持POST请求（支持：表单格式/JSON格式）",
		})
		return
	}

	var req PostRequest
	var err error
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		err = r.ParseMultipartForm(10 << 20)
		if err == nil {
			req = parseFormData(r)
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		err = r.ParseForm()
		if err == nil {
			req = parseFormData(r)
		}
	} else if strings.Contains(contentType, "application/json") {
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		err = decoder.Decode(&req)
	} else {
		sendResponse(w, http.StatusUnsupportedMediaType, Response{
			Status:  "error",
			Message: "不支持的请求格式（仅支持：multipart/form-data、x-www-form-urlencoded、application/json）",
		})
		return
	}

	if err != nil {
		sendResponse(w, http.StatusBadRequest, Response{
			Status:  "error",
			Message: "请求参数解析失败",
			Error:   fmt.Sprintf("错误原因：%v（表单提交时直接复制Markdown即可，无需修改）", err),
		})
		return
	}

	if req.Title == "" {
		sendResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "必填参数缺失：title（文章标题）"})
		return
	}
	if req.Content == "" {
		sendResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "必填参数缺失：content（文章正文）"})
		return
	}

	// 处理时间（与原有逻辑一致）
	cstZone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status: "error", Message: "加载北京时间时区失败", Error: fmt.Sprintf("错误原因：%v", err),
		})
		return
	}
	var postDate time.Time
	if req.Date != "" {
		postDate, err = time.ParseInLocation("2006-01-02 15:04:05", req.Date, cstZone)
		if err != nil {
			sendResponse(w, http.StatusBadRequest, Response{
				Status: "error", Message: "date参数格式错误", Error: fmt.Sprintf("正确格式：2006-01-02 15:04:05，错误原因：%v", err),
			})
			return
		}
	} else {
		postDate = time.Now().In(cstZone)
	}

	// 处理草稿状态（与原有逻辑一致）
	draft := false
	if req.Draft != nil {
		draft = *req.Draft
	}

	// 处理文件名（核心修改：保存路径改为 config.HugoMomentPath）
	var filename string
	if req.Filename != "" {
		if filepath.Ext(req.Filename) != ".md" {
			req.Filename += ".md"
		}
		filename = req.Filename
	} else {
		timestamp := postDate.Format("20060102150405")
		cleanTitle := sanitizeFilename(req.Title)
		filename = fmt.Sprintf("%s-%s.md", timestamp, cleanTitle)
	}
	// 核心修改点：从 config.HugoContentPath 改为 config.HugoMomentPath
	savePath := filepath.Join(config.HugoMomentPath, filename)

	// 生成Front Matter（与原有逻辑一致）
	frontMatter := fmt.Sprintf(`---
title: "%s"
date: %s
draft: %t
`, escapeQuotes(req.Title), postDate.Format(time.RFC3339), draft)
	if len(req.Tags) > 0 {
		frontMatter += "tags: ["
		for i, tag := range req.Tags {
			if i > 0 {
				frontMatter += ", "
			}
			frontMatter += fmt.Sprintf("\"%s\"", escapeQuotes(tag))
		}
		frontMatter += "]\n"
	}
	if len(req.Categories) > 0 {
		frontMatter += "categories: ["
		for i, cat := range req.Categories {
			if i > 0 {
				frontMatter += ", "
			}
			frontMatter += fmt.Sprintf("\"%s\"", escapeQuotes(cat))
		}
		frontMatter += "]\n"
	}
	frontMatter += "---\n\n"

	// 保存文章（路径已改为Moment路径）
	fullContent := frontMatter + req.Content
	if err := os.WriteFile(savePath, []byte(fullContent), 0644); err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			// 错误提示同步改为Moment路径
			Status: "error", Message: "保存文章失败", Error: fmt.Sprintf("错误原因：%v（检查%s权限）", err, config.HugoMomentPath),
		})
		return
	}

	// 执行Hugo构建（与原有逻辑一致）
	fmt.Printf("🔨 开始Hugo构建：%s\n", config.HugoProjectPath)
	hugoCmd := exec.Command(config.HugoExecPath)
	hugoCmd.Dir = config.HugoProjectPath
	hugoOutput, err := hugoCmd.CombinedOutput()
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status: "error", Message: "Hugo构建失败", Error: fmt.Sprintf("构建日志：%s，错误原因：%v", string(hugoOutput), err),
		})
		return
	}
	fmt.Printf("✅ Hugo构建成功：%s\n", string(hugoOutput))

	// 返回成功响应（提示改为/create-moment）
	sendResponse(w, http.StatusOK, Response{
		Status: "success", Message: "文章创建并发布成功（/create-moment）", Filename: filename,
	})
}

// ======================== 9. 表单参数解析函数 ========================
func parseFormData(r *http.Request) PostRequest {
	var req PostRequest

	// 1. 基础字段（直接提取）
	req.Title = r.FormValue("title")       // 标题
	req.Content = r.FormValue("content")   // Markdown正文（直接复制，无需转义）
	req.Filename = r.FormValue("filename") // 自定义文件名
	req.Date = r.FormValue("date")         // 自定义时间

	// 2. 标签（表单用逗号分隔，如"Go,编程,后端" → 转数组）
	tagsStr := strings.TrimSpace(r.FormValue("tags"))
	if tagsStr != "" {
		req.Tags = strings.Split(tagsStr, ",")
		// 清理标签中的空格（如"Go , 编程" → ["Go","编程"]）
		for i, tag := range req.Tags {
			req.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// 3. 分类（同标签，逗号分隔转数组）
	catsStr := strings.TrimSpace(r.FormValue("categories"))
	if catsStr != "" {
		req.Categories = strings.Split(catsStr, ",")
		// 清理分类中的空格
		for i, cat := range req.Categories {
			req.Categories[i] = strings.TrimSpace(cat)
		}
	}

	// 4. 草稿状态（表单填"true"/"false"，默认false）
	draftStr := strings.TrimSpace(r.FormValue("draft"))
	if draftStr != "" {
		draftVal := draftStr == "true" // 转布尔值
		req.Draft = &draftVal          // 赋值指针（匹配结构体类型）
	}

	return req
}

// ======================== 10. 工具函数 ========================
func sendResponse(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

func sanitizeFilename(name string) string {
	illegalChars := []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|', '\n', '\r'}
	result := ""
	for _, c := range name {
		isIllegal := false
		for _, illegalC := range illegalChars {
			if c == illegalC {
				isIllegal = true
				break
			}
		}
		if !isIllegal {
			result += string(c)
		}
	}
	result = strings.ReplaceAll(result, " ", "-")
	if len(result) > 100 {
		result = result[:100]
	}
	return result
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
