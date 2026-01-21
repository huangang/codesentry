# CodeSentry

AI-powered Code Review Platform for GitHub and GitLab.

[中文文档](./README_zh.md)

## Features

- **AI Code Review**: Automatically review code changes using OpenAI-compatible models
- **Multi-Platform Support**: GitHub and GitLab webhook integration
- **Dashboard**: Visual statistics and metrics for code review activities
- **Review History**: Track all code reviews with detailed logs
- **Project Management**: Manage multiple repositories
- **LLM Configuration**: Configure multiple AI models with custom endpoints
- **IM Notifications**: Send review results to DingTalk, Feishu, WeCom, Slack, or custom webhooks
- **Authentication**: Local authentication and LDAP support
- **Multi-Database**: SQLite for development, MySQL/PostgreSQL for production
- **Internationalization**: Support for English and Chinese

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 18+
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
# Build and run
docker-compose up --build

# Or run in background
docker-compose up -d --build
```

Access the application at `http://localhost:8080`

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

## Webhook Setup

### GitHub

1. Go to Repository Settings > Webhooks > Add webhook
2. Payload URL: `https://your-domain/api/webhook/github/{project_id}`
3. Content type: `application/json`
4. Secret: Your configured webhook secret
5. Events: Select "Pull requests" and "Pushes"

### GitLab

1. Go to Project Settings > Webhooks
2. URL: `https://your-domain/api/webhook/gitlab/{project_id}`
3. Secret Token: Your configured webhook secret
4. Trigger: Push events, Merge request events

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login
- `GET /api/auth/config` - Get auth config
- `GET /api/auth/me` - Get current user
- `POST /api/auth/logout` - Logout

### Projects
- `GET /api/projects` - List projects
- `POST /api/projects` - Create project
- `GET /api/projects/:id` - Get project
- `PUT /api/projects/:id` - Update project
- `DELETE /api/projects/:id` - Delete project

### Review Logs
- `GET /api/review-logs` - List review logs
- `GET /api/review-logs/:id` - Get review detail

### Dashboard
- `GET /api/dashboard/stats` - Get statistics

### LLM Config
- `GET /api/llm-configs` - List LLM configs
- `POST /api/llm-configs` - Create LLM config
- `PUT /api/llm-configs/:id` - Update LLM config
- `DELETE /api/llm-configs/:id` - Delete LLM config

### IM Bots
- `GET /api/im-bots` - List IM bots
- `POST /api/im-bots` - Create IM bot
- `PUT /api/im-bots/:id` - Update IM bot
- `DELETE /api/im-bots/:id` - Delete IM bot

### Webhooks
- `POST /api/webhook/github/:project_id` - GitHub webhook
- `POST /api/webhook/gitlab/:project_id` - GitLab webhook

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
- Go 1.23
- Gin (HTTP framework)
- GORM (ORM)
- JWT authentication
- LDAP support

### Frontend
- React 18
- TypeScript
- Ant Design 5
- Recharts
- Zustand (state management)
- React Router 7
- react-i18next (internationalization)

## License

MIT
