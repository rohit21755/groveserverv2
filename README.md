# Gamified Campus Ambassador Platform - Backend

Backend implementation for the Gamified Campus Ambassador Platform using Go, Chi, GraphQL, PostgreSQL, Redis, and WebSockets.

## Architecture

The project follows clean architecture principles:

- **Presentation Layer**: `/cmd/api` - REST APIs, GraphQL gateway, WebSocket handlers
- **Business Logic Layer**: `/internal/store` - Repository pattern
- **Infrastructure Layer**: `/internal/db`, `/internal/env` - Database connections, configuration
- **Data Layer**: PostgreSQL with migrations

## Prerequisites

- Go 1.24.3 or higher
- Docker and Docker Compose
- Make (optional, for using Makefile)

## Quick Start

### Using Docker (Recommended)

1. Copy `.env.example` to `.env` and update values if needed:
   ```bash
   cp .env.example .env
   ```

2. Start services (PostgreSQL, Redis, API):
   ```bash
   make docker-up
   ```

3. Run migrations:
   ```bash
   make migrate-up
   ```

4. View logs:
   ```bash
   make docker-logs
   ```

### Local Development

1. Start only PostgreSQL and Redis:
   ```bash
   docker-compose up -d postgres redis
   ```

2. Copy `.env.example` to `.env` and configure:
   ```bash
   cp .env.example .env
   ```

3. Run the application:
   ```bash
   make run
   # or
   go run ./cmd/api
   ```

## Makefile Commands

- `make build` - Build the application
- `make run` - Run the application locally
- `make test` - Run tests
- `make docker-build` - Build Docker image
- `make docker-up` - Start Docker containers
- `make docker-down` - Stop Docker containers
- `make docker-logs` - View Docker logs
- `make migrate-up` - Run database migrations
- `make migrate-down` - Rollback last migration
- `make migrate-create NAME=name` - Create new migration
- `make dev` - Start dev environment (docker + local app)
- `make deps` - Download dependencies
- `make fmt` - Format code
- `make lint` - Run linter

## Project Structure

```
.
├── cmd/
│   └── api/              # Application entry point
├── internal/
│   ├── db/              # Database connections (PostgreSQL, Redis)
│   ├── env/             # Configuration management
│   ├── router/          # HTTP routing (REST, WebSocket)
│   │   ├── api/         # REST API handlers
│   │   └── ws/          # WebSocket handlers
│   └── store/           # Repository pattern (to be implemented)
├── migrations/          # Database migration files
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

## API Endpoints

### Mobile (React Native)

- `POST /api/auth/login` - User login
- `POST /api/auth/register` - User registration
- `GET /api/user/me` - Get current user
- `GET /api/tasks` - Get tasks
- `POST /api/tasks/{id}/submit` - Submit task
- `GET /api/feed` - Get completed task feed
- `GET /api/leaderboard/pan-india` - Pan India leaderboard
- `GET /api/chat/rooms` - Get chat rooms
- `GET /api/notifications` - Get notifications

### Admin

- `POST /admin/tasks` - Create task
- `PUT /admin/tasks/{id}` - Update task
- `GET /admin/submissions` - Get submissions
- `POST /admin/submissions/{id}/approve` - Approve submission
- `POST /admin/submissions/{id}/reject` - Reject submission

## WebSocket Endpoints

- `/ws/chat` - Chat WebSocket
- `/ws/leaderboard` - Leaderboard updates
- `/ws/notifications` - Real-time notifications

## Environment Variables

See `.env.example` for all available configuration options.

## Database Migrations

Migrations are managed using `golang-migrate`. Create new migrations with:

```bash
make migrate-create NAME=create_users_table
```

## Development

The project structure is set up with placeholder handlers. Implement the business logic in:

- `/internal/store` - Repository implementations
- `/internal/router/api` - API handler implementations
- `/internal/router/ws` - WebSocket handler implementations

## License

[Your License Here]
