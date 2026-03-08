package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luca-trifilio/herald/internal/claude"
	"github.com/luca-trifilio/herald/internal/config"
	"github.com/luca-trifilio/herald/internal/db"
	"github.com/luca-trifilio/herald/internal/gmail"
	"github.com/luca-trifilio/herald/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--setup-gmail" {
		setupGmail()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	database, err := db.Open()
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	defer database.Close()

	if n, err := db.CountSources(database); err != nil {
		log.Fatalf("DB: %v", err)
	} else if n == 0 {
		if err := db.SeedDefaultSources(database); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	var claudeClient *claude.Client
	if cfg.AnthropicAPIKey != "" {
		claudeClient = claude.NewClient(cfg.AnthropicAPIKey)
	}

	var gmailClient *gmail.Client
	if cfg.GmailClientID != "" && cfg.GmailRefreshToken != "" {
		gmailClient = gmail.NewClient(cfg.GmailClientID, cfg.GmailClientSecret, cfg.GmailAccessToken, cfg.GmailRefreshToken)
	}

	app := tui.NewApp(database, claudeClient, gmailClient, cfg)

	var fetchCmd tea.Cmd
	if claudeClient != nil && gmailClient != nil {
		today := time.Now().Format("2006-01-02")
		digest, err := db.GetTodaysDigest(database, today)
		if err == nil && digest != nil && len(digest.Items) > 0 {
			app.GetDigestView().SetItems(digest.Items)
			app.GetDigestView().SetStatus(fmt.Sprintf("%d items loaded", len(app.GetDigestView().Items())))
		} else if err == nil && digest == nil {
			fetchCmd = tui.SetTriggerFetch(app)
		}
	}

	p := tea.NewProgram(&rootModel{app: app, fetchCmd: fetchCmd}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func setupGmail() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	read := func(prompt string) string {
		fmt.Print(prompt)
		s, _ := reader.ReadString('\n')
		return strings.TrimSpace(s)
	}

	if cfg.GmailClientID == "" {
		fmt.Println("Gmail OAuth Setup")
		fmt.Println("─────────────────────────────────────────")
		fmt.Println("1. Go to https://console.cloud.google.com")
		fmt.Println("2. Create a project → Enable Gmail API")
		fmt.Println("3. APIs & Services → Credentials →")
		fmt.Println("   Create OAuth 2.0 Client ID (Desktop app)")
		fmt.Println("─────────────────────────────────────────")
		fmt.Println()
		cfg.GmailClientID = read("Client ID:     ")
		cfg.GmailClientSecret = read("Client Secret: ")
	}

	fmt.Println()
	fmt.Println("Open this URL in your browser:")
	fmt.Println()
	fmt.Println(" ", gmail.AuthURL(cfg.GmailClientID))
	fmt.Println()
	fmt.Println("Waiting for authorization (will open callback on localhost:8585)…")

	code, err := gmail.WaitForCode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth callback failed: %v\n", err)
		os.Exit(1)
	}

	tok, err := gmail.ExchangeCode(cfg.GmailClientID, cfg.GmailClientSecret, code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
		os.Exit(1)
	}

	cfg.GmailAccessToken = tok.AccessToken
	cfg.GmailRefreshToken = tok.RefreshToken

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ Gmail connected. Run ./herald to start.")
}

type rootModel struct {
	app      *tui.App
	fetchCmd tea.Cmd
}

func (r *rootModel) Init() tea.Cmd {
	cmds := []tea.Cmd{r.app.Init()}
	if r.fetchCmd != nil {
		cmds = append(cmds, r.fetchCmd)
	}
	return tea.Batch(cmds...)
}

func (r *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := r.app.Update(msg)
	return r, cmd
}

func (r *rootModel) View() string { return r.app.View() }
