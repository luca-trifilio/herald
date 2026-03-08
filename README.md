# Herald

Terminal digest of your tech newsletters, powered by Gmail + Claude AI.

Every morning, Herald fetches emails from your configured newsletter senders, summarises each article via the Anthropic Claude API, and presents a prioritised, interactive digest in the terminal. Discarded items teach Herald what you dislike.

## Install

### Homebrew (macOS)
```bash
brew install luca-trifilio/herald/herald
```

### Go install
```bash
go install github.com/luca-trifilio/herald@latest
```

### Binary releases
Download from [GitHub Releases](https://github.com/luca-trifilio/herald/releases).

## Prerequisites

- An [Anthropic API key](https://console.anthropic.com/)
- Gmail account with OAuth 2.0 credentials (Google Cloud Console)

## Setup

### 1. Anthropic API key

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

Or add it to `~/.config/newsdigest/config.json`.

### 2. Gmail OAuth

```bash
./herald --setup-gmail
```

Follow the prompts: create an OAuth 2.0 Desktop app in [Google Cloud Console](https://console.cloud.google.com), enable the Gmail API, then paste your Client ID and Secret. Herald will open a local callback server on `:8585` and save the tokens automatically.

### 3. Run

```bash
./herald
```

On first launch Herald auto-fetches today's digest. Press `r` to force a re-fetch.

## Keybindings

### Digest view (main)
| Key | Action |
|-----|--------|
| `j` / `↓` | Next item |
| `k` / `↑` | Previous item |
| `Enter` | Open detail view |
| `o` | Open URL in browser |
| `d` | Discard item + learn (teaches Herald your dislikes) |
| `x` | Remove item without learning |
| `+` / `-` | Increase / decrease priority |
| `r` | Re-fetch today's digest |
| `s` | Settings |
| `?` | Help overlay |
| `q` | Quit |

### Detail view
| Key | Action |
|-----|--------|
| `o` | Open URL in browser |
| `d` | Discard + return |
| `Esc` / `b` | Back |
| `?` | Help overlay |

### Settings view
| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `a` | Add new source |
| `Space` | Toggle source active/inactive |
| `Del` / `Backspace` | Remove source or forget preference |
| `Esc` / `b` | Back |
| `?` | Help overlay |

## Layout

The main view is a **split pane**: the left side lists all items with priority badges; the right side shows a live preview of the selected item (summary, source, URL) as you navigate.

## Priority

Items are scored 1–10 by Claude at fetch time based on relevance to a senior backend/distributed-systems engineer (Go, Java, AWS, distributed systems, database internals, AI/LLM tooling, developer productivity). Lower = more relevant. You can manually adjust with `+`/`-`.

Discarded items (via `d`) have their topics extracted by Claude and stored in the preferences table with a negative weight, influencing future digest scoring.

## Config

`~/.config/newsdigest/config.json`

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

`ANTHROPIC_API_KEY` env var takes priority over the file.

## Data

SQLite DB at `~/.local/share/newsdigest/newsdigest.db`.

Tables: `digests`, `news_items`, `preferences`, `newsletter_sources`.

## Default newsletter sources

| Name | Sender |
|------|--------|
| ByteByteGo | bytebytego@substack.com |
| The Pragmatic Engineer | pragmaticengineer@substack.com |
| The Pragmatic Engineer | pragmaticengineer+the-pulse@substack.com |
| The Pragmatic Engineer | pragmaticengineer+deepdives@substack.com |
| Architecture Weekly | architecture-weekly@substack.com |

Add more via `s` → Settings → `a`.
