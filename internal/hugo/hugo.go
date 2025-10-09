// 模块作用:
// 封装所有与 Hugo 直接交互的操作
// handler 模块不关心 Hugo 命令具体是什么，它只需要调用 hugo.BuildSite() 这个函数，就像调用一个服务一样

package hugo

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"hugo-api/internal/utils"
)

// GenerateFrontMatter 根据文章数据生成 Hugo 的 Front Matter
func GenerateFrontMatter(title string, postDate time.Time, draft bool, tags, categories []string) string {
	frontMatter := fmt.Sprintf(`---
title: "%s"
date: %s
draft: %t
`, utils.EscapeQuotes(title), postDate.Format(time.RFC3339), draft)

	if len(tags) > 0 {
		frontMatter += "tags: ["
		for i, tag := range tags {
			if i > 0 {
				frontMatter += ", "
			}
			frontMatter += fmt.Sprintf("\"%s\"", utils.EscapeQuotes(tag))
		}
		frontMatter += "]\n"
	}

	if len(categories) > 0 {
		frontMatter += "categories: ["
		for i, cat := range categories {
			if i > 0 {
				frontMatter += ", "
			}
			frontMatter += fmt.Sprintf("\"%s\"", utils.EscapeQuotes(cat))
		}
		frontMatter += "]\n"
	}
	frontMatter += "---\n\n"
	return frontMatter
}

// WriteFile 将生成的完整内容写入指定路径
func WriteFile(savePath, content string) error {
	return os.WriteFile(savePath, []byte(content), 0644)
}

// BuildSite 执行 Hugo 构建命令
func BuildSite(projectPath, execPath string) (string, error) {
	fmt.Printf("🔨 开始Hugo构建：%s\n", projectPath)
	cmd := exec.Command(execPath)
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	fmt.Printf("✅ Hugo构建成功：%s\n", string(output))
	return string(output), nil
}
