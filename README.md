# Gamified Campus Ambassador Platform - Backend

Backend implementation for the Gamified Campus Ambassador Platform using Go, Chi, GraphQL, PostgreSQL, Redis, and WebSockets.

## Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [API Documentation](#api-documentation)
- [WebSocket Endpoints](#websocket-endpoints)
- [GraphQL API](#graphql-api)
- [Database Schema](#database-schema)
- [Environment Variables](#environment-variables)
- [Development](#development)
- [Deployment](#deployment)

## Architecture

The project follows clean architecture principles:

- **Presentation Layer**: `/cmd/api` - REST APIs, GraphQL gateway, WebSocket handlers
- **Business Logic Layer**: `/internal/store` - Repository pattern for data access
- **Infrastructure Layer**: `/internal/db`, `/internal/env` - Database connections, configuration
- **Data Layer**: PostgreSQL with migrations, Redis for caching/pub-sub

### Technology Stack

- **Language**: Go 1.24.3+
- **Web Framework**: Chi Router
- **Database**: PostgreSQL
- **Cache/Pub-Sub**: Redis
- **API Documentation**: Swagger/OpenAPI
- **GraphQL**: gqlgen
- **WebSocket**: Gorilla WebSocket
- **File Storage**: AWS S3
- **Authentication**: JWT (JSON Web Tokens)

## Prerequisites

- Go 1.24.3 or higher
- Docker and Docker Compose
- Make (optional, for using Makefile)
- AWS Account (for S3 file storage)

## Quick Start

### Using Docker (Recommended)

1. Copy `.env.example` to `.env` and update values:
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

5. Access the API:
   - API: http://localhost:8080
   - Swagger: http://localhost:8080/swagger/index.html
   - GraphQL Playground: http://localhost:8080/playground
   - Health Check: http://localhost:8080/health

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
│   └── api/                    # Application entry point
│       └── main.go
├── internal/
│   ├── auth/                  # JWT authentication utilities
│   │   └── jwt.go
│   ├── db/                    # Database connections
│   │   ├── postgres.go        # PostgreSQL connection
│   │   └── redis.go           # Redis connection
│   ├── env/                   # Configuration management
│   │   └── config.go
│   ├── router/                # HTTP routing
│   │   ├── api/               # REST API handlers
│   │   │   ├── auth.go        # Authentication endpoints
│   │   │   ├── user.go        # User management endpoints
│   │   │   ├── task.go        # Task endpoints
│   │   │   ├── feed.go        # Feed endpoints
│   │   │   ├── leaderboard.go # Leaderboard endpoints
│   │   │   ├── admin.go       # Admin endpoints
│   │   │   ├── routes.go      # Route setup
│   │   │   └── middleware.go  # JWT middleware
│   │   ├── ws/                # WebSocket handlers
│   │   │   ├── routes.go      # WebSocket route setup
│   │   │   └── leaderboard.go # Leaderboard WebSocket
│   │   └── graphql/           # GraphQL handlers
│   │       └── handler.go
│   ├── store/                 # Repository pattern
│   │   ├── user.go            # User repository
│   │   ├── task.go            # Task repository
│   │   ├── submission.go      # Submission repository
│   │   ├── feed.go            # Feed repository
│   │   ├── leaderboard.go     # Leaderboard repository
│   │   └── xp.go              # XP management repository
│   └── storage/               # File storage
│       └── s3.go              # AWS S3 integration
├── graph/                     # GraphQL schema and resolvers
│   ├── schema.graphqls        # GraphQL schema definition
│   ├── resolver.go            # GraphQL resolver
│   └── generated/             # Generated GraphQL code
├── migrations/                # Database migration files
├── docs/                      # Swagger documentation
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```

## API Documentation

### Base URL
```
http://localhost:8080/api
```

### Authentication

Most endpoints require JWT authentication. Include the token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

### Swagger Documentation

Interactive API documentation is available at:
```
http://localhost:8080/swagger/index.html
```

---

## REST API Endpoints

### Authentication Endpoints

#### POST `/api/auth/register`
Register a new user.

**Request Body:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "password123",
  "phone": "+1234567890",
  "state_id": "uuid",
  "college_id": "uuid",
  "referral_code": "ABC123",  // Optional
  "resume": "file",            // Optional - multipart/form-data
  "profile_pic": "file"       // Optional - multipart/form-data
}
```

**Response:**
```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com",
    "xp": 0,
    "level": 1,
    "referral_code": "XYZ789"
  }
}
```

**Features:**
- Auto-generates unique referral code
- Hashes password with bcrypt
- Uploads resume and profile picture to S3 (if provided)
- Automatically logs in user after registration

#### POST `/api/auth/login`
Login user and get JWT token.

**Request Body:**
```json
{
  "email": "john@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com",
    "xp": 100,
    "level": 2
  }
}
```

---

### User Endpoints (Protected)

#### GET `/api/user/me`
Get current authenticated user's profile.

**Response:**
```json
{
  "id": "uuid",
  "name": "John Doe",
  "email": "john@example.com",
  "xp": 100,
  "level": 2,
  "avatar_url": "https://...",
  "resume_url": "https://...",
  "state_id": "uuid",
  "college_id": "uuid"
}
```

#### GET `/api/user/{id}`
Get user profile by ID (public endpoint).

**Response:**
```json
{
  "user": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com",
    "xp": 100,
    "level": 2,
    "avatar_url": "https://...",
    "resume_url": "https://...",
    "state_id": "uuid",
    "college_id": "uuid"
  },
  "completed_tasks": [...],
  "following_count": 10,
  "followers_count": 25,
  "state_name": "Maharashtra",
  "college_name": "IIT Bombay"
}
```

#### POST `/api/user/{id}/follow`
Follow a user.

**Response:**
```json
{
  "message": "Successfully followed user",
  "following_id": "uuid"
}
```

#### POST `/api/user/{id}/unfollow`
Unfollow a user.

**Response:**
```json
{
  "message": "Successfully unfollowed user",
  "following_id": "uuid"
}
```

#### POST `/api/user/resume`
Upload resume (if not provided during registration).

**Request:** `multipart/form-data` with `resume` file

**Response:**
```json
{
  "message": "Resume uploaded successfully",
  "resume_url": "https://..."
}
```

#### PUT `/api/user/resume`
Update existing resume.

**Request:** `multipart/form-data` with `resume` file

#### POST `/api/user/profile-pic`
Upload profile picture (if not provided during registration).

**Request:** `multipart/form-data` with `profile_pic` file

#### PUT `/api/user/profile-pic`
Update existing profile picture.

**Request:** `multipart/form-data` with `profile_pic` file

---

### Task Endpoints (Protected)

#### GET `/api/tasks`
Get all tasks assigned to the authenticated user.

**Query Parameters:**
- None

**Response:**
```json
[
  {
    "id": "uuid",
    "title": "Complete Social Media Post",
    "description": "Post about the platform",
    "xp": 100,
    "type": "social",
    "proof_type": "image",
    "priority": "normal",
    "start_at": "2026-01-26T00:00:00Z",
    "end_at": "2026-01-30T23:59:59Z",
    "is_flash": false,
    "is_weekly": false
  }
]
```

**Features:**
- Only returns active tasks (not expired, started)
- Filters tasks based on user's state/college assignment

#### POST `/api/tasks/{id}/submit`
Submit a task with proof (image or video).

**Request:** `multipart/form-data` with `proof` file

**Supported File Types:**
- Images: JPG, JPEG, PNG, GIF, WEBP
- Videos: MP4, MOV, AVI, MKV, WEBM

**Response:**
```json
{
  "id": "uuid",
  "task_id": "uuid",
  "user_id": "uuid",
  "proof_url": "https://...",
  "status": "pending",
  "created_at": "2026-01-26T12:00:00Z"
}
```

**Features:**
- Validates task exists and is active
- Prevents duplicate submissions
- Allows resubmission if previous submission was rejected (if task deadline hasn't passed)
- Uploads proof to S3

---

### Feed Endpoints

#### GET `/api/feed`
Get feed items with pagination.

**Query Parameters:**
- `type` (optional): `pan-india`, `state`, `college` (default: `pan-india`)
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 20, max: 100)

**Note:** For `state` and `college` feeds, JWT authentication is required.

**Response:**
```json
{
  "items": [
    {
      "id": "uuid",
      "submission_id": "uuid",
      "user_id": "uuid",
      "task_id": "uuid",
      "user_name": "John Doe",
      "user_avatar": "https://...",
      "task_title": "Complete Social Media Post",
      "task_xp": 100,
      "proof_url": "https://...",
      "reaction_count": 5,
      "comment_count": 3,
      "user_reacted": false,
      "created_at": "2026-01-26T12:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5,
  "feed_type": "pan-india"
}
```

**Features:**
- Only shows approved submissions with image/video proof
- Supports three feed types: pan-india, state, college
- Includes reaction and comment counts
- Shows if current user reacted

#### GET `/api/feed/user/{userId}`
Get feed items for a specific user (their completed tasks).

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 20, max: 100)

#### POST `/api/feed/{feedId}/react` (Protected)
React to a feed item.

**Request Body:**
```json
{
  "reaction": "like"
}
```

#### POST `/api/feed/{feedId}/comment` (Protected)
Comment on a feed item.

**Request Body:**
```json
{
  "comment": "Great work!"
}
```

**Response:**
```json
{
  "comment": {
    "id": "uuid",
    "feed_id": "uuid",
    "user_id": "uuid",
    "user_name": "John Doe",
    "user_avatar": "https://...",
    "comment": "Great work!",
    "created_at": "2026-01-26T12:00:00Z"
  },
  "message": "Comment added successfully"
}
```

---

### Leaderboard Endpoints

#### GET `/api/leaderboard/pan-india`
Get pan-India leaderboard.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 100, max: 1000)

**Response:**
```json
{
  "entries": [
    {
      "rank": 1,
      "user_id": "uuid",
      "user_name": "John Doe",
      "user_avatar": "https://...",
      "xp": 5000,
      "level": 10
    }
  ],
  "type": "pan-india",
  "page": 1,
  "page_size": 100
}
```

#### GET `/api/leaderboard/state`
Get state leaderboard.

**Query Parameters:**
- `state_id` (required): State UUID
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 100, max: 1000)

**Response:**
```json
{
  "entries": [...],
  "type": "state",
  "scope_id": "state-uuid",
  "page": 1,
  "page_size": 100
}
```

#### GET `/api/leaderboard/college`
Get college leaderboard.

**Query Parameters:**
- `college_id` (required): College UUID
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 100, max: 1000)

**Response:**
```json
{
  "entries": [...],
  "type": "college",
  "scope_id": "college-uuid",
  "page": 1,
  "page_size": 100
}
```

**Features:**
- Real-time updates via WebSocket
- Ranks users by XP (descending)
- Includes user avatar, name, XP, and level

---

### State and College Endpoints

#### GET `/api/states`
Get all states.

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "Maharashtra"
  }
]
```

#### GET `/api/states/{stateId}/colleges`
Get colleges for a specific state.

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "IIT Bombay",
    "state_id": "uuid",
    "city": "Mumbai"
  }
]
```

---

## Admin Endpoints

### Base URL
```
http://localhost:8080/admin
```

### State Management

#### GET `/admin/states`
Get all states.

#### POST `/admin/states`
Create a new state.

**Request Body:**
```json
{
  "name": "Maharashtra"
}
```

### College Management

#### POST `/admin/colleges`
Create a new college.

**Request Body:**
```json
{
  "name": "IIT Bombay",
  "state_id": "uuid",
  "city": "Mumbai"  // Optional
}
```

### Task Management

#### POST `/admin/tasks`
Create a new task.

**Request Body:**
```json
{
  "title": "Complete Social Media Post",
  "description": "Post about the platform",
  "xp": 100,
  "type": "social",
  "proof_type": "image",
  "priority": "normal",
  "start_at": "2026-01-26T00:00:00Z",  // Optional
  "end_at": "2026-01-30T23:59:59Z",    // Optional
  "is_flash": false,
  "is_weekly": false,
  "assignment_type": "all",  // "all", "state", "college", "user"
  "assignment_id": ""       // Required if assignment_type is not "all"
}
```

**Assignment Types:**
- `all`: Assign to all students
- `state`: Assign to students from a specific state
- `college`: Assign to students from a specific college
- `user`: Assign to a single user

**Response:**
```json
{
  "task": {
    "id": "uuid",
    "title": "Complete Social Media Post",
    "xp": 100,
    ...
  },
  "assigned_to": 150
}
```

#### PUT `/admin/tasks/{id}`
Update a task (not implemented yet).

### Submission Management

#### GET `/admin/submissions`
Get all submissions.

**Query Parameters:**
- `status` (optional): Filter by status (`pending`, `approved`, `rejected`)

**Response:**
```json
[
  {
    "id": "uuid",
    "task_id": "uuid",
    "user_id": "uuid",
    "proof_url": "https://...",
    "status": "pending",
    "admin_comment": null,
    "reviewed_by": null,
    "created_at": "2026-01-26T12:00:00Z"
  }
]
```

#### POST `/admin/submissions/{id}/approve`
Approve a submission.

**Request Body:**
```json
{
  "comment": "Great work!"  // Optional
}
```

**Response:**
```json
{
  "id": "uuid",
  "task_id": "uuid",
  "user_id": "uuid",
  "proof_url": "https://...",
  "status": "approved",
  "admin_comment": "Great work!",
  "reviewed_by": "admin-uuid",
  "updated_at": "2026-01-26T12:30:00Z"
}
```

**Features:**
- Awards XP to user (from task XP value)
- Creates feed entry for approved submission
- Broadcasts leaderboard updates via Redis
- Prevents duplicate XP awards

#### POST `/admin/submissions/{id}/reject`
Reject a submission.

**Request Body:**
```json
{
  "comment": "Proof does not meet requirements"  // Required
}
```

**Response:**
```json
{
  "id": "uuid",
  "status": "rejected",
  "admin_comment": "Proof does not meet requirements",
  "reviewed_by": "admin-uuid"
}
```

**Features:**
- User can resubmit if task deadline hasn't passed
- Comment is required for rejection

---

## WebSocket Endpoints

### Base URL
```
ws://localhost:8080/ws
```

### Leaderboard WebSocket

#### `/ws/leaderboard`
Real-time leaderboard updates.

**Query Parameters:**
- `type` (optional): `pan-india`, `state`, `college` (default: `pan-india`)
- `scope_id` (optional): State ID or College ID (required for state/college types)

**Connection Example:**
```javascript
// Pan-India leaderboard
const ws = new WebSocket('ws://localhost:8080/ws/leaderboard?type=pan-india');

// State leaderboard
const ws = new WebSocket('ws://localhost:8080/ws/leaderboard?type=state&scope_id=STATE_ID');

// College leaderboard
const ws = new WebSocket('ws://localhost:8080/ws/leaderboard?type=college&scope_id=COLLEGE_ID');
```

**Message Format:**

Initial data (on connection):
```json
{
  "type": "leaderboard_data",
  "scope": "pan-india",
  "scope_id": "",
  "entries": [
    {
      "rank": 1,
      "user_id": "uuid",
      "user_name": "John Doe",
      "user_avatar": "https://...",
      "xp": 5000,
      "level": 10
    }
  ]
}
```

Update notification:
```json
{
  "type": "leaderboard_update",
  "scope": "pan-india",
  "scope_id": "",
  "timestamp": 1706284800
}
```

**Features:**
- Sends initial leaderboard data on connection
- Broadcasts updates when XP is awarded
- Uses Redis pub/sub for scalable updates
- Auto-reconnects with ping/pong mechanism

### Chat WebSocket

#### `/ws/chat`
Chat WebSocket (not implemented yet).

### Notifications WebSocket

#### `/ws/notifications`
Real-time notifications (not implemented yet).

---

## GraphQL API

### Endpoint
```
http://localhost:8080/graphql
```

### Playground
```
http://localhost:8080/playground
```

### Schema
GraphQL schema is defined in `graph/schema.graphqls`.

**Note:** GraphQL implementation is in progress. Check the schema file for available queries and mutations.

### Supported Transports
- GET
- POST
- Multipart Form (for file uploads)
- WebSocket (for subscriptions)

---

## Database Schema

### Core Tables

#### `users`
- User accounts with authentication
- Fields: id, name, email, password_hash, phone, state_id, college_id, role, xp, level, coins, bio, avatar_url, resume_url, referral_code, referred_by_id

#### `states`
- Indian states
- Fields: id, name

#### `colleges`
- Colleges/universities
- Fields: id, name, state_id, city

#### `tasks`
- Tasks assigned to users
- Fields: id, title, description, xp, type, proof_type, priority, start_at, end_at, is_flash, is_weekly, created_by

#### `submissions`
- Task submissions by users
- Fields: id, task_id, user_id, proof_url, status (pending/approved/rejected), admin_comment, reviewed_by

#### `completed_task_feed`
- Feed entries for approved submissions
- Fields: id, submission_id, user_id, task_id, visibility, created_at

#### `task_feed_reactions`
- Reactions on feed items
- Fields: feed_id, user_id, reaction, created_at

#### `task_feed_comments`
- Comments on feed items
- Fields: id, feed_id, user_id, comment, created_at

#### `user_follows`
- User follow relationships
- Fields: follower_id, following_id, created_at

#### `xp_logs`
- XP award history
- Fields: id, user_id, source, source_id, xp, created_at

### Indexes
All tables have appropriate indexes for performance optimization.

---

## Environment Variables

Create a `.env` file in the root directory:

```bash
# Server
ENV=development
API_HOST=0.0.0.0
API_PORT=8080

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/gamified_ambassador?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_EXIRY=24h

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001

# AWS S3
AWS_REGION=us-east-1
AWS_PROFILE_BUCKET=your-profile-bucket
AWS_RESUME_BUCKET=your-resume-bucket
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_PROFILE_PUBLIC_URL=https://your-profile-bucket.s3.region.amazonaws.com
AWS_RESUME_PUBLIC_URL=https://your-resume-bucket.s3.region.amazonaws.com
```

---

## Key Features

### Authentication & Authorization
- JWT-based authentication
- Password hashing with bcrypt
- Role-based access control (admin/student)
- Protected routes with middleware

### User Management
- User registration with optional referral code
- Profile management (resume, profile picture)
- Follow/unfollow functionality
- User profile views with completed tasks

### Task System
- Task creation with flexible assignment (all, state, college, user)
- Task submission with image/video proof
- Admin approval/rejection workflow
- Resubmission support for rejected tasks

### XP & Gamification
- XP awards on task approval
- XP logging for audit trail
- Extensible XP sources (task_approval, referral, daily_login, etc.)
- Level system based on XP

### Feed System
- Three feed types: pan-india, state, college
- Only approved submissions with image/video proof
- Reactions and comments
- Pagination support

### Leaderboard
- Pan-India, state, and college leaderboards
- Real-time updates via WebSocket
- Pagination support
- Ranked by XP

### File Storage
- AWS S3 integration
- Separate buckets for profiles and resumes
- Public URL support
- Automatic cleanup on updates

### Real-time Updates
- WebSocket support for leaderboard
- Redis pub/sub for scalable updates
- Automatic broadcast on XP changes

---

## Development

### Running Tests
```bash
make test
```

### Code Formatting
```bash
make fmt
```

### Linting
```bash
make lint
```

### Creating Migrations
```bash
make migrate-create NAME=create_new_table
```

### Database Migrations
```bash
# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check migration status
migrate -path ./migrations -database $DATABASE_URL version
```

---

## Deployment

### Docker Deployment

1. Build the image:
   ```bash
   make docker-build
   ```

2. Run with docker-compose:
   ```bash
   make docker-up
   ```

### Production Considerations

1. **Environment Variables**: Use secure secrets management
2. **JWT Secret**: Use a strong, random secret
3. **Database**: Use connection pooling and read replicas
4. **Redis**: Configure persistence if needed
5. **S3**: Use IAM roles instead of access keys when possible
6. **CORS**: Restrict allowed origins in production
7. **GraphQL Playground**: Disable in production
8. **Logging**: Implement structured logging
9. **Monitoring**: Add health checks and metrics
10. **Rate Limiting**: Implement rate limiting for API endpoints

---

## API Examples

See `CURL_EXAMPLES.md` for detailed curl command examples for all endpoints.

---

## License

[Your License Here]

---

## Support

For issues and questions, please open an issue on the repository.
