package claude

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/luca-trifilio/herald/internal/db"
	"github.com/luca-trifilio/herald/internal/models"
)

func ExtractTopics(database *sql.DB, client *Client, item models.NewsItem) error {
	system := "You are a topic classifier. Return ONLY a JSON array of 1-5 lowercase topic strings that best describe the given tech news item. No prose, no markdown."
	user := fmt.Sprintf("Title: %s\nSummary: %s", item.Title, item.Summary)

	raw, err := client.Complete(system, user)
	if err != nil {
		return fmt.Errorf("claude topics: %w", err)
	}

	cleaned := stripFences(raw)
	var topics []string
	if err := json.Unmarshal([]byte(cleaned), &topics); err != nil {
		return fmt.Errorf("parse topics JSON: %w", err)
	}

	for _, topic := range topics {
		if err := db.UpsertPreference(database, topic, -1.0); err != nil {
			return err
		}
	}
	return nil
}
