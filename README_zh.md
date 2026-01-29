# CodeSentry

<div align="center">
  <img src="https://raw.githubusercontent.com/huangang/codesentry/main/frontend/public/codesentry-icon.png" alt="CodeSentry Logo" width="120" height="120">
</div>

基于 AI 的代码审查平台，支持 GitHub、GitLab 和 Bitbucket。

[English](./README.md)

## 功能特性

- **AI 代码审查**: 原生支持 OpenAI、Anthropic (Claude)、Ollama、Google Gemini、Azure OpenAI
- **分批审查**: 大型 MR/PR 自动分批处理，确保审查质量
- **智能过滤**: 自动跳过配置文件、锁文件、生成文件（可自定义）
- **自动打分**: 自定义提示词缺少打分指令时，系统自动追加评分要求
- **Commit 评论**: 将 AI 审查结果作为评论发布到 commit（支持 GitLab/GitHub）
- **Commit 状态**: 设置 commit 状态，分数低于阈值时阻止合并（支持 GitLab/GitHub）
- **同步审查 API**: 为 Git pre-receive hook 提供同步审查接口，可阻止不合格的 push
- **防重复审查**: 跳过已审查的 commit，避免重复处理
- **多平台支持**: GitHub、GitLab 和 Bitbucket Webhook 集成，支持多级项目路径
- **可视化看板**: 代码审查活动的统计指标和图表
- **可视化看板**: 代码审查活动的统计指标和图表
- **审查历史**: 详细的审查记录追踪，支持直接跳转到 commit/MR 页面
- **项目管理**: 管理多个代码仓库

## 界面预览

![CodeSentry Dashboard](https://raw.githubusercontent.com/huangang/codesentry/main/frontend/public/dashboard-preview.png)

- **大模型配置**: 配置多个 AI 模型，原生 SDK 集成（Anthropic/Gemini 无需代理）
- **提示词模板**: 系统和自定义提示词模板，支持复制为新模板
- **IM 通知**: 发送审查结果到钉钉、飞书、企业微信、Slack、Discord、Microsoft Teams、Telegram
- **日报功能**: 自动生成每日代码审查报告，AI 分析总结，通过 IM 机器人发送
- **错误通知**: 通过 IM 机器人实时接收系统错误告警
- **Git 凭证**: 支持通过 Webhook 自动创建项目，统一管理凭证
- **系统日志**: 完整记录 Webhook 事件、错误和系统操作
- **认证支持**: 本地认证和 LDAP 登录（可在 Web 界面配置）
- **权限管理**: 管理员和普通用户角色，不同权限级别
- **多数据库**: SQLite 开发环境，MySQL/PostgreSQL 生产环境
- **国际化**: 支持中英文切换（包括日期选择器本地化）
- **响应式设计**: 适配手机和平板的移动端友好界面
- **暗黑模式**: 支持明暗主题切换，用户偏好自动保存

## 快速开始

### 前置要求

- Go 1.24+
- Node.js 20+
- Docker (可选)

### 开发环境

#### 后端

```bash
cd backend

# 创建配置文件
cp ../config.yaml.example config.yaml
# 编辑 config.yaml 配置

# 运行
go run ./cmd/server
```

#### 前端

```bash
cd frontend

# 安装依赖
npm install

# 运行开发服务器
npm run dev
```

访问 `http://localhost:5173`

**默认账号**: `admin` / `admin`

### Docker 部署

```bash
# 从 Docker Hub 拉取
docker pull huangangzhang/codesentry:latest

# 或从 GitHub Container Registry 拉取
docker pull ghcr.io/huangang/codesentry:latest
```

**选择数据库：**

```bash
# MySQL（默认，推荐生产环境使用）
docker-compose up -d

# SQLite（简单，单文件存储）
docker-compose -f docker-compose.sqlite.yml up -d

# PostgreSQL
docker-compose -f docker-compose.postgres.yml up -d
```

**或直接运行（SQLite）：**

```bash
docker run -d -p 8080:8080 -v codesentry-data:/app/data huangangzhang/codesentry:latest
```

本地开发（从源码构建）：

```bash
docker-compose -f docker-compose.dev.yml up --build
```

访问 `http://localhost:8080`

### 构建脚本（本地）

```bash
# 一键构建（前端+后端打包）
./build.sh

# 运行
./codesentry
```

这将构建前端并嵌入到 Go 二进制文件中，生成单个可执行文件。

## 配置说明

复制 `config.yaml.example` 为 `config.yaml` 并修改:

```yaml
server:
  port: 8080
  mode: release  # debug, release, test

database:
  driver: sqlite   # sqlite, mysql, postgres
  dsn: data/codesentry.db
  # MySQL: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
  # PostgreSQL: host=localhost user=postgres password=xxx dbname=codesentry port=5432 sslmode=disable

jwt:
  secret: your-secret-key-change-in-production
  expire_hours: 24
```

> **注意**: 所有业务配置(大模型、LDAP、提示词、IM 机器人、Git 凭证)均通过 Web 界面管理,存储在数据库中。

## Webhook 配置

### 推荐：统一 Webhook（自动识别）

使用单一 Webhook 地址同时支持 GitLab、GitHub 和 Bitbucket：

```
https://你的域名/webhook
# 或
https://你的域名/review/webhook
```

系统会通过请求头自动识别平台。

### GitHub

1. 进入仓库设置 > Webhooks > 添加 Webhook
2. Payload URL: `https://你的域名/webhook`
3. Content type: `application/json`
4. Secret: 您配置的 Webhook 密钥
5. Events: 选择 "Pull requests" 和 "Pushes"

### GitLab

1. 进入项目设置 > Webhooks
2. URL: `https://你的域名/webhook`
3. Secret Token: 您配置的 Webhook 密钥
4. Trigger: Push events, Merge request events

### Bitbucket

1. 进入仓库设置 > Webhooks > Add webhook
2. URL: `https://你的域名/webhook`
3. Secret: 您配置的 Webhook 密钥（用于 HMAC-SHA256 签名验证）
4. Triggers: 选择 "Repository push" 和 "Pull request created/updated"

## API 接口

### 认证

- `POST /api/auth/login` - 登录
- `GET /api/auth/config` - 获取认证配置
- `GET /api/auth/me` - 获取当前用户
- `POST /api/auth/logout` - 退出登录
- `POST /api/auth/change-password` - 修改密码（仅本地用户）

### 项目管理

- `GET /api/projects` - 项目列表
- `POST /api/projects` - 创建项目
- `GET /api/projects/:id` - 获取项目
- `PUT /api/projects/:id` - 更新项目
- `DELETE /api/projects/:id` - 删除项目

### 审查记录

- `GET /api/review-logs` - 审查记录列表
- `GET /api/review-logs/:id` - 审查详情
- `POST /api/review-logs/:id/retry` - 重试失败的审查（仅管理员）
- `DELETE /api/review-logs/:id` - 删除审查记录（仅管理员）

### 用户管理

- `GET /api/users` - 用户列表（仅管理员）
- `PUT /api/users/:id` - 更新用户（仅管理员）
- `DELETE /api/users/:id` - 删除用户（仅管理员）

### 看板

- `GET /api/dashboard/stats` - 获取统计数据

### 大模型配置

- `GET /api/llm-configs` - 模型列表
- `GET /api/llm-configs/active` - 获取激活的模型列表（用于项目选择）
- `POST /api/llm-configs` - 创建模型
- `PUT /api/llm-configs/:id` - 更新模型
- `DELETE /api/llm-configs/:id` - 删除模型

### 提示词模板

- `GET /api/prompts` - 提示词列表
- `GET /api/prompts/:id` - 提示词详情
- `GET /api/prompts/default` - 获取默认提示词
- `GET /api/prompts/active` - 获取激活的提示词列表
- `POST /api/prompts` - 创建提示词（仅管理员）
- `PUT /api/prompts/:id` - 更新提示词（仅管理员）
- `DELETE /api/prompts/:id` - 删除提示词（仅管理员）
- `POST /api/prompts/:id/set-default` - 设为默认模板（仅管理员）

### IM 机器人

- `GET /api/im-bots` - 机器人列表
- `POST /api/im-bots` - 创建机器人
- `PUT /api/im-bots/:id` - 更新机器人
- `DELETE /api/im-bots/:id` - 删除机器人

### 日报

- `GET /api/daily-reports` - 日报列表
- `GET /api/daily-reports/:id` - 日报详情
- `POST /api/daily-reports/generate` - 手动生成日报（不发送通知）
- `POST /api/daily-reports/:id/resend` - 发送/重发通知

### Webhooks

- `POST /webhook` - **统一 Webhook（自动识别 GitLab/GitHub/Bitbucket，推荐）**
- `POST /review/webhook` - 统一 Webhook 别名
- `POST /api/webhook` - /api 前缀下的统一 Webhook
- `POST /api/review/webhook` - /api 前缀下的别名
- `POST /api/webhook/gitlab` - GitLab Webhook（自动匹配项目）
- `POST /api/webhook/github` - GitHub Webhook（自动匹配项目）
- `POST /api/webhook/gitlab/:project_id` - GitLab Webhook（指定项目ID）
- `POST /api/webhook/github/:project_id` - GitHub Webhook（指定项目ID）
- `POST /api/webhook/bitbucket` - Bitbucket Webhook（自动匹配项目）
- `POST /api/webhook/bitbucket/:project_id` - Bitbucket Webhook（指定项目ID）

### 同步审查（用于 Git Hooks）

- `POST /review/sync` - 同步代码审查，用于 pre-receive hook
- `POST /api/review/sync` - /api 前缀下的同步审查
- `GET /review/score?commit_sha=xxx` - 通过 commit SHA 查询审查状态/分数
- `GET /api/review/score?commit_sha=xxx` - /api 前缀下的查询接口

请求体:

```json
{
  "project_url": "https://gitlab.example.com/group/project",
  "commit_sha": "abc123...",
  "ref": "refs/heads/main",
  "author": "John Doe",
  "message": "feat: add new feature",
  "diffs": "diff --git a/file.go..."
}
```

响应:

```json
{
  "passed": true,
  "score": 85,
  "min_score": 60,
  "message": "Score: 85/100 (min: 60)",
  "review_id": 123
}
```

参考 `scripts/pre-receive-hook.sh` 获取 GitLab pre-receive hook 示例脚本。

### 系统日志

- `GET /api/system-logs` - 日志列表
- `GET /api/system-logs/modules` - 获取模块列表
- `GET /api/system-logs/retention` - 获取日志保留天数
- `PUT /api/system-logs/retention` - 设置日志保留天数
- `POST /api/system-logs/cleanup` - 手动清理过期日志

### 健康检查

- `GET /health` - 服务健康检查

## 项目结构

```
codesentry/
├── backend/
│   ├── cmd/server/          # 应用入口
│   ├── internal/
│   │   ├── config/          # 配置
│   │   ├── handlers/        # HTTP 处理器
│   │   ├── middleware/      # 认证、CORS 中间件
│   │   ├── models/          # 数据库模型
│   │   ├── services/        # 业务逻辑
│   │   └── utils/           # 工具函数
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── i18n/            # 国际化
│   │   ├── layouts/         # 布局组件
│   │   ├── pages/           # 页面组件
│   │   ├── services/        # API 服务
│   │   ├── stores/          # 状态管理
│   │   └── types/           # TypeScript 类型
│   └── package.json
├── Dockerfile
├── docker-compose.yml
├── config.yaml.example
├── README.md
└── README_zh.md
```

## 技术栈

### 后端

- Go 1.24
- Gin v1.11 (HTTP 框架)
- GORM v1.31 (ORM)
- JWT 认证
- LDAP 支持

### 前端

- React 19
- TypeScript 5.9
- Ant Design 5
- Recharts
- Zustand (状态管理)
- React Router 7
- react-i18next (国际化)
- react-markdown (审查结果渲染)

## 许可证

MIT
