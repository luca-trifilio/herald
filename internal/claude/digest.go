package claude

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/luca-trifilio/herald/internal/config"
	"github.com/luca-trifilio/herald/internal/db"
	"github.com/luca-trifilio/herald/internal/gmail"
	"github.com/luca-trifilio/herald/internal/models"
)

type digestResponse struct {
	Date  string           `json:"date"`
	Items []digestItemJSON `json:"items"`
}

type digestItemJSON struct {
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	URL      string `json:"url"`
	Source   string `json:"source"`
	Priority int    `json:"priority"`
}

func BuildDigest(database *sql.DB, client *Client, gmailClient *gmail.Client, cfg *models.Config) (*models.Digest, error) {
	sources, err := db.GetActiveSources(database)
	if err != nil {
		return nil, fmt.Errorf("load sources: %w", err)
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no active newsletter sources configured")
	}

	prefs, err := db.GetNegativePreferences(database)
	if err != nil {
		return nil, fmt.Errorf("load preferences: %w", err)
	}

	sinceDate := cfg.LastFetchDate
	if sinceDate == "" {
		sinceDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	}
	today := time.Now().Format("2006-01-02")

	var senderEmails []string
	for _, s := range sources {
		senderEmails = append(senderEmails, s.SenderEmail)
	}

	emails, err := gmailClient.FetchEmails(senderEmails, sinceDate)
	if err != nil {
		return nil, fmt.Errorf("fetch emails: %w", err)
	}
	if len(emails) == 0 {
		return &models.Digest{Date: today}, fmt.Errorf("no new newsletters since %s", sinceDate)
	}

	var prefLines []string
	for _, p := range prefs {
		prefLines = append(prefLines, fmt.Sprintf("- %s (weight %.1f, seen %d times)", p.Topic, p.Weight, p.Occurrences))
	}
	prefBlock := "None."
	if len(prefLines) > 0 {
		prefBlock = strings.Join(prefLines, "\n")
	}

	system := fmt.Sprintf(`You are a tech newsletter digest builder for a senior backend/distributed-systems engineer.

Given the raw newsletter emails below, extract every individual article and return a JSON digest.

For each article:
- title: concise article title
- summary: 1-2 sentence summary
- url: the article URL (set to "" if not found — never fabricate URLs)
- source: the newsletter name (sender display name, not the email address)
- priority: integer 1-10 (1=most relevant, 10=least relevant)

Score by relevance to a senior backend engineer interested in:
Java, Spring Boot, Go, PostgreSQL, AWS, distributed systems, database internals,
system design, AI/LLM tooling, developer productivity.

DEPRIORITIZE (score 8-10) topics matching these learned dislikes:
%s

Return ONLY valid JSON — no prose, no markdown fences.
Format: {"date":"%s","items":[{"title":"...","summary":"...","url":"...","source":"...","priority":1}]}
If no articles found: {"date":"%s","items":[]}`,
		prefBlock, today, today,
	)

	user := "Here are the newsletter emails:\n\n" + gmail.FormatForPrompt(emails) + "\nBuild today's digest."

	raw, err := client.Complete(system, user)
	if err != nil {
		return nil, fmt.Errorf("claude API: %w", err)
	}

	cleaned := stripFences(raw)
	var parsed digestResponse
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("parse digest JSON: %w\nraw: %s", err, cleaned)
	}

	digestID, err := db.SaveDigest(database, today, raw)
	if err != nil {
		return nil, fmt.Errorf("save digest: %w", err)
	}

	var items []models.NewsItem
	for _, it := range parsed.Items {
		items = append(items, models.NewsItem{
			DigestID: digestID,
			Title:    it.Title,
			Summary:  it.Summary,
			URL:      it.URL,
			Source:   it.Source,
			Priority: it.Priority,
		})
	}
	if err := db.SaveItems(database, digestID, items); err != nil {
		return nil, fmt.Errorf("save items: %w", err)
	}

	cfg.LastFetchDate = today
	_ = config.Save(cfg)
	return db.GetTodaysDigest(database, today)
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.SplitN(s, "\n", 2)
		if len(lines) == 2 {
			s = lines[1]
		}
	}
	if strings.HasSuffix(s, "```") {
		s = s[:strings.LastIndex(s, "```")]
	}
	return strings.TrimSpace(s)
}
