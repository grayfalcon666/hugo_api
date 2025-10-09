package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"hugo-api/internal/config"
	"hugo-api/internal/hugo"
	"hugo-api/internal/response"
	"hugo-api/internal/utils"
)

// PostRequest 定义了创建文章或动态的请求体结构
type PostRequest struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	Categories []string `json:"categories"`
	Filename   string   `json:"filename"`
	Draft      *bool    `json:"draft"`
	Date       string   `json:"date"`
}

// createContentHandler 是一个通用的内容创建处理器
func createContentHandler(w http.ResponseWriter, r *http.Request, contentType string) {
	req, err := parseRequest(r)
	if err != nil {
		response.SendJSON(w, err.StatusCode, err.Response)
		return
	}

	if req.Title == "" || req.Content == "" {
		response.SendJSON(w, http.StatusBadRequest, response.CreateResponse{
			Status: "error", Message: "必填参数缺失：title 或 content",
		})
		return
	}

	postDate, apiErr := parseDate(req.Date)
	if apiErr != nil {
		response.SendJSON(w, apiErr.StatusCode, apiErr.Response)
		return
	}

	var saveDir string
	if contentType == "post" {
		saveDir = config.Cfg.HugoContentPath
	} else {
		saveDir = config.Cfg.HugoMomentPath
	}

	filename := generateFilename(req.Filename, req.Title, postDate)
	savePath := filepath.Join(saveDir, filename)

	draft := false
	if req.Draft != nil {
		draft = *req.Draft
	}

	frontMatter := hugo.GenerateFrontMatter(req.Title, postDate, draft, req.Tags, req.Categories)
	fullContent := frontMatter + req.Content
	if err := hugo.WriteFile(savePath, fullContent); err != nil {
		response.SendJSON(w, http.StatusInternalServerError, response.CreateResponse{
			Status: "error", Message: "保存文章失败", Error: fmt.Sprintf("错误原因：%v（检查 %s 权限）", err, saveDir),
		})
		return
	}

	output, buildErr := hugo.BuildSite(config.Cfg.HugoProjectPath, config.Cfg.HugoExecPath)
	if buildErr != nil {
		response.SendJSON(w, http.StatusInternalServerError, response.CreateResponse{
			Status: "error", Message: "Hugo构建失败", Error: fmt.Sprintf("构建日志：%s，错误原因：%v", output, buildErr),
		})
		return
	}

	response.SendJSON(w, http.StatusOK, response.CreateResponse{
		Status: "success", Message: fmt.Sprintf("内容创建并发布成功（/%s）", contentType), Filename: filename,
	})
}

// CreatePostHandler 专门处理 /create-post 请求
func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.SendJSON(w, http.StatusMethodNotAllowed, response.CreateResponse{Status: "error", Message: "仅支持POST请求"})
		return
	}
	createContentHandler(w, r, "post")
}

// CreateMomentHandler 专门处理 /create-moment 请求
func CreateMomentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.SendJSON(w, http.StatusMethodNotAllowed, response.CreateResponse{Status: "error", Message: "仅支持POST请求"})
		return
	}
	createContentHandler(w, r, "moment")
}

// listContentHandler 是一个通用的内容列表处理器
func listContentHandler(w http.ResponseWriter, r *http.Request, contentType string) {
	if r.Method != http.MethodGet {
		response.SendJSON(w, http.StatusMethodNotAllowed, response.CreateResponse{Status: "error", Message: "仅支持GET请求"})
		return
	}

	var targetDir string
	if contentType == "post" {
		targetDir = config.Cfg.HugoContentPath
	} else {
		targetDir = config.Cfg.HugoMomentPath
	}

	mdFiles, err := utils.ListMDFileNames(targetDir)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "目录不存在") {
			status = http.StatusNotFound
		}
		response.SendJSON(w, status, response.ListResponse{
			Status: "error", Message: "读取目录失败", Error: err.Error(), DirPath: targetDir,
		})
		return
	}

	response.SendJSON(w, http.StatusOK, response.ListResponse{
		Status:  "success",
		Message: fmt.Sprintf("成功获取 %s 下的.md文件（共%d个）", targetDir, len(mdFiles)),
		MDFiles: mdFiles,
		DirPath: targetDir,
	})
}

// ListPostHandler 处理 /list-post/post 请求
func ListPostHandler(w http.ResponseWriter, r *http.Request) {
	listContentHandler(w, r, "post")
}

// ListMomentHandler 处理 /list-post/moment 请求
func ListMomentHandler(w http.ResponseWriter, r *http.Request) {
	listContentHandler(w, r, "moment")
}

// --- 辅助函数 ---

// APIError 封装了带有状态码的错误信息
type APIError struct {
	StatusCode int
	Response   response.CreateResponse
}

func (e *APIError) Error() string {
	return e.Response.Message
}

// parseRequest 解析来自 JSON 或表单的请求
func parseRequest(r *http.Request) (*PostRequest, *APIError) {
	var req PostRequest
	contentType := r.Header.Get("Content-Type")
	var err error

	// ======================== FIX START ========================
	// 恢复对不同表单类型的判断
	if strings.Contains(contentType, "application/json") {
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, &APIError{http.StatusBadRequest, response.CreateResponse{Status: "error", Message: "JSON请求体解析失败", Error: err.Error()}}
		}
	} else if strings.Contains(contentType, "multipart/form-data") {
		// 必须使用 ParseMultipartForm 来处理 form-data
		if err = r.ParseMultipartForm(10 << 20); err != nil { // 10MB 内存限制
			return nil, &APIError{http.StatusBadRequest, response.CreateResponse{Status: "error", Message: "multipart/form-data 解析失败", Error: err.Error()}}
		}
		req = parseFormData(r)
	} else {
		// 默认处理 x-www-form-urlencoded
		if err = r.ParseForm(); err != nil {
			return nil, &APIError{http.StatusBadRequest, response.CreateResponse{Status: "error", Message: "表单数据解析失败", Error: err.Error()}}
		}
		req = parseFormData(r)
	}
	// ========================= FIX END =========================

	return &req, nil
}

// parseFormData 从表单数据中提取信息
func parseFormData(r *http.Request) PostRequest {
	var req PostRequest
	// r.FormValue 会智能地从 URL 参数和已解析的表单体中获取值
	req.Title = r.FormValue("title")
	req.Content = r.FormValue("content")
	req.Filename = r.FormValue("filename")
	req.Date = r.FormValue("date")

	if tagsStr := strings.TrimSpace(r.FormValue("tags")); tagsStr != "" {
		req.Tags = strings.Split(tagsStr, ",")
		for i := range req.Tags {
			req.Tags[i] = strings.TrimSpace(req.Tags[i])
		}
	}

	if catsStr := strings.TrimSpace(r.FormValue("categories")); catsStr != "" {
		req.Categories = strings.Split(catsStr, ",")
		for i := range req.Categories {
			req.Categories[i] = strings.TrimSpace(req.Categories[i])
		}
	}

	if draftStr := r.FormValue("draft"); draftStr != "" {
		draftVal := (strings.ToLower(draftStr) == "true")
		req.Draft = &draftVal
	}
	return req
}

// parseDate 解析日期字符串，如果为空则返回当前时间
func parseDate(dateStr string) (time.Time, *APIError) {
	cstZone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Time{}, &APIError{http.StatusInternalServerError, response.CreateResponse{Status: "error", Message: "加载时区失败", Error: err.Error()}}
	}

	if dateStr == "" {
		return time.Now().In(cstZone), nil
	}

	postDate, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr, cstZone)
	if err != nil {
		return time.Time{}, &APIError{http.StatusBadRequest, response.CreateResponse{Status: "error", Message: "date参数格式错误", Error: "正确格式: 2006-01-02 15:04:05"}}
	}
	return postDate, nil
}

// generateFilename 决定最终的文件名
func generateFilename(reqFilename, title string, date time.Time) string {
	if reqFilename != "" {
		if filepath.Ext(reqFilename) != ".md" {
			return reqFilename + ".md"
		}
		return reqFilename
	}
	timestamp := date.Format("20060102150405")
	cleanTitle := utils.SanitizeFilename(title)
	return fmt.Sprintf("%s-%s.md", timestamp, cleanTitle)
}
