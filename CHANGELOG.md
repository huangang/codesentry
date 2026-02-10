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

### Changed

- **Unified API Response**: All handlers migrated to `pkg/response` package with standardized `{code, data, message}` envelope
- **Structured Logging**: All 16 service files migrated from `log.Printf` to `pkg/logger` (zerolog-based `logger.Infof/Errorf/Warnf`)
- **Models Split**: Decomposed monolithic `models.go` (290 lines) into 13 individual model files for better maintainability
- **Main Entry Split**: Split `main.go` into `bootstrap.go` (initialization) and `routes.go` (routing)
- **Frontend API Layer**: Added automatic response envelope unwrapping in axios interceptor

### Fixed

- **Feedback Score Parsing**: Allow 0/100 as a valid score in AI feedback responses
- **Unreviewable Status**: Set `review_status` to `unreviewable` when feedback score is 0 (e.g., diff unavailable)

### Security

- JWT-based authentication
- LDAP integration support
- Webhook secret validation
- Duplicate review prevention for same commit across branches

[Unreleased]: https://github.com/huangang/codesentry/commits/main
