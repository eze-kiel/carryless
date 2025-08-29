# Carryless

Outdoor gear catalog and pack planner built with Go. Track gear weight, plan packs, and analyze weight distribution for backpacking and outdoor adventures.

## Quick Start

**Requirements:** Go 1.21+

```bash
git clone <repository-url>
cd carryless
go mod tidy
go build -o carryless .
./carryless
```

Open http://localhost:8080

## Configuration

Environment variables:
- `PORT` - Server port (default: 8080)
- `DATABASE_PATH` - SQLite database path (default: carryless.db)

```bash
export PORT=3000
./carryless
```

## Architecture

**Backend:** Go + Gin web framework  
**Database:** SQLite with foreign key constraints  
**Frontend:** Vanilla JavaScript, HTML5, Chart.js  

Clean architecture with organized internal packages:
- `internal/handlers/` - HTTP request handlers
- `internal/database/` - SQLite operations and migrations
- `internal/middleware/` - Auth, CSRF, rate limiting
- `internal/models/` - Data structures

## Core Features

- Multi-user gear inventory with categories
- Pack planning with weight analysis
- Public pack sharing via UUID URLs  
- Weight visualization with Chart.js
- Session-based authentication

## Development

```bash
# Run tests
go test ./...

# Build for production
go build -ldflags="-s -w" -o carryless .

# Docker
docker build -t carryless .
docker-compose -f docker-compose.traefik.yaml up -d
```

## Database Schema

- `users` - User accounts and preferences
- `categories` - User-defined gear categories
- `items` - Gear items with weight, price, notes
- `packs` - Trip pack collections
- `pack_items` - Items in packs with worn status

## API Structure

**Auth:** `/register`, `/login`, `/logout`  
**Categories:** `/categories` (CRUD)  
**Items:** `/inventory/items` (CRUD)  
**Packs:** `/packs` (CRUD), `/packs/:id/items` (manage pack contents)  
**Public:** `/p/packs/:id` (view shared packs)