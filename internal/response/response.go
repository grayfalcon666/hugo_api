package response

import (
	"encoding/json"
	"net/http"
)

// CreateResponse 用于创建操作的响应
type CreateResponse struct {
	Status   string `json:"status"`             // "success" 或 "error"
	Message  string `json:"message"`            // 结果描述
	Filename string `json:"filename,omitempty"` // 成功时返回文件名
	Error    string `json:"error,omitempty"`    // 失败时返回错误日志
}

// ListResponse 用于列出文件的响应
type ListResponse struct {
	Status  string   `json:"status"`          // "success" 或 "error"
	Message string   `json:"message"`         // 结果描述
	MDFiles []string `json:"md_files"`        // .md文件名列表
	DirPath string   `json:"dir_path"`        // 当前读取的目录路径
	Error   string   `json:"error,omitempty"` // 错误信息
}

// SendJSON 统一了发送JSON响应的逻辑
func SendJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}
