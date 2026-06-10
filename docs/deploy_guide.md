# Kavach Deployment Guide

> **Mode: PRIVATE DEPLOYMENT** This guide deploys Kavach with access restricted by a secret key. Only people with the key can access the dashboard. Trigger URLs (`/t/*`) remain public (required for honeypot functionality).

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Option 1: Railway (Recommended)](#option-1-railway-recommended--easiest)
3. [Option 2: Fly.io](#option-2-flyio-alternative)
4. [Option 3: Render.com](#option-3-rendercom-free-tier-available)
5. [Supabase Setup (Database)](#supabase-setup-database)
6. [Environment Variables Reference](#environment-variables-reference)
7. [Domain Setup](#domain-setup)
8. [Private Mode](#private-mode)
9. [Post-Deploy Checklist](#post-deploy-checklist)
10. [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before deploying, you need:

- [ ] A GitHub account (push your code to a private repo)
- [ ] A Supabase account (free — for the database)
- [ ] A Resend account (free — for alert emails)
- [ ] ~10 minutes

Push your code to GitHub first:

```bash
cd E:\kavach
git init
git add .
git commit -m "Initial commit"
git remote add origin https://github.com/YOUR_USERNAME/kavach.git
git push -u origin main

```

---

## Option 1: Railway (Recommended — Easiest)

Railway is the easiest way to deploy. It auto-detects your Dockerfile, builds, and deploys in one click.

**Cost:** Free tier gives you $5/month credit (enough for a low-traffic app like Kavach).

### Step 1: Sign Up

1. Go to [railway.app](https://railway.app)
2. Sign up with your GitHub account
3. Verify your account (may need a credit card for the free tier)

### Step 2: Create a New Project

1. Click **"New Project"** on the dashboard
2. Select **"Deploy from GitHub Repo"**
3. Authorize Railway to access your repo
4. Select your `kavach` repository

### Step 3: Set Environment Variables

Before the first deploy completes, add your env vars:

1. Click on your service (the purple box)
2. Go to the **"Variables"** tab
3. Click **"Raw Editor"** (easier to paste all at once)
4. Paste the following (fill in your values):

```env
ENV=production
PORT=8080
ALLOWED_ORIGINS=https://your-app-name.up.railway.app

# Database (from Supabase — see database section below)
DATABASE_URL=postgresql://postgres.xxxx:password@aws-0-us-east-1.pooler.supabase.com:6543/postgres

# Auth
JWT_SECRET=generate-a-random-64-char-string-here

# Private Access (THIS IS WHAT KEEPS IT PRIVATE)
KAVACH_ACCESS_KEY=your-secret-key-here-make-it-long-and-random

# Email Alerts (Resend — free tier: 100 emails/day)
RESEND_API_KEY=re_xxxxxxxxxxxxxxxxxxxx
ALERT_FROM_EMAIL=alerts@yourdomain.com
ALERT_TO_EMAIL=parth@youremail.com

# Slack Alerts (optional)
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T00/B00/xxxxx

# IP Geolocation (optional — free tier: 50k req/month)
IPINFO_TOKEN=your_ipinfo_token

# Trigger Base URL (the public URL for trigger links)
TRIGGER_BASE_URL=https://your-app-name.up.railway.app

# Redis (optional — for rate limiting / caching)
REDIS_URL=redis://default:password@your-redis-host:6379

```

### Step 4: Deploy

Railway auto-deploys when you push to `main`. It will:

1. Detect the `Dockerfile` in your repo root
2. Build the multi-stage Docker image
3. Start the container on port 8080
4. Assign a public URL

You can watch the build logs in the **"Deployments"** tab.

### Step 5: Get Your URL

1. Go to **Settings** → **Networking** → **Generate Domain**
2. Railway gives you a URL like: `kavach-production.up.railway.app`
3. Update your `ALLOWED_ORIGINS` and `TRIGGER_BASE_URL` env vars with this URL

### Step 6: Set Custom Domain (Optional)

1. In **Settings** → **Networking** → **Custom Domain**
2. Enter your domain: `kavach.dev` or `app.kavach.dev`
3. Railway shows you DNS records to add
4. Add the CNAME record at your domain registrar
5. Wait for SSL certificate (automatic, takes ~2 minutes)

### Step 7: Verify It Works

```bash
# Health check
curl https://your-app-name.up.railway.app/health

# Access dashboard (note the ?key= parameter)
open "https://your-app-name.up.railway.app?key=your-secret-key-here"

```

---

## Option 2: Fly.io (Alternative)

Fly.io gives you more control and runs containers on edge servers worldwide.

**Cost:** Free tier includes 3 shared-cpu VMs. More than enough.

### Step 1: Install the CLI

```bash
# macOS
brew install flyctl

# Windows (PowerShell)
powershell -Command "iwr https://fly.io/install.ps1 -useb | iex"

# Linux
curl -L https://fly.io/install.sh | sh

```

### Step 2: Sign Up & Login

```bash
fly auth signup
# or if you already have an account:
fly auth login

```

### Step 3: Launch the App

From your project root:

```bash
cd E:\kavach
fly launch

```

When prompted:

- **App name:** `kavach` (or `kavach-prod`)
- **Region:** Pick the closest to you (e.g., `iad` for US East)
- **Database:** Say **No** (we're using Supabase)
- **Redis:** Say **No** (unless you want Fly's managed Redis)
- **Deploy now:** Say **No** (we need to set secrets first)

This creates a `fly.toml` file. Verify it looks like this:

```toml
app = "kavach"
primary_region = "iad"

[build]
  dockerfile = "Dockerfile"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0

[env]
  ENV = "production"
  PORT = "8080"

```

### Step 4: Set Secrets (Environment Variables)

```bash
fly secrets set \
  DATABASE_URL="postgresql://postgres.xxxx:password@aws-0-us-east-1.pooler.supabase.com:6543/postgres" \
  JWT_SECRET="generate-a-random-64-char-string-here" \
  KAVACH_ACCESS_KEY="your-secret-key-here" \
  RESEND_API_KEY="re_xxxxxxxxxxxxxxxxxxxx" \
  ALERT_FROM_EMAIL="alerts@yourdomain.com" \
  ALERT_TO_EMAIL="parth@youremail.com" \
  ALLOWED_ORIGINS="https://kavach.fly.dev" \
  TRIGGER_BASE_URL="https://kavach.fly.dev" \
  IPINFO_TOKEN="your_ipinfo_token" \
  SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T00/B00/xxxxx"

```

### Step 5: Deploy

```bash
fly deploy

```

Watch the build output. Once done:

```bash
# Check status
fly status

# View logs
fly logs

# Open in browser
fly open

```

### Step 6: Get Your URL

Your app is live at: `https://kavach.fly.dev`

Access it with: `https://kavach.fly.dev?key=your-secret-key-here`

### Custom Domain on Fly.io

```bash
# Add a custom domain
fly certs add kavach.dev

# It shows DNS records — add them at your registrar
fly certs show kavach.dev

```

---

## Option 3: Render.com (Free Tier Available)

Render is simple and has a generous free tier, but free instances spin down after inactivity (cold starts of ~30 seconds).

**Cost:** Free tier available. Paid starts at $7/month (always-on).

### Step 1: Sign Up & Connect Repo

1. Go to [render.com](https://render.com)
2. Sign up with GitHub
3. Click **"New" → "Web Service"**
4. Connect your `kavach` repository

### Step 2: Configure the Service

- **Name:** `kavach`
- **Region:** Oregon (US West) or closest to you
- **Branch:** `main`
- **Runtime:** **Docker** (it will detect the Dockerfile)
- **Instance type:** Free (or Starter $7/mo for always-on)

### Step 3: Set Environment Variables

In the **"Environment"** section, add each variable:

| Key | Value |
| --- | --- |
| `ENV` | `production` |
| `PORT` | `8080` |
| `DATABASE_URL` | `postgresql://...` (from Supabase) |
| `JWT_SECRET` | `your-random-64-char-string` |
| `KAVACH_ACCESS_KEY` | `your-secret-key` |
| `RESEND_API_KEY` | `re_xxxx` |
| `ALERT_FROM_EMAIL` | `alerts@yourdomain.com` |
| `ALERT_TO_EMAIL` | `parth@youremail.com` |
| `ALLOWED_ORIGINS` | `https://kavach.onrender.com` |
| `TRIGGER_BASE_URL` | `https://kavach.onrender.com` |
| `IPINFO_TOKEN` | `your_token` |
| `SLACK_WEBHOOK_URL` | `https://hooks.slack.com/...` |

### Step 4: Deploy

Click **"Create Web Service"**. Render will:

1. Clone your repo
2. Build the Docker image
3. Deploy it
4. Give you a URL: `https://kavach.onrender.com`

### ⚠️ Note on Free Tier

Free instances on Render spin down after 15 minutes of inactivity. The first request after sleep takes ~30 seconds. This is fine for Kavach triggers (they still fire — Render wakes up on request), but the dashboard will be slow to load initially.

**Recommendation:** If you're using this in production, upgrade to the $7/month Starter plan.

---

## Supabase Setup (Database)

Supabase provides a free PostgreSQL database — perfect for Kavach.

### Step 1: Create a Project

1. Go to [supabase.com](https://supabase.com) and sign up (free)
2. Click **"New Project"**
3. Fill in:- **Project name:** `kavach`
- **Database password:** Generate a strong one (SAVE THIS!)
- **Region:** Choose the closest to your deploy platform
4. Click **"Create new project"**
5. Wait ~2 minutes for provisioning

### Step 2: Get Your DATABASE_URL

1. Go to **Project Settings** (gear icon) → **Database**
2. Scroll to **"Connection string"** section
3. Select **"URI"** tab
4. Copy the connection string — it looks like:

```
postgresql://postgres.abcdefghijk:[YOUR-PASSWORD]@aws-0-us-east-1.pooler.supabase.com:6543/postgres

```

1. Replace `[YOUR-PASSWORD]` with the password you set in Step 1

> **Important:** Use the **"Connection pooling"** URI (port `6543`), NOT the direct connection (port `5432`). Railway/Fly/Render don't support persistent connections well without pooling.

### Step 3: Run Migrations

You have two options:

**Option A: Using the Supabase SQL Editor (Easiest)**

1. Go to your Supabase project → **SQL Editor**
2. Click **"New Query"**
3. Copy-paste the contents of `migrations/001_initial_schema.sql`
4. Click **"Run"**
5. You should see: "Success. No rows returned"

**Option B: Using psql from your terminal**

```bash
# Install psql if you don't have it
# macOS: brew install libpq
# Windows: comes with PostgreSQL installer

# Run migrations
psql "postgresql://postgres.xxxx:password@aws-0-us-east-1.pooler.supabase.com:6543/postgres" \
  -f migrations/001_initial_schema.sql

```

### Step 4: Verify Connection

Check that tables were created:

1. Go to Supabase → **Table Editor**
2. You should see these tables:- `users`
- `tokens`
- `attackers`
- `trigger_events`
- `alert_configs`

If you see all 5, you're good! 🎉

---

## Environment Variables Reference

Here's every environment variable Kavach uses:

### Core (Required)

| Variable | What It Does | Example | Where to Get It |
| --- | --- | --- | --- |
| `PORT` | Port the server listens on | `8080` | Just use `8080` — Railway/Fly/Render expect this |
| `ENV` | Runtime mode (`development` or `production`) | `production` | Set to `production` for deploys |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://postgres.xx:pass@host:6543/postgres` | Supabase → Project Settings → Database → Connection String |
| `JWT_SECRET` | Secret key for signing auth tokens (min 32 chars) | `k8s9d7f6g5h4j3k2l1...` | Generate: `openssl rand -hex 32` |

### Private Access (Required for Private Mode)

| Variable | What It Does | Example | Where to Get It |
| --- | --- | --- | --- |
| `KAVACH_ACCESS_KEY` | Secret key to access the dashboard | `my-super-secret-beta-key-2024` | Make one up! Any string works. Longer = more secure |

### Networking

| Variable | What It Does | Example | Where to Get It |
| --- | --- | --- | --- |
| `ALLOWED_ORIGINS` | CORS allowed origins (comma-separated) | `https://kavach-prod.up.railway.app` | Your deployed app URL |
| `TRIGGER_BASE_URL` | Base URL for generated trigger links | `https://kavach-prod.up.railway.app` | Your deployed app URL (or custom domain) |

### Email Alerts (Recommended)

| Variable | What It Does | Example | Where to Get It |
| --- | --- | --- | --- |
| `RESEND_API_KEY` | API key for sending alert emails | `re_123abc456def` | [resend.com](https://resend.com) → API Keys → Create |
| `ALERT_FROM_EMAIL` | "From" address on alert emails | `alerts@kavach.dev` | Must be a verified domain on Resend, or use `onboarding@resend.dev` for testing |
| `ALERT_TO_EMAIL` | Where alert emails are sent | `parth@gmail.com` | Your personal email |

> **Resend Setup (2 minutes):**
> 1. Sign up at [resend.com](https://resend.com) (free: 100 emails/day)
> 2. Go to **API Keys** → **Create API Key**
> 3. Copy the key (starts with `re_`)
> 4. For testing, use `onboarding@resend.dev` as FROM email
> 5. For production, add your domain under **Domains** and verify it

### Slack Alerts (Optional)

| Variable | What It Does | Example | Where to Get It |
| --- | --- | --- | --- |
| `SLACK_WEBHOOK_URL` | Incoming webhook for Slack notifications | `https://hooks.slack.com/services/T00/B00/xxx` | Slack → Apps → Incoming Webhooks → Add |

> **Slack Webhook Setup:**
> 1. Go to [api.slack.com/apps](https://api.slack.com/apps) → Create New App
> 2. Choose "From scratch", name it "Kavach Alerts"
> 3. Go to **Incoming Webhooks** → Toggle ON
> 4. Click **"Add New Webhook to Workspace"**
> 5. Select a channel (e.g., `#security-alerts`)
> 6. Copy the webhook URL

### IP Intelligence (Optional)

| Variable | What It Does | Example | Where to Get It |
| --- | --- | --- | --- |
| `IPINFO_TOKEN` | Token for IP geolocation lookups | `abc123def456` | [ipinfo.io](https://ipinfo.io) → Sign up (free: 50k req/month) → Token |

### Redis (Optional)

| Variable | What It Does | Example | Where to Get It |
| --- | --- | --- | --- |
| `REDIS_URL` | Redis connection for rate limiting/caching | `redis://default:pass@host:6379` | Upstash (free), Railway Redis addon, or Fly Redis |

> **Easy Redis Options:**
> - **Upstash** (free tier: 10k commands/day): [upstash.com](https://upstash.com)
> - **Railway:** Add a Redis service in your project → copy the URL
> - Skip it for now — Kavach works without Redis (just no rate limiting)

---

## Domain Setup

A custom domain makes your trigger URLs look more legitimate (important for honeypots!).

### Step 1: Buy a Domain

Good options for Kavach:

- `kavach.dev` (~$12/year on Google Domains / Cloudflare)
- `getkavach.com` (~$10/year)
- Something innocent for triggers: `cdn-assets.dev`, `static-content.app`

**Where to buy:** [Cloudflare Registrar](https://dash.cloudflare.com) (cheapest, no markup) or [Namecheap](https://namecheap.com)

### Step 2: Point DNS to Your Platform

**For Railway:**

```
Type: CNAME
Name: @ (or subdomain like "app")
Value: kavach-production.up.railway.app

```

**For Fly.io:**

```
Type: CNAME
Name: @ (or subdomain)
Value: kavach.fly.dev

```

**For Render:**

```
Type: CNAME
Name: @ (or subdomain)
Value: kavach.onrender.com

```

### Step 3: Update Environment Variables

After your domain is set up, update these env vars:

```env
ALLOWED_ORIGINS=https://kavach.dev
TRIGGER_BASE_URL=https://kavach.dev

```

### Step 4: Consider a Separate Trigger Domain

For maximum effectiveness, your trigger URLs should look innocent. Consider:

- Main app: `app.kavach.dev` (your dashboard)
- Trigger URLs: `cdn-static.dev` or `api-analytics.io` (looks like a CDN/analytics)

This way, even if an attacker sees the URL, they won't immediately think "canary token."

---

## Private Mode

### How KAVACH_ACCESS_KEY Works

When `KAVACH_ACCESS_KEY` is set, Kavach restricts access:

| Route | Behavior |
| --- | --- |
| `/?key=YOUR_KEY` | ✅ Dashboard loads normally |
| `/` (no key) | ❌ Returns 401 Unauthorized |
| `/login?key=YOUR_KEY` | ✅ Login page works |
| `/login` (no key) | ❌ Blocked |
| `/api/v1/*` | ❌ Blocked without key |
| `/t/:id` (trigger URLs) | ✅ **Always public** — honeypots must work! |
| `/health` | ✅ Always public — for health checks |

### Setting It Up

1. **Generate a secret key:**

```bash
# Use any random string — the longer the better
openssl rand -hex 24
# Example output: a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9

```

1. **Set the env var** on your platform:

```bash
# Railway: Set in Variables tab
KAVACH_ACCESS_KEY=a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9

# Fly.io:
fly secrets set KAVACH_ACCESS_KEY=a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9

```

1. **Access your app with the key:**

```
https://kavach-production.up.railway.app?key=a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9

```

1. **Share with beta testers** by sending them the full URL with `?key=...`

### Important Notes

- The `?key=` parameter is passed on every page navigation (the app preserves it)
- API calls from the dashboard include the key automatically via session/cookie
- Trigger URLs (`/t/*`) **never** require the key — they must be publicly accessible or the honeypot doesn't work
- Bookmark the URL with the key so you don't have to type it each time

### Going Public Later

When you're ready to launch publicly, just remove the env var:

```bash
# Railway: Delete KAVACH_ACCESS_KEY from the Variables tab

# Fly.io:
fly secrets unset KAVACH_ACCESS_KEY

# Render: Remove from Environment settings

```

That's it. No code changes needed. The app instantly becomes public.

---

## Post-Deploy Checklist

Run through this after deploying:

### Health & Connectivity

- [ ] **Health check passes:**
  ```bash
  curl https://YOUR_URL/health
  # Expected: 200 OK with {"status": "ok"}
  
  ```
- [ ] **Dashboard loads:**
  ```bash
  open "https://YOUR_URL?key=YOUR_SECRET_KEY"
  # Expected: Login/signup page renders correctly
  
  ```

### Authentication

- [ ] **Can sign up:**
  - Go to `/signup?key=YOUR_KEY`
  - Create an account with email/password
  - Should redirect to dashboard
- [ ] **Can log in:**
  - Go to `/login?key=YOUR_KEY`
  - Log in with the account you created

### Token Creation

- [ ] **Can create a canary token:**
  - Go to `/tokens/new?key=YOUR_KEY`
  - Create a URL-type token
  - Should get a trigger URL like `https://YOUR_URL/t/abc123`

### Trigger Testing

- [ ] **Trigger URL works:**
  ```bash
  # Test the trigger URL from the token you created
  curl -v https://YOUR_URL/t/YOUR_TOKEN_TRIGGER_ID
  # Expected: 200 (with tracking pixel) or 404 (depending on token type)
  
  ```
- [ ] **Trigger event recorded:**
  - Check the dashboard — you should see the trigger event logged
  - Check the attacker profile (your IP will show up)

### Alerts

- [ ] **Email alert received (if Resend configured):**
  - After triggering a token, check your `ALERT_TO_EMAIL` inbox
  - Should receive an alert within seconds
- [ ] **Slack notification received (if webhook configured):**
  - After triggering a token, check your Slack channel
  - Should see a formatted alert message

### Security

- [ ] **Private mode working:**
  ```bash
  # Without key — should be blocked
  curl https://YOUR_URL/
  # Expected: 401 Unauthorized
  
  # With wrong key — should be blocked
  curl "https://YOUR_URL/?key=wrong-key"
  # Expected: 401 Unauthorized
  
  # Trigger URLs work without key (correct behavior!)
  curl https://YOUR_URL/t/any-valid-trigger-id
  # Expected: 200 (not 401)
  
  ```

---

## Troubleshooting

### Build Fails

```
Error: failed to build: exit code 1

```

- Check that `go.mod` and `go.sum` are committed
- Ensure `cmd/server/main.go` exists
- Check build logs for the specific Go error

### Database Connection Fails

```
Error: pq: password authentication failed

```

- Double-check your `DATABASE_URL` — password correct?
- Use the **pooler** connection string (port `6543`), not direct (port `5432`)
- Ensure you ran migrations (tables exist)

### 502 Bad Gateway

- App is crashing on startup — check logs:```bash
# Railway: Deployments tab → click latest → View Logs
# Fly.io: fly logs
# Render: Logs tab

```
- Usually means a missing env var or bad DATABASE_URL

### Trigger URLs Return 500

- Database might not have the `tokens` table — run migrations
- Check that `TRIGGER_BASE_URL` is set correctly

### Emails Not Sending

- Verify `RESEND_API_KEY` is correct (starts with `re_`)
- If using a custom FROM domain, verify it in Resend dashboard
- Check Resend's logs at [resend.com/emails](https://resend.com/emails)
- For testing, use `onboarding@resend.dev` as FROM (no domain setup needed)

### CORS Errors in Browser Console

- Update `ALLOWED_ORIGINS` to match your exact deployed URL
- Include the protocol: `https://kavach-prod.up.railway.app` (not just the domain)
- If using a custom domain, add both the Railway URL and custom domain

---

## Quick Reference

### Generate Secrets

```bash
# JWT_SECRET (64 hex chars)
openssl rand -hex 32

# KAVACH_ACCESS_KEY (48 hex chars)
openssl rand -hex 24

# Or use Python if you don't have openssl:
python3 -c "import secrets; print(secrets.token_hex(32))"

```

### Minimum Viable Deploy (Just the Essentials)

If you want the absolute minimum to get running:

```env
ENV=production
PORT=8080
DATABASE_URL=your_supabase_url
JWT_SECRET=your_random_secret
KAVACH_ACCESS_KEY=your_access_key
ALLOWED_ORIGINS=https://your-app.up.railway.app
TRIGGER_BASE_URL=https://your-app.up.railway.app

```

That's 7 variables. Everything else is optional (but recommended).

### Redeploy After Changes

```bash
# Just push to main — all platforms auto-deploy
git add .
git commit -m "your changes"
git push origin main

```

---

## 🎉 You're Live!

Once everything checks out, your private Kavach instance is running. Share the `?key=` URL with trusted beta testers and start planting canary tokens.

When you're ready to go public:

1. Remove `KAVACH_ACCESS_KEY` from env vars
2. Set up a proper domain
3. Update `TRIGGER_BASE_URL` and `ALLOWED_ORIGINS`
4. Ship it! 🚀

