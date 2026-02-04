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
│   │   ├── hooks/          # 自定义 Hooks
│   │   ├── layouts/        # 布局组件
│   │   ├── pages/          # 页面组件
│   │   ├── services/       # API 服务
│   │   ├── i18n/           # 国际化
│   │   ├── responsive.css  # 响应式样式
│   │   └── App.tsx
│   └── package.json
├── docker-compose.yml
└── Dockerfile
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

### 数据库列名规范

**⚠️ 重要**: 查询 `system_configs` 表时，必须使用反引号包裹保留字列名：

```go
// ✅ 正确: 使用反引号
db.Where("`key` = ?", key)
db.Where("`group` = ?", group)

// ❌ 错误: 不使用反引号或使用错误的列名
db.Where("key = ?", key)        // MySQL 中 key 是保留字，可能报错
db.Where("config_key = ?", key) // 列名不存在
```

**system_configs 表结构**:

| 列名 | 类型 | 说明 |
|------|------|------|
| `key` | varchar(100) | 配置键（MySQL 保留字，查询需加反引号） |
| `value` | text | 配置值 |
| `type` | varchar(20) | 值类型: string/int/bool/json |
| `group` | varchar(50) | 分组（MySQL 保留字，查询需加反引号） |
| `label` | varchar(200) | 显示标签 |

### 系统配置读取规范

**⚠️ 重要**: 读取系统配置时，必须使用 `SystemConfigService`，不要直接查询数据库：

```go
// ✅ 正确: 使用 SystemConfigService
configService := NewSystemConfigService(db)
value := configService.GetWithDefault("daily_report_timezone", "Asia/Shanghai")

// ❌ 错误: 直接查询数据库（重复代码）
var config models.SystemConfig
db.Where("`key` = ?", "daily_report_timezone").First(&config)
```

**Service 依赖注入模式**:

```go
type DailyReportService struct {
    db            *gorm.DB
    configService *SystemConfigService  // 注入 SystemConfigService
    // ...
}

func NewDailyReportService(db *gorm.DB) *DailyReportService {
    return &DailyReportService{
        db:            db,
        configService: NewSystemConfigService(db),
    }
}
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

### 响应式设计规范

项目已适配移动端，开发时需遵循以下规范：

**断点定义**:

- 移动端: `< 768px`
- 平板: `768px - 1024px`  
- 桌面: `> 1024px`

**布局规范**:

```tsx
// 表格必须添加水平滚动
<Table scroll={{ x: 800 }} ... />

// Modal/Drawer 响应式宽度：使用 getResponsiveWidth 工具函数
import { getResponsiveWidth } from '../hooks';
<Modal width={getResponsiveWidth(640)} ... />
<Drawer width={getResponsiveWidth(720)} ... />

// 使用 Ant Design 响应式栅格
<Col xs={24} sm={12} lg={6}>  // 移动端全宽，平板半宽，桌面1/4宽

// Descriptions 响应式列数
<Descriptions column={{ xs: 1, sm: 2 }}>

// Space 组件添加 wrap
<Space wrap>
```

**CSS 类**:

- `.hide-on-mobile` - 移动端隐藏
- `.show-on-mobile` - 仅移动端显示
- `.filter-area` - 筛选区域（自动换行）

**MainLayout 移动端行为**:

- 侧边栏隐藏，显示汉堡菜单按钮
- 点击菜单按钮弹出抽屉式菜单
- Header 高度从 64px 减少到 56px

### 暗黑模式规范

项目支持明暗主题切换，开发时需遵循以下规范：

**主题状态管理**:

```tsx
// 使用 themeStore 获取当前主题状态
import { useThemeStore } from '../stores/themeStore';

const MyComponent: React.FC = () => {
  const { isDark, toggleTheme } = useThemeStore();
  // isDark: boolean - 是否暗色模式
  // toggleTheme: () => void - 切换主题
};
```

**颜色适配规则**:

```tsx
// ✅ 正确: 使用动态颜色
style={{ 
  background: isDark ? '#1e293b' : '#ffffff',
  color: isDark ? '#e2e8f0' : '#1a1a1a',
  borderColor: isDark ? '#334155' : '#f0f0f0'
}}

// ❌ 错误: 硬编码颜色
style={{ background: '#fff', color: '#333' }}
```

**CSS 变量** (定义在 `index.css`):

```css
:root { --color-bg-primary: #ffffff; }
[data-theme='dark'] { --color-bg-primary: #1e293b; }
```

**Ant Design 组件**: 通过 `theme.ts` 的 `darkAlgorithm` 自动适配，无需额外处理。

**需要手动适配的场景**:

- Modal/Drawer 内的自定义内容区域
- ReactMarkdown 渲染的内容
- 内联 style 中的硬编码颜色

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

| 平台 | 字符限制 | 说明 |
|------|----------|------|
| 企业微信 | 4096 | 使用 markdown_v2 格式 |
| 钉钉 | 20000 | 支持加签密钥 |
| 飞书 | 4096 | 支持签名密钥 |
| Slack | 40000 | |
| Discord | 2000 | 直接 Webhook |
| Microsoft Teams | - | 使用 Adaptive Card |
| Telegram | - | 需要配置 chat_id |

### Git 凭证自动创建项目

Git 凭证功能支持：

1. **自动创建项目**: 当 webhook 回调触发时，如果项目不存在但匹配到凭证，自动创建项目
2. **补全不完整数据**: 如果项目存在但 access_token 为空，自动从匹配的凭证中补全
3. **私有服务器支持**: 通过 `base_url` 字段支持自托管 GitLab/GitHub Enterprise

配置流程：

1. 在「Git 凭证」页面创建凭证，填写平台、服务器地址、Access Token、Webhook Secret
2. 在 GitLab/GitHub/Bitbucket 配置统一 Webhook URL: `https://your-domain/webhook`
3. 新仓库触发 webhook 时自动创建项目并开始代码审查

### Bitbucket 支持

系统支持 Bitbucket Cloud 的 webhook 集成，包括推送事件和 Pull Request 事件。

**支持的事件**:

| 事件类型 | Bitbucket Event Key | 说明 |
|----------|---------------------|------|
| 推送 | `repo:push` | 代码推送到仓库 |
| PR 创建 | `pullrequest:created` | 创建新的 Pull Request |
| PR 更新 | `pullrequest:updated` | 更新现有的 Pull Request |

**Webhook 配置** (Bitbucket Cloud):

1. 进入仓库 Settings > Webhooks
2. 点击 "Add webhook"
3. URL: `https://your-domain/webhook`
4. Secret: 配置的 webhook 密钥（用于 HMAC-SHA256 签名验证）
5. Triggers: 选择 "Repository push" 和 "Pull request created/updated"

**API 端点**:

- `POST /webhook` - 统一 webhook（自动检测平台，推荐）
- `POST /api/webhook/bitbucket` - Bitbucket 专用 webhook（自动匹配项目）
- `POST /api/webhook/bitbucket/:project_id` - 指定项目 ID 的 Bitbucket webhook

**Bitbucket API 调用**:

| 功能 | API 端点 |
|------|----------|
| 获取提交 Diff | `GET /2.0/repositories/{workspace}/{repo}/diff/{commit}` |
| 获取 PR Diff | `GET /2.0/repositories/{workspace}/{repo}/pullrequests/{id}/diff` |
| 设置构建状态 | `POST /2.0/repositories/{workspace}/{repo}/commit/{commit}/statuses/build` |
| 发布提交评论 | `POST /2.0/repositories/{workspace}/{repo}/commit/{commit}/comments` |
| 发布 PR 评论 | `POST /2.0/repositories/{workspace}/{repo}/pullrequests/{id}/comments` |

**构建状态**:

- `INPROGRESS` - AI 审查进行中
- `SUCCESSFUL` - 审查通过（分数 >= 最低分）
- `FAILED` - 审查失败或分数低于阈值

**注意事项**:

- Bitbucket Cloud 使用 `https://api.bitbucket.org/2.0/` 作为 API 基础 URL
- Access Token 需要有仓库读取和写入权限（用于获取 diff 和发布评论）
- 签名验证使用 HMAC-SHA256 算法

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

### LDAP 配置

系统设置页面支持在线配置 LDAP 认证，无需修改配置文件。

配置项：

- **启用 LDAP**: 开启/关闭 LDAP 认证
- **LDAP 服务器**: 服务器地址
- **端口**: 默认 389 (SSL: 636)
- **Base DN**: 例如 `dc=example,dc=com`
- **Bind DN**: 绑定账号，例如 `cn=admin,dc=example,dc=com`
- **Bind 密码**: 绑定密码
- **用户过滤器**: 例如 `(uid=%s)` 或 `(sAMAccountName=%s)`
- **使用 SSL**: 是否使用 LDAPS

### 权限管理

系统支持两种角色：

| 角色 | 权限 |
|------|------|
| admin | 完全权限，可访问所有页面和所有操作 |
| user | 只读权限，只能访问 Dashboard、Review Logs、Projects（只读）、Member Analysis、Prompts（只读） |

**LDAP 用户默认角色**: user（只读）

**Admin-only 页面**:

- LLM Models
- IM Bots
- Git Credentials
- Users
- System Logs
- Settings
- Daily Reports

**Admin-only 操作**:

- 项目的创建、编辑、删除
- 审查记录的删除和重试
- 用户的编辑和删除
- 提示词模板的创建、编辑、删除、设为默认
- 日报的生成和发送

### 日报功能

系统支持自动生成每日代码审查报告，并通过 IM 机器人发送。

**功能特性**:

- 自动统计当日审查数据（提交数、通过率、平均分等）
- AI 分析生成摘要和建议
- 定时自动发送或手动触发
- 同一天多次生成会覆盖更新

**配置项**（系统设置页面）:

| 配置 | 说明 | 默认值 |
|------|------|--------|
| 启用日报 | 是否启用定时日报 | false |
| 发送时间 | 每日发送时间 | 18:00 |
| 时区 | 定时器使用的时区 | Asia/Shanghai |
| 低分阈值 | 低于此分数的提交会被标注 | 60 |
| AI 模型 | 用于生成分析的 LLM | 系统默认 |
| 通知机器人 | 接收日报的 IM 机器人（多选） | 启用日报的机器人 |
| 仅工作日 | 周末和节假日跳过日报生成 | true |
| 假日国家 | 用于假日检测的国家代码 | CN |

**假日检测**:

- 中国（CN）: 使用 `6tail/lunar-go` 库，支持法定节假日和调休
- 其他国家: 使用 `rickar/cal` 库，支持 22+ 国家的节假日
- 支持的国家代码: CN, US, GB, DE, FR, JP, AU, CA, NZ, IT, ES, NL, BE, AT, CH, SE, NO, DK, FI, PL, PT, IE, BR, NONE（仅周末）

**API**:

- `GET /api/daily-reports` - 日报列表
- `GET /api/daily-reports/:id` - 日报详情
- `POST /api/daily-reports/generate` - 手动生成（不发送通知）
- `POST /api/daily-reports/:id/resend` - 发送通知

**行为说明**:

| 操作 | 生成数据 | 保存数据库 | 发送通知 |
|------|---------|-----------|---------|
| 手动生成 | ✅ | ✅ | ❌ |
| 手动发送 | ❌ | ✅ (更新 notified_at) | ✅ |
| 定时器 | ✅ | ✅ | ✅ |

**多 Pod 部署**:

- 定时器使用数据库锁（`scheduler_locks` 表）防止重复执行
- 同一天的日报任务只会被一个 Pod 执行
- 锁有效期 10 分钟，超时自动释放

### 大型 MR/PR 分批审查

当 Merge Request 或 Pull Request 涉及大量文件或改动时，系统会自动分批审查。

**工作原理**:

1. **过滤阶段**: 自动跳过不需要审查的文件（配置文件、锁文件、生成文件等）
2. **分批阶段**: 按 token 预估将文件分组，每批控制在配置的上限内
3. **审查阶段**: 并行审查每个批次
4. **汇总阶段**: 使用加权平均（按代码改动量）计算最终分数，合并所有问题

**默认忽略的文件类型** (`DefaultIgnorePatterns`):

| 类型 | 模式 |
|------|------|
| 配置文件 | `*.json, *.yaml, *.yml, *.toml, *.xml, *.ini, *.env, *.config` |
| 锁文件 | `package-lock.json, yarn.lock, pnpm-lock.yaml, go.sum, Cargo.lock, composer.lock, Gemfile.lock, poetry.lock` |
| 压缩/打包文件 | `*.min.js, *.min.css, *.bundle.js, *.bundle.css` |
| 构建目录 | `dist/, build/, out/, target/, .next/` |
| 依赖目录 | `vendor/, node_modules/, __pycache__/, .venv/, venv/` |

**注意**: 项目级别的 `ignore_patterns` 配置会与默认忽略模式合并，不会覆盖。

**系统配置项**（数据库 `system_configs` 表）:

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `chunked_review_enabled` | `true` | 是否启用分批审查 |
| `chunked_review_threshold` | `50000` | 触发分批的字符阈值 |
| `chunked_review_max_tokens_per_batch` | `30000` | 每批最大 token 数 |

**涉及文件**:

- `/backend/internal/services/chunked_review.go` - 分批逻辑（解析、分组、汇总）
- `/backend/internal/services/ai.go` - `ReviewChunked()` 方法
- `/backend/internal/services/webhook.go` - `DefaultIgnorePatterns` 常量、`filterDiff()` 增强

### 文件上下文 (File Context)

为了解决只基于 diff 片段进行 code review 导致的误判问题（缺乏完整文件上下文），系统支持获取完整文件内容。

**功能特性**:

- 自动获取修改文件的完整源代码
- 在文件中标记变更行范围（使用 `»` 标记）
- 按改动量排序，优先获取改动最大的文件
- 支持 GitLab、GitHub、Bitbucket 三个平台

**Prompt 变量**:

- `{{file_context}}` - 完整文件内容（带行号和变更标记）
- `{{diffs}}` - 代码变更 diff
- `{{commits}}` - 提交信息

**系统配置项**（系统设置页面 → 文件上下文设置）:

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `file_context_enabled` | `false` | 是否启用文件上下文（默认关闭，需手动开启） |
| `file_context_max_file_size` | `102400` | 每个文件获取的最大字节数（默认 100KB） |
| `file_context_max_files` | `10` | 获取上下文的最大文件数量 |

**注意事项**:

- 启用此功能会增加 token 消耗（每个文件的完整内容会发送给 LLM）
- 建议在 prompt 中添加说明，指导 AI 结合上下文进行审查
- 对于非常大的文件（超过 max_file_size）会被跳过

**涉及文件**:

- `/backend/internal/services/file_context.go` - 文件上下文服务（获取、解析、格式化）
- `/backend/internal/services/system_config.go` - `FileContextConfig` 配置
- `/backend/internal/services/webhook.go` - 集成到审查流程
- `/backend/internal/services/ai.go` - `{{file_context}}` 变量替换

### 优雅关闭 (Graceful Shutdown)

服务器支持优雅关闭，确保在收到终止信号时正确清理资源。

**触发信号**: SIGINT (Ctrl+C), SIGTERM (Docker/K8s)

**关闭流程**:

1. 停止所有定时器调度器（DailyReport、LogCleanup、Retry）
2. 等待进行中的 HTTP 请求完成（超时 30 秒）
3. 关闭数据库连接
4. 输出日志确认退出

**涉及文件**:

- `/backend/cmd/server/main.go` - 信号监听和关闭协调
- `/backend/internal/services/daily_report.go` - `StopScheduler()`
- `/backend/internal/services/system_log.go` - `StopLogCleanupScheduler()`
- `/backend/internal/services/retry.go` - `StopRetryScheduler()`

## 待完成功能

（暂无）
