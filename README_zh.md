# CodeSentry

基于 AI 的代码审查平台，支持 GitHub 和 GitLab。

[English](./README.md)

## 功能特性

- **AI 代码审查**: 使用 OpenAI 兼容模型自动审查代码变更
- **多平台支持**: GitHub 和 GitLab Webhook 集成
- **可视化看板**: 代码审查活动的统计指标和图表
- **审查历史**: 详细的审查记录追踪
- **项目管理**: 管理多个代码仓库
- **大模型配置**: 配置多个 AI 模型和自定义接口
- **IM 通知**: 发送审查结果到钉钉、飞书、企业微信、Slack 或自定义 Webhook
- **认证支持**: 本地认证和 LDAP 登录
- **多数据库**: SQLite 开发环境，MySQL/PostgreSQL 生产环境
- **国际化**: 支持中英文切换

## 快速开始

### 前置要求

- Go 1.23+
- Node.js 18+
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
# 构建并运行
docker-compose up --build

# 或后台运行
docker-compose up -d --build
```

访问 `http://localhost:8080`

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

ldap:
  enabled: false
  host: ldap.example.com
  port: 389
  use_ssl: false
  bind_dn: cn=admin,dc=example,dc=com
  bind_password: password
  base_dn: dc=example,dc=com
  user_filter: (uid=%s)
```

## Webhook 配置

### GitHub

1. 进入仓库设置 > Webhooks > 添加 Webhook
2. Payload URL: `https://your-domain/api/webhook/github/{project_id}`
3. Content type: `application/json`
4. Secret: 您配置的 Webhook 密钥
5. Events: 选择 "Pull requests" 和 "Pushes"

### GitLab

1. 进入项目设置 > Webhooks
2. URL: `https://your-domain/api/webhook/gitlab/{project_id}`
3. Secret Token: 您配置的 Webhook 密钥
4. Trigger: Push events, Merge request events

## API 接口

### 认证
- `POST /api/auth/login` - 登录
- `GET /api/auth/config` - 获取认证配置
- `GET /api/auth/me` - 获取当前用户
- `POST /api/auth/logout` - 退出登录

### 项目管理
- `GET /api/projects` - 项目列表
- `POST /api/projects` - 创建项目
- `GET /api/projects/:id` - 获取项目
- `PUT /api/projects/:id` - 更新项目
- `DELETE /api/projects/:id` - 删除项目

### 审查记录
- `GET /api/review-logs` - 审查记录列表
- `GET /api/review-logs/:id` - 审查详情

### 看板
- `GET /api/dashboard/stats` - 获取统计数据

### 大模型配置
- `GET /api/llm-configs` - 模型列表
- `POST /api/llm-configs` - 创建模型
- `PUT /api/llm-configs/:id` - 更新模型
- `DELETE /api/llm-configs/:id` - 删除模型

### IM 机器人
- `GET /api/im-bots` - 机器人列表
- `POST /api/im-bots` - 创建机器人
- `PUT /api/im-bots/:id` - 更新机器人
- `DELETE /api/im-bots/:id` - 删除机器人

### Webhooks
- `POST /api/webhook/github/:project_id` - GitHub Webhook
- `POST /api/webhook/gitlab/:project_id` - GitLab Webhook

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
- Go 1.23
- Gin (HTTP 框架)
- GORM (ORM)
- JWT 认证
- LDAP 支持

### 前端
- React 18
- TypeScript
- Ant Design 5
- Recharts
- Zustand (状态管理)
- React Router 7
- react-i18next (国际化)

## 许可证

MIT
