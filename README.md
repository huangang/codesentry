# CodeSentry

<div align="center">
  <img src="https://raw.githubusercontent.com/huangang/codesentry/main/frontend/public/codesentry-icon.png" alt="CodeSentry Logo" width="120" height="120">
</div>

AI-powered Code Review Platform for GitHub, GitLab, and Bitbucket.

[中文文档](./README_zh.md)

## Features

- **AI Code Review**: Native API support for OpenAI, Anthropic (Claude), Ollama, Google Gemini, and Azure OpenAI
- **File Context**: Fetch full file content to provide better context for AI review, reducing false positives
- **Chunked Review**: Automatically splits large MRs/PRs into batches for optimal review quality
- **Smart Filtering**: Auto-skips config files, lock files, and generated files (customizable)
- **Auto-Scoring**: Automatically appends scoring instructions if custom prompts lack them
- **Commit Comments**: Post AI review results as comments on commits (GitLab/GitHub)
- **Commit Status**: Set commit status to block merges when score is below threshold (GitLab/GitHub)
- **Sync Review API**: Synchronous review endpoint for Git pre-receive hooks to block pushes
- **Duplicate Prevention**: Skip already reviewed commits to avoid redundant processing
- **Multi-Platform Support**: GitHub, GitLab, and Bitbucket webhook integration with multi-level project path support
- **Dashboard**: Visual statistics and metrics for code review activities
- **Real-time Updates**: SSE-powered live status updates (pending → analyzing → completed) without page refresh
- **Review History**: Track all code reviews with detailed logs and direct links to commits/MRs
- **Project Management**: Manage multiple repositories

## Preview

![CodeSentry Dashboard](https://raw.githubusercontent.com/huangang/codesentry/main/frontend/public/dashboard-preview.png)

- **LLM Configuration**: Configure multiple AI models with native SDK integration (no proxy required for Anthropic/Gemini)
- **Prompt Templates**: System and custom prompt templates with copy functionality
- **IM Notifications**: Send review results to DingTalk, Feishu, WeCom, Slack, Discord, Microsoft Teams, Telegram
- **Daily Reports**: Automated daily code review summary with AI analysis, sent via IM bots
- **Error Notifications**: Real-time error alerts via IM bots
- **Git Credentials**: Auto-create projects from webhooks with credential management
- **System Logging**: Comprehensive logging for webhook events, errors, and system operations
- **Authentication**: Local authentication and LDAP support (configurable via web UI)
- **Role-based Access Control**: Admin, Developer, and User roles with granular permissions
- **Multi-Database**: SQLite for development, MySQL/PostgreSQL for production
- **Async Task Queue**: Optional Redis-based async processing for AI reviews (graceful fallback to sync mode)
- **Internationalization**: Support for English and Chinese (including DatePicker localization)
- **Responsive Design**: Mobile-friendly interface with adaptive layouts for phones and tablets
- **Dark Mode**: Toggle between light and dark themes, with preference persistence
- **Global Search**: Cross-project search for reviews and projects from the header
- **Multi-Language Review Prompts**: Auto-detect programming language from diffs and inject language-specific review guidelines (Go, Python, JS/TS, Java, Rust, Ruby, PHP, Swift, Kotlin, C/C++)
- **Batch Operations**: Batch retry and batch delete for review logs
- **Real-time Notifications**: SSE-powered notification bell with unread badge and live review events
- **Reports**: Weekly/monthly report API with period comparison, daily trends, and author rankings
- **Issue Tracker Integration**: Auto-create Jira, Linear, or GitHub Issues when review score is below threshold
- **Rule Engine**: Automated CI/CD policies with conditions (score_below, files_changed_above, has_keyword) and actions (block, warn, notify)
- **Prometheus Metrics**: `/metrics` endpoint for monitoring
- **Audit Logging**: Automatic audit logging for all admin write operations
- **Review Diff Cache**: SHA-256 hash deduplication to skip already-reviewed diffs
- **CSV Export**: Export review logs as CSV for offline analysis

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+
- Docker (optional)

### Development Setup

#### Backend

```bash
cd backend

# Create config file
cp ../config.yaml.example config.yaml
# Edit config.yaml with your settings

# Run
go run ./cmd/server
```

#### Frontend

```bash
cd frontend

# Install dependencies
npm install

# Run development server
npm run dev
```

Access the application at `http://localhost:5173`

**Default credentials**: `admin` / `admin`

### Docker Deployment

```bash
# Pull from Docker Hub
docker pull huangangzhang/codesentry:latest

# Or pull from GitHub Container Registry
docker pull ghcr.io/huangang/codesentry:latest
```

**Choose your database:**

```bash
# MySQL (default, recommended for production)
docker-compose up -d

# SQLite (simple, single file)
docker-compose -f docker-compose.sqlite.yml up -d

# PostgreSQL
docker-compose -f docker-compose.postgres.yml up -d
```

**Or run directly (SQLite):**

```bash
docker run -d -p 8080:8080 -v codesentry-data:/app/data huangangzhang/codesentry:latest
```

For local development (build from source):

```bash
docker-compose -f docker-compose.dev.yml up --build
```

Access the application at `http://localhost:8080`

### Build Script (Local)

```bash
# One-command build (frontend + backend combined)
./build.sh

# Run the binary
./codesentry
```

This builds frontend, embeds it into the Go binary, producing a single executable.

## Configuration

Copy `config.yaml.example` to `config.yaml` and update:

```yaml
server:
  port: 8080
  mode: release  # debug, release, test

database:
  driver: sqlite   # sqlite, mysql, postgres
  dsn: data/codesentry.db
  # For MySQL: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
  # For PostgreSQL: host=localhost user=postgres password=xxx dbname=codesentry port=5432 sslmode=disable

jwt:
  secret: your-secret-key-change-in-production
  expire_hour: 24
```

### Session & Token Expiration

CodeSentry uses a **short-lived access token** (JWT) plus a **long-lived refresh token** for silent re-login.

- **Access token**: returned by `POST /api/auth/login`, stored by frontend in `localStorage` and sent as `Authorization: Bearer <token>`.
- **Refresh token**: stored in a **httpOnly cookie** (not accessible from JavaScript), used by `POST /api/auth/refresh`.

Default expirations (configurable via System Config in DB):

- `auth_access_token_expire_hours` (default: `2`)
- `auth_refresh_token_expire_hours` (default: `720` = 30 days)

`jwt.expire_hour` in `config.yaml` is used as a fallback default for access token expiration.

> **Note**: When the access token expires, the frontend will automatically call `/api/auth/refresh` and retry the original request. Only when refresh fails will it redirect to `/login`.

The frontend also performs **proactive refresh** before token expiration (default: refresh 5 minutes before access token expires) to reduce user-visible 401 interruptions.

### CORS (Important for Refresh Cookie)

Because refresh token is stored in a cookie, deployments that serve frontend and backend on different origins must configure CORS correctly:

- `Access-Control-Allow-Credentials: true` is required
- `Access-Control-Allow-Origin` **cannot** be `*`
- Prefer an explicit origin allowlist in production

> **Note**: All business configurations (LLM models, LDAP, prompts, IM bots, Git credentials) are managed via the web UI and stored in the database.

## Webhook Setup

### Recommended: Unified Webhook (Auto-detect)

Use a single webhook URL for GitLab, GitHub, and Bitbucket:

```
https://your-domain/webhook
# or
https://your-domain/review/webhook
```

The system automatically detects the platform via request headers.

### GitHub

1. Go to Repository Settings > Webhooks > Add webhook
2. Payload URL: `https://your-domain/webhook`
3. Content type: `application/json`
4. Secret: Your configured webhook secret
5. Events: Select "Pull requests" and "Pushes"

### GitLab

1. Go to Project Settings > Webhooks
2. URL: `https://your-domain/webhook`
3. Secret Token: Your configured webhook secret
4. Trigger: Push events, Merge request events

### Bitbucket

1. Go to Repository Settings > Webhooks > Add webhook
2. URL: `https://your-domain/webhook`
3. Secret: Your configured webhook secret (for HMAC-SHA256 signature)
4. Triggers: Select "Repository push" and "Pull request created/updated"

## API Endpoints

### Authentication

- `POST /api/auth/login` - Login
- `POST /api/auth/refresh` - Refresh access token (uses httpOnly refresh cookie)
- `GET /api/auth/config` - Get auth config
- `GET /api/auth/me` - Get current user
- `POST /api/auth/logout` - Logout (revokes refresh token and clears cookie)
- `POST /api/auth/change-password` - Change password (local users only)

### System Config (Admin)

- `GET /api/system-config/auth-session` - Get auth session config
- `PUT /api/system-config/auth-session` - Update auth session config

### Projects

- `GET /api/projects` - List projects
- `POST /api/projects` - Create project
- `GET /api/projects/:id` - Get project
- `PUT /api/projects/:id` - Update project
- `DELETE /api/projects/:id` - Delete project

### Review Logs

- `GET /api/review-logs` - List review logs
- `GET /api/review-logs/:id` - Get review detail
- `POST /api/review-logs/:id/retry` - Retry failed review (admin only)
- `DELETE /api/review-logs/:id` - Delete review log (admin only)

### Real-time Events (SSE)

- `GET /api/events/reviews` - Stream review status updates (requires `token` query param)

### Users

- `GET /api/users` - List users (admin only)
- `PUT /api/users/:id` - Update user (admin only)
- `DELETE /api/users/:id` - Delete user (admin only)

### Dashboard

- `GET /api/dashboard/stats` - Get statistics

### Global Search

- `GET /api/search?q=<query>&limit=<n>` - Search across reviews and projects

### Reports

- `GET /api/reports?period=weekly|monthly&project_id=N` - Period stats with trend and author rankings

### Review Logs

- `GET /api/review-logs` - List review logs (supports score range, status, author, date filters)
- `GET /api/review-logs/:id` - Get review detail
- `GET /api/review-logs/export` - Export review logs as CSV (admin only)
- `POST /api/review-logs/:id/retry` - Retry failed review (admin only)
- `POST /api/review-logs/batch-retry` - Batch retry (admin only)
- `POST /api/review-logs/batch-delete` - Batch delete (admin only)
- `DELETE /api/review-logs/:id` - Delete review log (admin only)

### Issue Trackers

- `GET /api/issue-trackers` - List issue tracker integrations (admin only)
- `POST /api/issue-trackers` - Create integration (admin only)
- `PUT /api/issue-trackers/:id` - Update integration (admin only)
- `DELETE /api/issue-trackers/:id` - Delete integration (admin only)

### Review Rules (CI/CD Policies)

- `GET /api/review-rules` - List review rules (admin only)
- `POST /api/review-rules` - Create rule (admin only)
- `PUT /api/review-rules/:id` - Update rule (admin only)
- `DELETE /api/review-rules/:id` - Delete rule (admin only)
- `POST /api/review-rules/evaluate/:id` - Test rules against a review log (admin only)

### Member Analysis

- `GET /api/members` - List member statistics
- `GET /api/members/detail` - Get member detail with trend and project stats
- `GET /api/members/overview` - Get team overview (total stats, trend, score distribution, top members)

### LLM Config

- `GET /api/llm-configs` - List LLM configs
- `GET /api/llm-configs/active` - List active LLM configs (for project selection)
- `POST /api/llm-configs` - Create LLM config
- `PUT /api/llm-configs/:id` - Update LLM config
- `DELETE /api/llm-configs/:id` - Delete LLM config

### Prompt Templates

- `GET /api/prompts` - List prompt templates
- `GET /api/prompts/:id` - Get prompt template detail
- `GET /api/prompts/default` - Get default prompt template
- `GET /api/prompts/active` - List active prompt templates
- `POST /api/prompts` - Create prompt template (admin only)
- `PUT /api/prompts/:id` - Update prompt template (admin only)
- `DELETE /api/prompts/:id` - Delete prompt template (admin only)
- `POST /api/prompts/:id/set-default` - Set as default template (admin only)

### IM Bots

- `GET /api/im-bots` - List IM bots
- `POST /api/im-bots` - Create IM bot
- `PUT /api/im-bots/:id` - Update IM bot
- `DELETE /api/im-bots/:id` - Delete IM bot

### Daily Reports

- `GET /api/daily-reports` - List daily reports
- `GET /api/daily-reports/:id` - Get daily report detail
- `POST /api/daily-reports/generate` - Generate daily report (manual, no notification)
- `POST /api/daily-reports/:id/resend` - Send/resend notification

### Webhooks

- `POST /webhook` - **Unified webhook (auto-detect GitLab/GitHub/Bitbucket, recommended)**
- `POST /review/webhook` - Alias for unified webhook
- `POST /api/webhook` - Unified webhook under /api prefix
- `POST /api/review/webhook` - Alias under /api prefix
- `POST /api/webhook/gitlab` - GitLab webhook (auto-detect project by URL)
- `POST /api/webhook/github` - GitHub webhook (auto-detect project by URL)
- `POST /api/webhook/gitlab/:project_id` - GitLab webhook (with project ID)
- `POST /api/webhook/github/:project_id` - GitHub webhook (with project ID)
- `POST /api/webhook/bitbucket` - Bitbucket webhook (auto-detect project by URL)
- `POST /api/webhook/bitbucket/:project_id` - Bitbucket webhook (with project ID)

### Sync Review (for Git Hooks)

- `POST /review/sync` - Synchronous code review for pre-receive hooks
- `POST /api/review/sync` - Same endpoint under /api prefix
- `GET /review/score?commit_sha=xxx` - Query review status/score by commit SHA
- `GET /api/review/score?commit_sha=xxx` - Same endpoint under /api prefix

Request body:

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

Response:

```json
{
  "passed": true,
  "score": 85,
  "min_score": 60,
  "message": "Score: 85/100 (min: 60)",
  "review_id": 123
}
```

See `scripts/pre-receive-hook.sh` for GitLab pre-receive hook example.

### System Logs

- `GET /api/system-logs` - List system logs
- `GET /api/system-logs/modules` - Get module list
- `GET /api/system-logs/retention` - Get log retention days
- `PUT /api/system-logs/retention` - Set log retention days
- `POST /api/system-logs/cleanup` - Manually cleanup old logs

### Health Check & Metrics

- `GET /health` - Service health check
- `GET /metrics` - Prometheus metrics

## Project Structure

```
codesentry/
├── backend/
│   ├── cmd/server/          # Application entry point
│   ├── internal/
│   │   ├── config/          # Configuration
│   │   ├── handlers/        # HTTP handlers
│   │   ├── middleware/      # Auth, CORS middleware
│   │   ├── models/          # Database models
│   │   ├── services/        # Business logic
│   │   └── utils/           # Utilities
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── i18n/            # Internationalization
│   │   ├── layouts/         # Layout components
│   │   ├── pages/           # Page components
│   │   ├── services/        # API services
│   │   ├── stores/          # State management
│   │   └── types/           # TypeScript types
│   └── package.json
├── Dockerfile
├── docker-compose.yml
├── config.yaml.example
├── README.md
└── README_zh.md
```

## Tech Stack

### Backend

- Go 1.24
- Gin v1.11 (HTTP framework)
- GORM v1.31 (ORM)
- JWT authentication
- LDAP support

### Frontend

- React 19
- TypeScript 5.9
- Ant Design 5
- TanStack Query (data fetching & caching)
- Recharts
- Zustand (state management)
- React Router 7
- react-i18next (internationalization)
- react-markdown (review result rendering)

## License

MIT
