# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Carryless is an outdoor gear catalog and pack planner built with Go, similar to Lighterpack. It allows users to track gear, plan packs, and analyze weight distribution for outdoor adventures.

## Development Commands

### Build and Run
```bash
# Install dependencies
go mod tidy

# Build the application
go build -o carryless .

# Run locally
./carryless
# Server will start on http://localhost:8080
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run specific test file
go test ./internal/database/
```

### Docker
```bash
# Build Docker image
docker build -t carryless .

# Run with Docker Compose (production setup with Traefik)
docker-compose -f docker-compose.traefik.yaml up -d
```

## Architecture Overview

### Clean Architecture Structure
The application follows Go's internal package convention with clean separation:

- `internal/config/` - Environment configuration management
- `internal/database/` - SQLite operations, migrations, and database layer
- `internal/handlers/` - HTTP request handlers organized by domain (auth, inventory, packs, categories)
- `internal/middleware/` - Custom middleware (auth, CSRF, rate limiting, security headers)
- `internal/models/` - Data structures and domain models

### Key Architectural Patterns

**Database Layer**: Direct SQL with SQLite using database/sql. All database operations are in dedicated files:
- `database.go` - Connection, migrations, core setup
- `auth.go` - User authentication and session management
- `items.go` - Inventory item operations
- `packs.go` - Pack and pack item operations
- `categories.go` - Category management
- `admin.go` - Administrative functions

**Security Middleware Stack**: Applied in handlers/handlers.go:SetupRoutes()
- Rate limiting with token bucket (golang.org/x/time/rate)
- CSRF protection with token validation
- Security headers (CSP, X-Frame-Options, etc.)
- Session-based authentication with bcrypt password hashing

**Template System**: Go's html/template with custom functions:
- `jsonify` - Marshal data for JavaScript consumption
- `groupByCategory` - Group items by category for display
- Template inheritance with `base.html`

### Data Flow

1. **Request Flow**: Gin router → Security middleware → Auth middleware → Handler → Database layer
2. **Authentication**: Session-based with SQLite storage, CSRF tokens for state-changing operations
3. **Pack Sharing**: UUID-based public pack URLs, toggle public/private per pack

### Database Schema

Core entities with foreign key relationships:
- `users` (id, username, email, password_hash, currency, is_admin)
- `categories` (user-scoped gear categories)
- `items` (gear items with weight, price, notes)
- `packs` (UUID-identified collections with public sharing)
- `pack_items` (join table with worn status and counts)
- `sessions` & `csrf_tokens` (security management)

### Frontend Integration

- Vanilla JavaScript in `static/js/app.js`
- Chart.js for weight distribution visualization
- Forms use CSRF tokens from hidden inputs
- JSON data passed via template `jsonify` function

### Configuration

Environment variables (see internal/config/config.go):
- `PORT` - Server port (default: 8080)
- `DATABASE_PATH` - SQLite file location (default: carryless.db)

## Testing Strategy

Single test file `internal/database/database_test.go` covers:
- User creation and authentication
- Session management
- CRUD operations for categories, items, and packs
- Uses in-memory SQLite (`:memory:`) for isolation