这是博主开源到github的另一个项目，此处贴的是使用文档。如果是跟我一样使用hugo建站并且苦于如何发布的小伙伴们，我强力推荐这个api，你会用上的！！

✿✿✿来支持一波吧✿✿✿

👇👇👇

[grayfalcon666/hugo_api: Quickly publish posts for your Hugo blog!](https://github.com/grayfalcon666/hugo_api)

**preview**：
![hugo-api-preview.gif|750](https://raw.githubusercontent.com/grayfalcon666/OSS-FOR-PICGO2/refs/heads/main//hugo-api-preview.gif)

一个用 Go 编写的轻量级 API 服务，支持通过 **表单提交**（直接复制 Markdown）快速创建 Hugo 静态博客文章，自动生成 Front Matter 并触发 Hugo 构建，无需手动操作文件或执行命令。

## 🌟 核心功能

- **发送文章与动态**：
    - /api/hugo/create-post
    - /api/hugo/create-moment
- **外置config文件**: 可自定义文章发布路径、密钥、api监听端口号
- **自动处理**：
    - 生成 Hugo 标准 Front Matter（标题、时间、标签、分类等）
    - 自动触发 Hugo 构建，发布后立即生效

## 🚀 快速开始
### 1. 克隆仓库到本地

### 2. 配置 `config.json`

在项目根目录创建 `config.json` 文件，按实际环境填写配置：

```json
{
  "api_key": "your-strong-secret-key",  
  "hugo_content_path": "/home/user/blog/content/posts", 
  "hugo_moment_path": "/home/user/blog/content/moments",
  "hugo_project_path": "/home/user/blog",  
  "hugo_exec_path": "/usr/local/bin/hugo",  
  "listen_addr": ":8080" 
}
```
### 3. 编译与启动
```bash
# 编译（生成可执行文件）
go build -o hugo-api hugo-api.go

# 启动服务
./hugo-api
```

#### 后台运行

linux 写一个系统服务即可，以下为示例:
```bash
[Unit]
Description=Hugo Blog Upload API
After=network.target nginx.service

[Service]
User=grayfalcon
WorkingDirectory=/home/grayfalcon/Hugo_Sites/hugo_api
ExecStart=/home/grayfalcon/Hugo_Sites/hugo_api/hugo_api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 4. 验证启动成功

若输出以下日志，说明服务正常运行：
```plaintext
✅ API服务启动成功
📌 监听地址：:8080
📌 /create-post 文章路径：/home/user/blog/content/posts
📌 /create-moment 文章路径：/home/user/blog/content/moments
```

## 📡 API 使用文档

### 1. 路由 1：创建普通文章 `/api/hugo/create-post`

文章将保存到 `config.json` 配置的 `hugo_content_path` 目录。

#### 请求参数说明

|参数名|类型|是否必填|说明|示例|
|---|---|---|---|---|
|`title`|字符串|✅ 是|文章标题（支持中文 / 特殊字符）|`"Go语言入门：从Hello World到API"`|
|`content`|字符串|✅ 是|文章正文（支持 Markdown，表单提交可直接复制，JSON 需转义 `\n`）|`"# 前言\n这是一篇测试文章\n\n## 正文"`|
|`tags`|字符串数组|❌ 否|文章标签（JSON 传数组，表单传逗号分隔字符串）|JSON: `["Go","Hugo"]`；表单: `"Go,Hugo"`|
|`categories`|字符串数组|❌ 否|文章分类（规则同 tags）|JSON: `["技术教程"]`；表单: `"技术教程"`|
|`filename`|字符串|❌ 否|自定义文件名（无需带 `.md`，默认用「时间戳 + 标题」生成）|`"go-starter-tutorial"`|
|`draft`|布尔值|❌ 否|是否为草稿（默认 `false`，表单传 `"true"`/`"false"`）|`false`|
|`date`|字符串|❌ 否|自定义文章时间（格式：`2006-01-02 15:04:05`，默认当前北京时间）|`"2025-10-03 09:30:00"`|

#### 请求示例

使用 `curl` 提交表单，`content` 字段可直接粘贴 Markdown 内容：

```bash
curl -X POST http://localhost:8080/api/hugo/create-post \
  -H "X-API-Key: your-strong-secret-key-123" \
  -F "title=Go语言入门：从Hello World到API" \
  -F "content=# 1. 环境准备\n需安装Go 1.20+\n\n# 2. Hello World代码\n```go\npackage main\nimport \"fmt\"\nfunc main() { fmt.Println(\"Hello\") }\n```" \
  -F "tags=Go,编程,后端" \
  -F "categories=技术教程" \
  -F "draft=false"
```

### 2. 路由 2：创建动态 / 短内容 `/api/hugo/create-moment`

功能与 `/create-post` 完全一致，仅构建文件路径不同。

## 🤝 贡献

欢迎提交 Issue 或 Pull Request 改进功能，例如：
- 增加文章更新 / 删除接口
- 支持更多 Hugo 配置（如自定义 Front Matter 字段）

## 📄 许可证

[MIT License](https://opensource.org/licenses/MIT)允许自由使用、修改和分发，仅需保留原作者版权声明。


2025-10-08 23:30:09更新：新增/api/hugo/list-post/post 与 /api/hugo/list-post/moment 查询md文件列表