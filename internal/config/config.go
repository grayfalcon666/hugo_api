package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 结构体定义了所有配置项
type Config struct {
	APIKey          string `json:"api_key"`
	HugoContentPath string `json:"hugo_content_path"`
	HugoMomentPath  string `json:"hugo_moment_path"`
	HugoProjectPath string `json:"hugo_project_path"`
	HugoExecPath    string `json:"hugo_exec_path"`
	ListenAddr      string `json:"listen_addr"`
}

// Cfg 是一个全局的、私有的配置实例
var Cfg *Config

// Load 函数负责读取和解析 config.json 文件
func Load(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("找不到config.json：%w（请确保文件在程序同级目录）", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("config.json格式错误：%w（请检查JSON语法）", err)
	}

	// 校验必填字段
	if cfg.APIKey == "" {
		return fmt.Errorf("config.json中api_key不能为空")
	}
	if cfg.HugoContentPath == "" {
		return fmt.Errorf("config.json中hugo_content_path不能为空")
	}
	if cfg.HugoMomentPath == "" {
		return fmt.Errorf("config.json中hugo_moment_path不能为空")
	}
	if cfg.HugoProjectPath == "" {
		return fmt.Errorf("config.json中hugo_project_path不能为空")
	}
	// 如果 hugo_exec_path 未填写，则默认为 "hugo"，使其变为可选配置
	if cfg.HugoExecPath == "" {
		cfg.HugoExecPath = "hugo"
	}

	Cfg = &cfg
	return nil
}
