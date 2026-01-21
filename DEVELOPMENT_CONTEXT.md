# CodeSentry 项目开发上下文

## 项目概述
CodeSentry 是一个 AI 代码审查平台，支持 GitHub 和 GitLab，使用 Go 后端 + React 前端。

## 已完成的工作

### 1. 核心功能（100% 完成）
- **后端**: Go + Gin + GORM，支持 SQLite/MySQL/PostgreSQL
- **前端**: React + TypeScript + Ant Design 5 + Recharts
- **认证**: 本地登录 + LDAP 支持
- **国际化**: react-i18next，支持中英文切换
- **Docker**: 多阶段构建 Dockerfile + docker-compose

### 2. 已完成的页面和功能
- ✅ Login（登录）
- ✅ Dashboard（仪表盘）- 修复了日期查询和 i18n
- ✅ Projects（项目管理）- ID 列、Webhook URL 复制按钮
- ✅ ReviewLogs（审查记录）
- ✅ LLMModels（大模型管理）
- ✅ Prompts（提示词管理）- 系统提示词和自定义提示词
- ✅ IMBots（通知机器人）- 密钥字段逻辑修复
- ✅ MemberAnalysis（成员分析）- 完整实现，包括详情抽屉
- ✅ SystemLogs（系统日志）

### 3. Webhook 和 AI 审查
- ✅ GitLab/GitHub Webhook 处理
- ✅ 异步 AI 审查（5分钟超时）
- ✅ IM 通知（企业微信、钉钉、飞书、Slack）
- ✅ 消息截断防止超长（commit 100字符，review 2000字符）
- ✅ 详细日志输出（diff获取、AI输入输出）

### 4. 最近修复的问题
- 日期查询：将 `time.Time` 改为 `string` 解析，支持前端 `YYYY-MM-DD` 格式
- 仪表盘 i18n：添加了时间范围选项的中文翻译
- 成员分析：完整实现后端 API 和前端页面

## 当前正在进行的工作

### ✅ IM 通知消息分段发送（已完成）
用户需求：当企业微信或其他机器人消息超长时，分成多条消息发送，而不是截断。

已完成：
- ✅ 添加 `splitMessage` 辅助函数（在换行处智能分割）
- ✅ 企业微信：超过 4000 字符时分段发送，带 `[1/N]` 标识
- ✅ 钉钉：超过 19000 字符时分段发送
- ✅ 飞书：超过 4000 字符时分段发送
- ✅ Slack：超过 3000 字符时分段发送
- ✅ 移除 `buildMessage` 中的 2000 字符截断逻辑

## 待解决问题

（暂无）

## 关键文件位置

### 后端
- `backend/cmd/server/main.go` - 主入口和路由
- `backend/internal/services/notification.go` - IM 通知服务
- `backend/internal/services/webhook.go` - Webhook 处理
- `backend/internal/services/ai.go` - AI 审查服务
- `backend/internal/services/member.go` - 成员分析服务
- `backend/internal/services/dashboard.go` - 仪表盘服务
- `backend/internal/services/system_log.go` - 系统日志服务
- `backend/internal/handlers/` - API handlers

### 前端
- `frontend/src/pages/` - 页面组件
- `frontend/src/services/index.ts` - API 服务
- `frontend/src/i18n/locales/en.json` - 英文语言包
- `frontend/src/i18n/locales/zh.json` - 中文语言包

## 重要约束

1. **不要使用 antd v6** - 当前使用 antd v5
2. **Go module 路径**: `github.com/huangang/codesentry/backend`
3. **默认登录账号**: admin / admin

## 构建命令

```bash
# 后端构建
cd backend && go build -o codesentry ./cmd/server

# 前端构建
cd frontend && npm run build

# Docker 构建
docker-compose up --build
```

## IM 机器人消息限制

| 平台 | 消息类型 | 字符限制 |
|------|----------|----------|
| 企业微信 | Markdown | 4096 |
| 钉钉 | Markdown | 20000 |
| 飞书 | Text | 4096 |
| Slack | Text | 40000 |
