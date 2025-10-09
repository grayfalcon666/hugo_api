package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeFilename 清理文件名中的非法字符
func SanitizeFilename(name string) string {
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

// EscapeQuotes 转义字符串中的双引号
func EscapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

// ListMDFileNames 读取指定目录下所有.md文件的文件名
func ListMDFileNames(dirPath string) ([]string, error) {
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("目录不存在：%s", dirPath)
		}
		return nil, fmt.Errorf("访问目录失败：%s（错误：%v）", dirPath, err)
	}

	if !dirInfo.IsDir() {
		return nil, fmt.Errorf("%s 不是有效目录", dirPath)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("读取目录 %s 失败（错误：%v）", dirPath, err)
	}

	var mdFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.ToLower(filepath.Ext(entry.Name())) == ".md" {
			mdFiles = append(mdFiles, entry.Name())
		}
	}

	return mdFiles, nil
}
