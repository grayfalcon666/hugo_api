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

// ======================== 1. 配置结构体（与config.json对应） ========================
type Config struct {
	APIKey          string `json:"api_key"`           // 从config.json读取密钥
	HugoContentPath string `json:"hugo_content_path"` // 从config.json读取文章路径
	HugoProjectPath string `json:"hugo_project_path"` // 从config.json读取Hugo根路径
	HugoExecPath    string `json:"hugo_exec_path"`    // 从config.json读取Hugo执行路径
	ListenAddr      string `json:"listen_addr"`       // 从config.json读取监听地址
}

// 全局配置变量（程序启动时加载config.json）
var config Config

// ======================== 2. 请求参数结构体（严格匹配7个参数） ========================
type PostRequest struct {
	Title      string   `json:"title"`      // 是：文章标题（支持中文/特殊字符）
	Content    string   `json:"content"`    // 是：文章正文（支持Markdown，表单提交时直接复制）
	Tags       []string `json:"tags"`       // 否：标签数组（表单用逗号分隔，如"Go,编程"）
	Categories []string `json:"categories"` // 否：分类数组（表单用逗号分隔，如"技术教程,Go语言"）
	Filename   string   `json:"filename"`   // 否：自定义文件名（无需带.md）
	Draft      *bool    `json:"draft"`      // 否：是否草稿（表单填"true"/"false"，默认false）
	Date       string   `json:"date"`       // 否：自定义时间（格式2006-01-02 15:04:05）
}

// ======================== 3. 响应结构体 ========================
type Response struct {
	Status   string `json:"status"`             // success/error
	Message  string `json:"message"`            // 结果描述
	Filename string `json:"filename,omitempty"` // 成功时返回文件名
	Error    string `json:"error,omitempty"`    // 失败时返回错误日志
}

// ======================== 4. 加载配置文件（核心：从config.json读取配置） ========================
func loadConfig(filePath string) error {
	// 1. 读取config.json文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("找不到config.json：%w（请确保文件在程序同级目录）", err)
	}

	// 2. 解析JSON到Config结构体
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("config.json格式错误：%w（请检查JSON语法）", err)
	}

	// 3. 校验必填配置（避免配置缺失导致报错）
	if config.APIKey == "" {
		return fmt.Errorf("config.json中api_key不能为空")
	}
	if config.HugoContentPath == "" {
		return fmt.Errorf("config.json中hugo_content_path不能为空")
	}
	if config.HugoProjectPath == "" {
		return fmt.Errorf("config.json中hugo_project_path不能为空")
	}
	return nil
}

// ======================== 5. 主函数（先加载配置，再启动服务） ========================
func main() {
	// 第一步：加载config.json（修改配置仅需改文件，无需编译）
	if err := loadConfig("config.json"); err != nil {
		fmt.Printf("❌ 配置加载失败：%v\n", err)
		os.Exit(1)
	}

	// 第二步：注册路由（路径稳定，避免变更）
	http.HandleFunc("/api/hugo/create-post", authMiddleware(createPostHandler))

	// 第三步：启动API服务（用config里的监听地址）
	fmt.Printf("✅ API服务启动成功\n")
	fmt.Printf("📌 监听地址：%s\n", config.ListenAddr)
	fmt.Printf("📌 文章存放目录：%s\n", config.HugoContentPath)
	fmt.Printf("📌 支持格式：JSON请求 + 表单请求（直接复制Markdown）\n")
	if err := http.ListenAndServe(config.ListenAddr, nil); err != nil {
		fmt.Printf("❌ API启动失败：%v\n", err)
		os.Exit(1)
	}
}

// ======================== 6. 密钥认证中间件（用config里的api_key） ========================
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

// ======================== 7. 核心业务：支持表单+JSON双格式解析 + 文章生成 ========================
func createPostHandler(w http.ResponseWriter, r *http.Request) {
	// 仅支持POST请求
	if r.Method != http.MethodPost {
		sendResponse(w, http.StatusMethodNotAllowed, Response{
			Status:  "error",
			Message: "仅支持POST请求（支持：表单格式/JSON格式）",
		})
		return
	}

	// ======================== 修复语法错误：else必须紧跟if的} ========================
	var req PostRequest
	var err error
	contentType := r.Header.Get("Content-Type")

	// 1. 处理表单格式（multipart/form-data 或 application/x-www-form-urlencoded）
	if strings.Contains(contentType, "multipart/form-data") {
		// 解析multipart表单（适合大文本/特殊字符，推荐用于Markdown）
		err = r.ParseMultipartForm(10 << 20) // 最大支持10MB内容（足够存长文）
		if err == nil {
			req = parseFormData(r) // 从表单提取参数
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") { // 修复：else和前一个}同行
		// 解析普通表单（适合简单文本）
		err = r.ParseForm()
		if err == nil {
			req = parseFormData(r) // 从表单提取参数
		}
		// 2. 处理JSON格式（兼容原有用法）—— 这里是原错误位置，已修复else与}的换行问题
	} else if strings.Contains(contentType, "application/json") { // 修复：else和前一个}同行
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close() // 避免资源泄漏
		err = decoder.Decode(&req)
	} else {
		// 3. 不支持的格式
		sendResponse(w, http.StatusUnsupportedMediaType, Response{
			Status:  "error",
			Message: "不支持的请求格式（仅支持：multipart/form-data、x-www-form-urlencoded、application/json）",
		})
		return
	}

	// 解析错误统一处理
	if err != nil {
		sendResponse(w, http.StatusBadRequest, Response{
			Status:  "error",
			Message: "请求参数解析失败",
			Error:   fmt.Sprintf("错误原因：%v（表单提交时直接复制Markdown即可，无需修改）", err),
		})
		return
	}

	// ======================== 原有逻辑：校验必填参数 ========================
	if req.Title == "" {
		sendResponse(w, http.StatusBadRequest, Response{
			Status:  "error",
			Message: "必填参数缺失：title（文章标题）",
		})
		return
	}
	if req.Content == "" {
		sendResponse(w, http.StatusBadRequest, Response{
			Status:  "error",
			Message: "必填参数缺失：content（文章正文，表单可直接复制Markdown）",
		})
		return
	}

	// ======================== 原有逻辑：处理时间（中国时区，默认当前时间） ========================
	cstZone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "加载北京时间时区失败",
			Error:   fmt.Sprintf("错误原因：%v（服务器可能缺少时区数据库）", err),
		})
		return
	}

	var postDate time.Time
	if req.Date != "" {
		// 解析用户自定义的时间（按指定格式）
		postDate, err = time.ParseInLocation("2006-01-02 15:04:05", req.Date, cstZone)
		if err != nil {
			sendResponse(w, http.StatusBadRequest, Response{
				Status:  "error",
				Message: "date参数格式错误",
				Error:   fmt.Sprintf("正确格式：2006-01-02 15:04:05（示例：2025-09-28 14:30:00），错误原因：%v", err),
			})
			return
		}
	} else {
		// 默认使用当前北京时间
		postDate = time.Now().In(cstZone)
	}

	// ======================== 原有逻辑：处理草稿状态（默认false） ========================
	draft := false
	if req.Draft != nil {
		draft = *req.Draft // 用户传了draft就用用户的值
	}

	// ======================== 原有逻辑：处理文件名（自定义/默认时间戳） ========================
	var filename string
	if req.Filename != "" {
		// 自定义文件名：自动补.md后缀
		if filepath.Ext(req.Filename) != ".md" {
			req.Filename += ".md"
		}
		filename = req.Filename
	} else {
		// 默认文件名：时间戳（20060102150405）+ 清理后的标题 + .md
		timestamp := postDate.Format("20060102150405")
		cleanTitle := sanitizeFilename(req.Title) // 清理非法字符
		filename = fmt.Sprintf("%s-%s.md", timestamp, cleanTitle)
	}
	// 拼接最终保存路径（用config里的hugo_content_path）
	savePath := filepath.Join(config.HugoContentPath, filename)

	// ======================== 原有逻辑：生成Hugo Front Matter ========================
	frontMatter := fmt.Sprintf(`---
title: "%s"
date: %s
draft: %t
`, escapeQuotes(req.Title), postDate.Format(time.RFC3339), draft) // 转义标题中的双引号

	// 追加标签（用户传了tags才加）
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

	// 追加分类（用户传了categories才加）
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

	// 闭合Front Matter
	frontMatter += "---\n\n"

	// ======================== 原有逻辑：组合完整文章 + 保存 ========================
	fullContent := frontMatter + req.Content
	if err := os.WriteFile(savePath, []byte(fullContent), 0644); err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "保存文章失败",
			Error:   fmt.Sprintf("错误原因：%v（可能是目录权限不足，检查%s的读写权限）", err, config.HugoContentPath),
		})
		return
	}

	// ======================== 原有逻辑：执行Hugo构建 ========================
	fmt.Printf("🔨 开始Hugo构建：%s\n", config.HugoProjectPath)
	hugoCmd := exec.Command(config.HugoExecPath) // 用config里的Hugo路径
	hugoCmd.Dir = config.HugoProjectPath         // 切换到Hugo根目录
	hugoOutput, err := hugoCmd.CombinedOutput()  // 获取构建日志
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Hugo构建失败",
			Error:   fmt.Sprintf("构建日志：%s，错误原因：%v", string(hugoOutput), err),
		})
		return
	}
	fmt.Printf("✅ Hugo构建成功：%s\n", string(hugoOutput))

	// ======================== 原有逻辑：返回成功响应 ========================
	sendResponse(w, http.StatusOK, Response{
		Status:   "success",
		Message:  "文章创建并发布成功",
		Filename: filename,
	})
}

// ======================== 新增：从表单数据提取参数到PostRequest ========================
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

// ======================== 工具函数：发送JSON响应 ========================
func sendResponse(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

// ======================== 工具函数：清理文件名（避免非法字符） ========================
func sanitizeFilename(name string) string {
	// 过滤系统非法字符（Windows/Linux通用）
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
	// 空格替换为连字符（避免路径解析问题）
	result = strings.ReplaceAll(result, " ", "-")
	// 限制文件名长度（避免超过系统限制）
	if len(result) > 100 {
		result = result[:100]
	}
	return result
}

// ======================== 工具函数：转义双引号（避免破坏Front Matter） ========================
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
