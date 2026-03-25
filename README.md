# SDS Referral Dashboard

A read-only dashboard that tracks employee referrals by pulling data from **Greenhouse** and monitoring a **Slack channel**. Built on Apps Platform with a Go backend and React frontend.

## Features

- **Referrals tab** — searchable, filterable table of all referrals with candidate name, role, referrer, stage, and source
- **Priority Roles tab** — grid of open positions from Greenhouse with referral counts and toggleable priority flags
- **Analytics tab** — summary stats, referrals by stage (funnel), by role (bar chart), and weekly trend
- **Greenhouse sync** — periodic pull of open jobs and referral applications (every 30 minutes)
- **Slack monitoring** — passively captures referral submissions posted to a configured channel

## Architecture

```
Go (Gin) backend
├── Greenhouse API client (jobs + referral applications)
├── Slack listener (channel monitor via slacklib)
├── PostgreSQL (cached data + priority flags)
└── REST API → React (Vite + TailwindCSS) frontend
```

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 18+
- PostgreSQL (local or Cloud SQL)
- Greenhouse API key
- Slack Bot Token

### Environment Variables

```bash
export GREENHOUSE_API_KEY=your-api-key
export SLACK_BOT_TOKEN=xoxb-your-token
export REFERRAL_CHANNEL_ID=C0AMGLG0S74  # Slack channel to monitor
export DB_USER=postgres
export DB_NAME=postgres
```

### Local Development

```bash
make deps    # Install Go + npm dependencies
make run     # Start backend (8080) + frontend (3000) in dev mode
```

Or run separately:

```bash
make backend   # Go backend on :8080
make frontend  # Vite dev server on :3000
```

### Production Build

```bash
make build   # Builds frontend + Go binary
```

### Deploy to Apps Platform

```bash
make deploy
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/referrals` | List referrals (query: `stage`, `role`, `job_id`) |
| GET | `/api/referrals/stats` | Summary statistics |
| GET | `/api/referrals/by-stage` | Referral counts grouped by stage |
| GET | `/api/referrals/by-role` | Referral counts grouped by role |
| GET | `/api/referrals/weekly` | Weekly submission trend (12 weeks) |
| GET | `/api/jobs` | Open positions (query: `status`, `priority`) |
| PATCH | `/api/jobs/:id/priority` | Toggle priority flag on a job |
| GET | `/api/sync/status` | Last sync timestamp and status |
| POST | `/api/sync/trigger` | Manually trigger a Greenhouse sync |
| GET | `/api/teams` | List configured teams |
| GET | `/api/recruiters` | List configured recruiters |

## Project Structure

```
sds-referral-dashboard/
├── main.go              # Server entry point
├── api.go               # REST API handlers
├── config.go            # Team/recruiter DRI mapping
├── db.go                # PostgreSQL connection + queries
├── greenhouse.go        # Greenhouse Harvest API client
├── sync.go              # Periodic Greenhouse sync logic
├── slack_monitor.go     # Slack channel listener
├── schema.sql           # Database schema (reference)
├── project.toml         # Apps Platform config
├── Makefile             # Build/run/deploy commands
├── go.mod / go.sum      # Go dependencies
└── frontend/
    ├── src/
    │   ├── App.tsx
    │   ├── main.tsx
    │   └── components/
    │       ├── Sidebar.tsx
    │       ├── ReferralDashboard.tsx
    │       ├── PriorityRoles.tsx
    │       └── Analytics.tsx
    ├── package.json
    ├── vite.config.js
    └── tailwind.config.js
```

## Configuration

### Team/Recruiter Mapping

Edit `config.go` to update the `TeamDRIMapping` with your teams and their DRI recruiters.

### Greenhouse Sync

The sync runs every 30 minutes by default. Adjust the interval in `main.go`:

```go
go StartPeriodicSync(ctx, 30) // minutes
```

You can also trigger a manual sync from the sidebar button or via `POST /api/sync/trigger`.
