# CodeSentry 开发规范

## 项目结构

```
codesentry/
├── backend/                 # Go 后端
│   ├── cmd/server/         # 主入口
│   ├── internal/
│   │   ├── config/         # 配置
│   │   ├── handlers/       # API handlers
│   │   ├── middleware/     # 中间件
│   │   ├── models/         # 数据模型
│   │   ├── services/       # 业务逻辑
│   │   └── utils/          # 工具函数
│   └── go.mod
├── frontend/               # React 前端
│   ├── src/
│   │   ├── components/     # 通用组件
│   │   ├── pages/          # 页面组件
│   │   ├── services/       # API 服务
│   │   ├── i18n/           # 国际化
│   │   └── App.tsx
│   └── package.json
├── docker-compose.yml
├── Dockerfile
└── DEVELOPMENT_CONTEXT.md  # 开发上下文（必读）
```

## 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **ORM**: GORM
- **数据库**: SQLite (默认) / MySQL / PostgreSQL
- **Module 路径**: `github.com/huangang/codesentry/backend`

### 前端
- **框架**: React 18 + TypeScript
- **UI 库**: Ant Design 5 (⚠️ 不要使用 v6)
- **图表**: Recharts
- **国际化**: react-i18next
- **HTTP**: Axios

## 代码规范

### Go 后端

```go
// Handler 命名: XxxHandler
func (h *ProjectHandler) List(c *gin.Context) {}

// Service 命名: XxxService
type ProjectService struct { db *gorm.DB }

// 错误处理: 统一返回格式
c.JSON(http.StatusOK, gin.H{"data": result})
c.JSON(http.StatusBadRequest, gin.H{"error": "message"})

// 日志格式: 带模块标签
log.Printf("[ModuleName] message: %v", value)
```

### React 前端

```tsx
// 页面组件: 函数式 + Hooks
const ProjectsPage: React.FC = () => { ... }

// API 调用: 统一在 services/index.ts
export const projectAPI = {
  list: () => request.get('/api/projects'),
  create: (data) => request.post('/api/projects', data),
}

// 国际化: 使用 useTranslation
const { t } = useTranslation();
<span>{t('projects.title')}</span>
```

## 重要约束

1. **Ant Design 版本**: 使用 v5，不要升级到 v6
2. **不要使用 any**: 避免 `as any`、`@ts-ignore`
3. **API 路径**: 所有 API 以 `/api/` 开头
4. **认证**: JWT Token 存储在 localStorage
5. **默认账号**: admin / admin

## 构建验证

修改代码后必须验证：

```bash
# 后端
cd backend && go build -o codesentry ./cmd/server

# 前端
cd frontend && npm run build
```

## 常见任务

### 添加新页面
1. 创建 `frontend/src/pages/XxxPage.tsx`
2. 在 `App.tsx` 添加路由
3. 添加 i18n 翻译 (`locales/en.json`, `locales/zh.json`)
4. 添加菜单项

### 添加新 API
1. 在 `backend/internal/handlers/` 创建 handler
2. 在 `backend/internal/services/` 创建 service
3. 在 `main.go` 注册路由
4. 在 `frontend/src/services/index.ts` 添加 API 调用

### IM 机器人限制
| 平台 | 字符限制 |
|------|----------|
| 企业微信 | 4096 |
| 钉钉 | 20000 |
| 飞书 | 4096 |
| Slack | 40000 |

### Git 凭证自动创建项目
Git 凭证功能支持：
1. **自动创建项目**: 当 webhook 回调触发时，如果项目不存在但匹配到凭证，自动创建项目
2. **补全不完整数据**: 如果项目存在但 access_token 为空，自动从匹配的凭证中补全
3. **私有服务器支持**: 通过 `base_url` 字段支持自托管 GitLab/GitHub Enterprise

配置流程：
1. 在「Git 凭证」页面创建凭证，填写平台、服务器地址、Access Token、Webhook Secret
2. 在 GitLab/GitHub 配置统一 Webhook URL: `https://your-domain/webhook`
3. 新仓库触发 webhook 时自动创建项目并开始代码审查

### 错误日志 IM 通知
系统错误可以通过 IM 机器人实时通知，便于及时发现和处理问题。

配置流程：
1. 在「IM 机器人」页面创建或编辑机器人
2. 开启「错误通知」开关
3. 系统发生错误时会自动发送通知到所有启用错误通知的活跃机器人

技术实现：
- `LogError()` 函数在写入数据库后，异步发送 IM 通知
- 支持多个机器人同时接收错误通知
- 通知内容包含：模块、操作、错误信息、时间、额外数据

## 待完成功能

（暂无）
