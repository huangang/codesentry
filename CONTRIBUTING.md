# Contributing to CodeSentry

Thank you for your interest in contributing to CodeSentry! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and considerate in all interactions. We welcome contributors of all backgrounds and experience levels.

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 20+
- Docker (optional)

### Development Setup

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/your-username/codesentry.git
   cd codesentry
   ```

2. Set up the backend:
   ```bash
   cd backend
   cp ../config.yaml.example config.yaml
   # Edit config.yaml with your settings
   go run ./cmd/server
   ```

3. Set up the frontend:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

4. Access the application at `http://localhost:5173`

## How to Contribute

### Reporting Bugs

- Check if the issue already exists
- Use the bug report template
- Include steps to reproduce, expected behavior, and actual behavior
- Include environment details (OS, Go version, Node version)

### Suggesting Features

- Check if the feature has already been suggested
- Describe the use case and expected behavior
- Explain why this feature would be useful

### Pull Requests

1. Create a new branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding standards

3. Write or update tests as needed

4. Run tests and linting:
   ```bash
   # Backend
   make test
   make lint

   # Frontend
   cd frontend && npm run lint
   ```

5. Commit your changes with a clear message:
   ```bash
   git commit -m "feat: add new feature description"
   ```

6. Push and create a pull request

### Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

Examples:
```
feat: add Slack notification support
fix: resolve webhook duplicate processing issue
docs: update API documentation
```

## Coding Standards

### Go (Backend)

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Run `go vet` before committing
- Write unit tests for new functionality
- Keep functions focused and concise

### TypeScript/React (Frontend)

- Use TypeScript for all new code
- Follow ESLint rules (run `npm run lint`)
- Use functional components with hooks
- Keep components small and focused
- Use meaningful variable and function names

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
│   │   ├── components/      # Shared components
│   │   ├── pages/           # Page components
│   │   ├── services/        # API services
│   │   ├── stores/          # State management
│   │   └── types/           # TypeScript types
│   └── package.json
└── ...
```

## Testing

### Backend Tests

```bash
cd backend
go test ./...
```

### Frontend Tests

```bash
cd frontend
npm run lint
npm run build
```

## Questions?

Feel free to open an issue for any questions about contributing.

Thank you for contributing to CodeSentry!
