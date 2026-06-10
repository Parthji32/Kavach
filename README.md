# Kavach - Armor That Fights Back

A full-stack cybersecurity honeypot platform built with Go + Fiber + HTMX. Deploy canary tokens, capture attacker fingerprints, and get real-time alerts when your traps are triggered.

## Features

- **Canary Tokens** - Deploy URL, Document, API Key, DNS, and Email tokens
- **Attacker Fingerprinting** - Capture browser, OS, TLS, and network fingerprints
- **Geo Enrichment** - IP geolocation via ipinfo.io with VPN/Tor detection
- **Attacker Correlation** - Link multiple events to the same threat actor
- **Real-time Alerts** - Slack, Email (Resend), and custom webhooks
- **Dark Theme Dashboard** - Built with Tailwind CSS + HTMX for live updates
- **Demo Mode** - Works without a database using realistic mock data

## Quick Start

```bash
# Clone and run (no database needed)
git clone https://github.com/parthjindal/kavach.git
cd kavach
go mod tidy
go run cmd/server/main.go

# Open http://localhost:8080
```

## Prerequisites

- Go 1.22+
- PostgreSQL 15+ (optional - demo mode works without it)

## Project Structure

```
kavach/
+-- cmd/server/main.go          # Application entry point
+-- internal/
|   +-- alerts/alerts.go        # Email + Slack alert dispatch
|   +-- database/               # PostgreSQL connection + repos
|   +-- fingerprint/            # Request fingerprinting engine
|   +-- handlers/               # HTTP route handlers
|   |   +-- auth_handler.go     # Login/signup API
|   |   +-- page_handler.go     # HTML page rendering
|   |   +-- trigger_handler.go  # Token trigger capture
|   +-- middleware/auth.go      # JWT authentication
|   +-- models/                 # Data models
|   +-- services/               # Business logic
|       +-- token_service.go    # Token generation
|       +-- geo_service.go      # IP geolocation
|       +-- attacker_service.go # Attacker correlation
+-- templates/                  # Go HTML templates
|   +-- layouts/base.html       # Shared layout with nav
|   +-- dashboard/index.html    # Dashboard page
|   +-- tokens/                 # Token pages
|   +-- alerts/index.html       # Alert history
|   +-- attackers/              # Attacker profiles
|   +-- auth/                   # Login/signup forms
|   +-- integrations/           # Integration settings
|   +-- settings/               # User settings
+-- static/js/app.js           # Client-side JavaScript
+-- migrations/                 # SQL schema
+-- Dockerfile                  # Container build
+-- Makefile                    # Dev commands
+-- go.mod                      # Go modules
```

## Routes

### HTML Pages

| Route | Description |
|-------|-------------|
| `GET /` | Dashboard with KPI cards, alerts, attack map |
| `GET /login` | Login form |
| `GET /signup` | Registration form |
| `GET /tokens` | All deployed tokens |
| `GET /tokens/new` | Create new token |
| `GET /alerts` | Full alert history with filtering |
| `GET /attackers` | Profiled threat actors |
| `GET /attackers/:id` | Attacker detail + timeline |
| `GET /integrations` | Slack, email, webhook config |
| `GET /settings` | User preferences |

### API Endpoints

| Route | Description |
|-------|-------------|
| `POST /api/v1/auth/signup` | Register new account |
| `POST /api/v1/auth/login` | Authenticate |
| `POST /api/v1/auth/logout` | Clear session |
| `GET /api/v1/auth/me` | Current user info |
| `GET /api/v1/tokens` | List tokens |
| `POST /api/v1/tokens` | Create token |
| `DELETE /api/v1/tokens/:id` | Delete token |
| `GET /api/v1/alerts` | List alerts |
| `GET /api/v1/attackers` | List attackers |
| `GET /api/v1/stats` | Dashboard statistics |
| `GET /health` | Health check |

### Token Triggers (Public)

| Route | Description |
|-------|-------------|
| `GET /t/:id` | URL token trigger |
| `GET /t/:id/doc` | Document token (returns tracking pixel) |
| `GET /t/:id/key` | API key token (returns 401) |
| `GET /t/:id/dns` | DNS token trigger |
| `GET /t/:id/email` | Email token trigger |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | Server port (default: 8080) |
| `DATABASE_URL` | No | PostgreSQL connection string |
| `JWT_SECRET` | No | JWT signing key (auto-generated in dev) |
| `IPINFO_TOKEN` | No | ipinfo.io API token for geo enrichment |
| `SLACK_WEBHOOK_URL` | No | Slack incoming webhook URL |
| `RESEND_API_KEY` | No | Resend.com API key for email alerts |
| `ALERT_FROM_EMAIL` | No | Email sender address |
| `ALERT_TO_EMAIL` | No | Alert recipient email |
| `BASE_URL` | No | Base URL for token trigger links |

## Development

```bash
# Run with hot reload (install air first: go install github.com/cosmtrek/air@latest)
make dev

# Build binary
make build

# Run migrations
make migrate

# Format code
make fmt

# See all routes
make routes
```

## Docker

```bash
# Build image
docker build -t kavach:latest .

# Run container
docker run -p 8080:8080 kavach:latest

# With environment file
docker run -p 8080:8080 --env-file .env kavach:latest
```

## How It Works

1. **Deploy tokens** - Create canary tokens (fake credentials, URLs, documents)
2. **Plant them** - Place tokens where attackers would find them (repos, shared drives, configs)
3. **Wait for triggers** - When an attacker uses a token, Kavach captures their fingerprint
4. **Get alerted** - Instant notifications via Slack, email, or webhooks
5. **Profile attackers** - Correlate multiple events to build threat actor profiles

## Tech Stack

- **Backend**: Go 1.22, Fiber v2
- **Frontend**: Go templates, Tailwind CSS (CDN), HTMX
- **Database**: PostgreSQL (optional)
- **Auth**: JWT with HTTP-only cookies
- **Alerts**: Resend (email), Slack webhooks
- **Geo**: ipinfo.io API

## License

MIT
