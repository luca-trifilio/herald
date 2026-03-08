# Herald — Project Context for Claude Code

## What this project is
A terminal UI application (Go + Bubble Tea) that every morning fetches tech newsletters
from the user's Gmail, summarises them via the Anthropic Claude API, and presents a
prioritised daily digest. The user can discard uninteresting items; the app learns from
those signals to improve future digests.

## Tech stack
- **Language**: Go 1.22+ (installed via Homebrew at /opt/homebrew/bin/go)
- **TUI framework**: Bubble Tea (github.com/charmbracelet/bubbletea) + Lipgloss + Bubbles
- **Storage**: SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **AI**: Anthropic Claude API (claude-sonnet-4-20250514) — summarisation + topic extraction
- **Gmail access**: Direct Google Gmail API via OAuth 2.0 (standard `net/http`, no SDK)
- **Config file**: `~/.config/newsdigest/config.json`
- **DB file**: `~/.local/share/newsdigest/newsdigest.db`

## Repository layout
herald/
├── CLAUDE.md
├── README.md
├── go.mod
├── go.sum
├── main.go                    # entry point; --setup-gmail flag; rootModel wrapper
├── internal/
│   ├── config/
│   │   └── config.go          # Load/Save; env vars override file
│   ├── db/
│   │   ├── db.go              # Open(), full repo functions; SeedDefaultSources()
│   │   └── schema.sql         # embedded via go:embed
│   ├── claude/
│   │   ├── client.go          # HTTP wrapper; Complete(system, user)
│   │   ├── digest.go          # BuildDigest(); calls config.Save after fetch
│   │   └── learn.go           # ExtractTopics(); upserts preferences
│   ├── models/
│   │   └── models.go          # NewsItem, Digest, Preference, NewsletterSource, Config
│   └── tui/
│       ├── app.go             # root BubbleTea model; view-stack (push/pop)
│       ├── digest_view.go     # split-pane main view (list left, preview right)
│       ├── detail_view.go     # full-screen single item view
│       ├── settings_view.go   # sources + preferences CRUD
│       ├── help_overlay.go    # ? overlay + persistent bottom keybinding bar
│       └── styles.go          # all Lipgloss styles; dark palette
└── scripts/
    └── setup.sh

## Build & run
```bash
export PATH=$PATH:/opt/homebrew/bin
go build -o herald .
./herald
```

## Gmail OAuth setup
```bash
./herald --setup-gmail
```
Uses Google OAuth 2.0 with localhost:8585 callback. Tokens saved to config.json.

## Config struct (config.json)
```json
{
  "anthropic_api_key": "",
  "gmail_client_id": "",
  "gmail_client_secret": "",
  "gmail_access_token": "",
  "gmail_refresh_token": "",
  "last_fetch_date": "",
  "theme": "dark"
}
```
`ANTHROPIC_API_KEY` env var takes priority over file.

## Claude API — key details
- Endpoint: `https://api.anthropic.com/v1/messages`
- Model: `claude-sonnet-4-20250514`
- Headers: `x-api-key`, `anthropic-version: 2023-06-01`, `Content-Type: application/json`
- No MCP — Gmail is called directly via Gmail API, emails passed as text in the prompt

## BuildDigest() flow
1. Load active sources from SQLite
2. Load negative preferences from SQLite
3. `sinceDate` = `cfg.LastFetchDate` or yesterday if empty
4. Fetch emails via `gmail.Client.FetchEmails(senderEmails, sinceDate)`
5. Format emails as text, send to Claude with system prompt
6. Parse JSON response → save digest + items to SQLite
7. Call `config.Save(cfg)` to persist `last_fetch_date = today`

## SaveDigest() — important
Uses `ON CONFLICT(date) DO UPDATE`, then queries `SELECT id FROM digests WHERE date = ?`
to get the actual row ID (SQLite `LastInsertId()` returns 0 on upsert).

## Default newsletter sources
| Name | Sender email |
|------|-------------|
| ByteByteGo | bytebytego@substack.com |
| The Pragmatic Engineer | pragmaticengineer@substack.com |
| The Pragmatic Engineer | pragmaticengineer+the-pulse@substack.com |
| The Pragmatic Engineer | pragmaticengineer+deepdives@substack.com |
| Architecture Weekly | architecture-weekly@substack.com |

## TUI — views and keybindings

### DigestView (split pane)
- Left 40%: scrollable item list with priority badges
- Right 60%: live preview of selected item (updates on j/k)
- Two-line bottom bar: status line + persistent keybinding hints
- Keys: `j/k` navigate, `Enter` detail, `o` open URL, `d` discard+learn,
  `x` remove (no learning), `+/-` adjust priority, `r` refresh, `s` settings, `?` help, `q` quit

### DetailView
- Full-screen single item; `o` browser, `d` discard, `b`/`Esc` back, `?` help

### SettingsView
- Two sections: Newsletter Sources + Learned Preferences
- `j/k` navigate, `a` add source, `Space` toggle active, `Del` remove, `b`/`Esc` back

### Help overlay
- `?` toggles a centered modal showing all keybindings for the current view
- `Esc` or `?` dismisses it

## Priority
- Claude-assigned 1–10 at fetch time (1=most relevant)
- Based on relevance to: Go, Java, Spring Boot, AWS, distributed systems, DB internals,
  system design, AI/LLM tooling, developer productivity
- Negative preferences (from discards) deprioritize matching topics (score 8–10)
- User can manually adjust with `+`/`-`; persisted via `db.UpdateItemPriority()`

## Error handling
- All errors surfaced in TUI status bar, never panic
- Gmail 401 → auto-refresh access token via refresh_token
- No emails found → "No new newsletters since <date>"
- No API key or Gmail → onboarding screen with instructions
