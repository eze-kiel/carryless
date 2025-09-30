# Carryless

A comprehensive outdoor gear catalog and pack planner for backpackers and outdoor enthusiasts. Track your gear inventory, plan perfect packs, analyze weight distribution, and share your setups with the community.

## Features

### Gear Management
- **Complete inventory** - Catalog gear with weight, price, notes, and categories
- **Weight verification** - Mark items that need weight confirmation
- **Multi-currency support** - Display prices in your preferred currency (USD, EUR, etc.)
- **Smart categories** - Organize by shelter, cooking, clothing, electronics, etc.
- **Weight units** - Switch between grams and ounces globally

### Pack Planning
- **Advanced pack creation** - Design packs for specific trips with detailed notes
- **Container organization** - Use labels to organize items into pack compartments/containers
- **Worn vs carried** - Track what you wear vs carry for accurate pack weight
- **Item quantities** - Specify counts and partial quantities (e.g., 2.5 items)
- **Weight analysis** - Total weight, category breakdowns, visual charts
- **Print-ready checklists** - Generate PDFs with checkboxes for packing

### Sharing & Community
- **Public pack sharing** - Share packs via clean URLs (e.g., `/p/abc123`)
- **Short link system** - Generate memorable short IDs for public packs
- **Privacy controls** - Toggle packs between private and public instantly

### User Management
- **Email-based activation** - Secure account activation workflow
- **Smart session management** - Automatic session extension for active users
- **Configurable session durations** - Customize how long users stay logged in
- **Admin panel** - User management, system controls, registration settings
- **Multi-user support** - Each user has isolated data and preferences

## Screenshots

<details>
<summary>View Screenshots</summary>

### Homepage
![Homepage](img/home.png)

### Inventory Management
![Inventory Management](img/inventory.png)

### Pack Statistics
![Pack Statistics](img/pack-stats.png)

### Pack Content
![Pack Content](img/pack-content.png)

</details>

## Quick Start

**Requirements:** Go 1.21+

```bash
# Clone and build
git clone <repository-url>
cd carryless
go mod tidy
go build -o carryless .

# Run locally
./carryless
```

Visit http://localhost:8080 to get started.

## Configuration

### Basic Configuration
```bash
export PORT=3000                    # Server port (default: 8080)
export DATABASE_PATH=/data/app.db   # SQLite database path (default: carryless.db)
export ALLOWED_ORIGINS="https://yourdomain.com,https://app.yourdomain.com"
```

### Session Configuration
Control user session behavior with sliding window sessions:

```bash
export SESSION_DURATION=336        # Session duration in hours (default: 336 = 14 days)
```

You can also use Go duration strings:
```bash
export SESSION_DURATION="30d"      # 30 days
```

**Session Behavior:**
- Sessions use a sliding window approach - each request automatically extends the session by the full duration
- Active users stay logged in as long as they're active
- Default: 14-day sessions that reset to 14 days on each request

### Email Configuration (Mailgun)
For user activation emails and admin notifications:

```bash
# Required for email functionality
export MAILGUN_DOMAIN="yourdomain.com"
export MAILGUN_API_KEY="your-mailgun-api-key"

# Optional email settings
export MAILGUN_SENDER_EMAIL="noreply@yourdomain.com"  # Default: noreply@carryless.org
export MAILGUN_SENDER_NAME="Your App Name"            # Default: Carryless
export MAILGUN_REGION="US"                            # Default: EU (EU/US)
```

**Note:** Without Mailgun configuration, users will be automatically activated and no emails will be sent.

### System Features
- **Registration control** - Admins can enable/disable new user registration
- **Admin notifications** - Get notified when new users register  
- **Activation workflow** - Email-based account activation (if email is configured)
- **Advanced rate limiting** - Different limits for authentication, activation, and general requests
- **IP blocking** - Automatic blocking of IPs with excessive 404 errors
- **Security headers** - CSP, HSTS, X-Frame-Options, etc.

## Development

### Testing
```bash
go test ./...                 # Run all tests
go test ./... -v              # Verbose test output
go test ./internal/database/  # Test specific package
```

### Production Build
```bash
go build -ldflags="-s -w" -o carryless .
```

### Docker
```bash
# Build image
docker build -t carryless .

# Run with Docker Compose (includes Traefik labels)
docker-compose -f docker-compose.dev.yaml up -d
```

### Database
- **SQLite** with foreign key constraints
- **Automatic migrations** - Schema updates handled automatically
- **Indexes** - Optimized for performance with proper indexing
- **Data isolation** - Complete separation between users

## Security Features

- **Rate limiting** - Configurable per IP (20 req/sec general, 5/min auth, 3 per 5min activation)
- **CSRF protection** - Token-based protection for state-changing operations  
- **Smart session security** - HTTP-only, secure, SameSite cookies with automatic extension
- **Configurable session lifetimes** - Customizable session durations and extension policies
- **IP blocking** - Auto-block IPs with 10+ 404s in 5 minutes (15min timeout)
- **Security headers** - Comprehensive CSP, HSTS, XSS protection
- **Input validation** - Automatic trimming, SQL injection prevention

## API Endpoints

### Authentication
- `POST /register` - User registration
- `POST /login` - User login  
- `POST /logout` - User logout
- `GET /activate/:token` - Account activation

### Inventory Management
- `GET /categories` - List categories
- `POST /categories` - Create category
- `GET /inventory` - List items
- `POST /inventory/items` - Create item
- `PUT /inventory/items/:id` - Update item

### Pack Management  
- `GET /packs` - List user's packs
- `POST /packs` - Create pack
- `GET /packs/:id` - View pack details
- `POST /packs/:id/items` - Add item to pack
- `GET /p/packs/:shortid` - Public pack view

### Admin
- `GET /admin` - Admin dashboard
- `POST /admin/settings` - Update system settings

## Technical Architecture

- **Backend:** Go 1.21+ with Gin web framework
- **Database:** SQLite with comprehensive migrations
- **Frontend:** Vanilla JavaScript, Chart.js for visualizations  
- **Security:** Multi-layer middleware stack
- **Email:** Mailgun integration for transactional emails
- **Deployment:** Docker with Traefik reverse proxy support

## License

This project is open source and available under the [MIT License](LICENSE).
