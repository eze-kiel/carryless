# Carryless

A lightweight outdoor gear catalog and pack planner built with Go, similar to Lighterpack. Track your gear, plan your packs, and analyze weight distribution for your outdoor adventures.

## Features

- **User Authentication**: Secure user registration and login with bcrypt password hashing
- **Gear Inventory**: Catalog your outdoor gear with details like weight, category, price, and notes
- **Category Management**: Organize gear into custom categories (Sleeping, Cooking, Clothing, etc.)
- **Pack Planning**: Create and manage packs for different trips
- **Weight Analysis**: Track total weight, worn weight, and visualize weight distribution by category
- **Pack Sharing**: Make packs public and share them via UUID-based URLs
- **Chart Visualization**: Interactive charts using Chart.js to visualize weight distribution
- **Security**: CSRF protection, rate limiting, and secure session management

## Technology Stack

- **Backend**: Go with Gin web framework
- **Database**: SQLite with foreign key constraints
- **Frontend**: Vanilla JavaScript, HTML5, and custom CSS
- **Visualization**: Chart.js for weight distribution charts
- **Security**: bcrypt for password hashing, CSRF tokens, rate limiting

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd carryless
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the application:
   ```bash
   go build -o carryless .
   ```

4. Run the application:
   ```bash
   ./carryless
   ```

5. Open your browser and navigate to `http://localhost:8080`

### Configuration

The application can be configured using environment variables:

- `PORT`: Server port (default: 8080)
- `DATABASE_PATH`: SQLite database file path (default: carryless.db)
- `SECRET_KEY`: Secret key for sessions (change in production)

Example:
```bash
export PORT=3000
export DATABASE_PATH=/path/to/database.db
export SECRET_KEY=your-secret-key
./carryless
```

## Usage

### Getting Started

1. **Register an Account**: Create a new user account
2. **Create Categories**: Set up categories for your gear (e.g., "Sleeping", "Cooking", "Clothing")
3. **Add Items**: Add your gear items with weights, categories, and notes
4. **Create Packs**: Build packs for specific trips by adding items
5. **Analyze Weight**: Use the weight visualization to see category distribution
6. **Share Packs**: Make packs public to share with others

### Database Schema

The application uses SQLite with the following main tables:

- `users`: User accounts with encrypted passwords
- `categories`: User-defined gear categories
- `items`: Individual gear items with weight and details
- `packs`: Pack collections with UUID identifiers
- `pack_items`: Items assigned to packs with worn status
- `sessions`: User session management
- `csrf_tokens`: CSRF protection tokens

## API Endpoints

### Authentication
- `GET /register` - Registration page
- `POST /register` - Create new user
- `GET /login` - Login page
- `POST /login` - Authenticate user
- `POST /logout` - Logout user

### Categories
- `GET /categories` - List user categories
- `POST /categories` - Create category
- `PUT /categories/:id` - Update category
- `DELETE /categories/:id` - Delete category

### Inventory
- `GET /inventory` - List user items
- `POST /inventory/items` - Create item
- `PUT /inventory/items/:id` - Update item
- `DELETE /inventory/items/:id` - Delete item

### Packs
- `GET /packs` - List user packs
- `POST /packs` - Create pack
- `GET /packs/:id` - View pack details
- `PUT /packs/:id` - Update pack
- `DELETE /packs/:id` - Delete pack
- `POST /packs/:id/items` - Add item to pack
- `DELETE /packs/:id/items/:item_id` - Remove item from pack
- `PUT /packs/:id/items/:item_id/worn` - Toggle worn status

### Public
- `GET /p/packs/:id` - View public pack

## Testing

Run the test suite:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test ./... -v
```

## Security Features

- **Password Security**: bcrypt hashing with salt
- **Session Management**: Secure session tokens with expiration
- **CSRF Protection**: Cross-site request forgery protection
- **Rate Limiting**: IP-based rate limiting to prevent abuse
- **SQL Injection Prevention**: Parameterized queries
- **XSS Prevention**: Proper input sanitization and output encoding
- **Security Headers**: CSP, X-Frame-Options, etc.

## Performance

- **Lightweight**: Minimal dependencies and optimized for speed
- **Efficient Database**: Indexed queries and foreign key constraints
- **Fast Frontend**: Vanilla JavaScript without heavy frameworks
- **Caching**: Static asset caching and session management

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is open source and available under the MIT License.

## Acknowledgments

- Inspired by Lighterpack for the outdoor gear tracking concept
- Built with security and performance as primary concerns
- Designed for ultralight backpacking and outdoor enthusiasts