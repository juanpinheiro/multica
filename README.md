# Multica

Personal AI agent management platform — assign work to local agents (Claude Code, Codex, Gemini), watch them run via the local daemon, track progress in the dashboard.

This is a personal fork of Multica. Removed:

- Desktop and mobile apps (web only)
- Magic-link / Google OAuth / PAT auth (loopback-trusted — no login screen on localhost)
- Multi-user features (invitations, workspace roles, email notifications)
- SaaS infrastructure (PostHog analytics, CloudFront, Redis, Cloud Runtime fleet, contact sales)
- Release pipeline (GoReleaser, Homebrew tap, install scripts, self-hosting docs)
- Chinese translations (English only)

## Running

```bash
make dev
```

Then visit `http://localhost:3000`. No login required — loopback requests are trusted automatically.

To expose over a network, set `MULTICA_TOKEN` in `.env` and pass `Authorization: Bearer <token>` from the remote client.

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Next.js    │────>│  Go Backend  │────>│  PostgreSQL  │
│   Frontend   │<────│  (Chi + WS)  │<────│              │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                     ┌──────┴───────┐
                     │ Agent Daemon │  runs on your machine
                     └──────────────┘  (Claude Code, Codex, Gemini, …)
```

| Layer | Stack |
|-------|-------|
| Frontend | Next.js (App Router, Turbopack) |
| Backend | Go (Chi router, sqlc, gorilla/websocket) |
| Database | PostgreSQL 17 |
| Agent Runtime | Local daemon |

## Everything else

See `CLAUDE.md` for architecture details, coding conventions, commands, and testing rules.
