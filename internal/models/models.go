package models

import "time"

type NewsItem struct {
	ID        int64
	DigestID  int64
	Title     string
	Summary   string
	URL       string
	Source    string
	Priority  int
	Discarded bool
	Opened    bool
	CreatedAt time.Time
}

type Digest struct {
	ID        int64
	Date      string
	CreatedAt time.Time
	RawJSON   string
	Items     []NewsItem
}

type Preference struct {
	ID          int64
	Topic       string
	Weight      float64
	Occurrences int
	UpdatedAt   time.Time
}

type NewsletterSource struct {
	ID          int64
	Name        string
	SenderEmail string
	Active      bool
}

type Config struct {
	AnthropicAPIKey   string `json:"anthropic_api_key"`
	GmailClientID     string `json:"gmail_client_id"`
	GmailClientSecret string `json:"gmail_client_secret"`
	GmailAccessToken  string `json:"gmail_access_token"`
	GmailRefreshToken string `json:"gmail_refresh_token"`
	LastFetchDate     string `json:"last_fetch_date"`
	Theme             string `json:"theme"`
}
