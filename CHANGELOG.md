# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **AI Code Review**: Support for OpenAI, Anthropic (Claude), Ollama, Google Gemini, and Azure OpenAI
- **File Context**: Fetch full file content for better AI review context
- **Chunked Review**: Automatically split large MRs/PRs into batches
- **Smart Filtering**: Auto-skip config files, lock files, and generated files
- **Multi-Platform Webhooks**: Unified webhook for GitHub, GitLab, and Bitbucket
- **Real-time Updates**: SSE-powered live status updates
- **Dashboard**: Visual statistics and metrics
- **Review History**: Track all code reviews with detailed logs
- **Project Management**: Manage multiple repositories
- **Member Analysis**: GitHub-style contribution heatmap and statistics
- **LLM Configuration**: Configure multiple AI models
- **Prompt Templates**: System and custom prompt templates
- **IM Notifications**: DingTalk, Feishu, WeCom, Slack, Discord, MS Teams, Telegram
- **Daily Reports**: Automated daily code review summary with AI analysis
- **Git Credentials**: Auto-create projects from webhooks
- **System Logging**: Comprehensive logging for all events
- **Authentication**: Local and LDAP authentication
- **Role-based Access Control**: Admin and User roles
- **Multi-Database Support**: SQLite, MySQL, PostgreSQL
- **Async Task Queue**: Optional Redis-based async processing with SSE notifications
- **Sync Review API**: Synchronous review endpoint for Git pre-receive hooks
- **Internationalization**: English and Chinese support
- **Responsive Design**: Mobile-friendly interface
- **Dark Mode**: Light and dark theme toggle

- **Auth Refresh**: Refresh-token based session renewal (`POST /api/auth/refresh`) with httpOnly cookie storage
- **Refresh Token Rotation**: Server-side refresh token storage with rotation and revocation support
- **Proactive Session Refresh**: Frontend proactively refreshes access token before expiration to reduce 401 interruptions
- **Auth Session Settings UI**: Admin settings page now supports configuring access/refresh token expiration

### Changed

- **Unified API Response**: All handlers migrated to `pkg/response` package with standardized `{code, data, message}` envelope
- **Structured Logging**: All 16 service files migrated from `log.Printf` to `pkg/logger` (zerolog-based `logger.Infof/Errorf/Warnf`)
- **Models Split**: Decomposed monolithic `models.go` (290 lines) into 13 individual model files for better maintainability
- **Main Entry Split**: Split `main.go` into `bootstrap.go` (initialization) and `routes.go` (routing)
- **Frontend API Layer**: Added automatic response envelope unwrapping in axios interceptor
- **Notification Refactor**: Refactored notification service using Strategy Pattern with `NotificationAdapter` interface, reducing ~600 lines to ~90 lines + 8 platform adapters
- **DB Query Optimization**: Consolidated 9 separate DB queries in daily report stats collection into a single aggregate SQL query
- **Regex Pre-compilation**: Moved regex patterns in `ai.go` and `file_context.go` to package-level pre-compiled variables
- **Function Extraction Dedup**: Extracted shared `matchBoundariesToRanges` helper to eliminate ~80 lines of duplicated code across 4 language extractors
- **HTTP Timeout**: Added 10-second timeout to notification HTTP client to prevent goroutine leaks
- **Dockerfile Hardening**: Upgraded to Node 22 & Alpine 3.21, added non-root user, added `-ldflags="-s -w"` for ~30% smaller binary
- **Vite Code Splitting**: Added `manualChunks` configuration to split vendor dependencies (antd, react, recharts, utils) into separate cacheable bundles
- **Dependency Upgrades**: Updated `anthropic-sdk-go` (v1.19→v1.26), `genai` (v1.43→v1.47), `ollama` (v0.15→v0.17), `asynq` (v0.25→v0.26)

### Fixed

- **Feedback Score Parsing**: Allow 0/100 as a valid score in AI feedback responses
- **Unreviewable Status**: Set `review_status` to `unreviewable` when feedback score is 0 (e.g., diff unavailable)

### Security

- JWT-based authentication
- Refresh-token based session renewal with httpOnly cookie
- LDAP integration support
- Webhook secret validation
- Duplicate review prevention for same commit across branches

[Unreleased]: https://github.com/huangang/codesentry/commits/main
