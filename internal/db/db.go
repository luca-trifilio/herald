package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luca-trifilio/herald/internal/models"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

func Open() (*sql.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".local", "share", "newsdigest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "newsdigest.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	return db, nil
}

// Sources

func GetActiveSources(db *sql.DB) ([]models.NewsletterSource, error) {
	rows, err := db.Query(`SELECT id, name, sender_email, active FROM newsletter_sources WHERE active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []models.NewsletterSource
	for rows.Next() {
		var s models.NewsletterSource
		var active int
		if err := rows.Scan(&s.ID, &s.Name, &s.SenderEmail, &active); err != nil {
			return nil, err
		}
		s.Active = active == 1
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func GetAllSources(db *sql.DB) ([]models.NewsletterSource, error) {
	rows, err := db.Query(`SELECT id, name, sender_email, active FROM newsletter_sources ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []models.NewsletterSource
	for rows.Next() {
		var s models.NewsletterSource
		var active int
		if err := rows.Scan(&s.ID, &s.Name, &s.SenderEmail, &active); err != nil {
			return nil, err
		}
		s.Active = active == 1
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func CountSources(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM newsletter_sources`).Scan(&count)
	return count, err
}

func InsertSource(db *sql.DB, name, email string) error {
	_, err := db.Exec(`INSERT INTO newsletter_sources (name, sender_email) VALUES (?, ?)`, name, email)
	return err
}

func DeleteSource(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM newsletter_sources WHERE id = ?`, id)
	return err
}

func ToggleSource(db *sql.DB, id int64, active bool) error {
	a := 0
	if active {
		a = 1
	}
	_, err := db.Exec(`UPDATE newsletter_sources SET active = ? WHERE id = ?`, a, id)
	return err
}

// Digests

func GetTodaysDigest(db *sql.DB, date string) (*models.Digest, error) {
	var d models.Digest
	err := db.QueryRow(`SELECT id, date, created_at, COALESCE(raw_json,'') FROM digests WHERE date = ?`, date).
		Scan(&d.ID, &d.Date, &d.CreatedAt, &d.RawJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := GetItemsByDigest(db, d.ID)
	if err != nil {
		return nil, err
	}
	d.Items = items
	return &d, nil
}

func SaveDigest(db *sql.DB, date, rawJSON string) (int64, error) {
	_, err := db.Exec(`INSERT INTO digests (date, raw_json) VALUES (?, ?)
		ON CONFLICT(date) DO UPDATE SET raw_json = excluded.raw_json, created_at = CURRENT_TIMESTAMP`,
		date, rawJSON)
	if err != nil {
		return 0, err
	}
	var id int64
	err = db.QueryRow(`SELECT id FROM digests WHERE date = ?`, date).Scan(&id)
	return id, err
}

func SaveItems(db *sql.DB, digestID int64, items []models.NewsItem) error {
	_, err := db.Exec(`DELETE FROM news_items WHERE digest_id = ?`, digestID)
	if err != nil {
		return err
	}
	for _, item := range items {
		_, err := db.Exec(`INSERT INTO news_items (digest_id, title, summary, url, source, priority) VALUES (?, ?, ?, ?, ?, ?)`,
			digestID, item.Title, item.Summary, item.URL, item.Source, item.Priority)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetItemsByDigest(db *sql.DB, digestID int64) ([]models.NewsItem, error) {
	rows, err := db.Query(`SELECT id, digest_id, title, summary, COALESCE(url,''), COALESCE(source,''), priority, discarded, opened, created_at
		FROM news_items WHERE digest_id = ? ORDER BY priority ASC`, digestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.NewsItem
	for rows.Next() {
		var it models.NewsItem
		var discarded, opened int
		if err := rows.Scan(&it.ID, &it.DigestID, &it.Title, &it.Summary, &it.URL, &it.Source, &it.Priority, &discarded, &opened, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.Discarded = discarded == 1
		it.Opened = opened == 1
		items = append(items, it)
	}
	return items, rows.Err()
}

func DiscardItem(db *sql.DB, id int64) error {
	_, err := db.Exec(`UPDATE news_items SET discarded = 1 WHERE id = ?`, id)
	return err
}

func MarkOpened(db *sql.DB, id int64) error {
	_, err := db.Exec(`UPDATE news_items SET opened = 1 WHERE id = ?`, id)
	return err
}

func UpdateItemPriority(database *sql.DB, itemID int64, priority int) error {
	_, err := database.Exec(`UPDATE news_items SET priority = ? WHERE id = ?`, priority, itemID)
	return err
}

// Preferences

func GetNegativePreferences(db *sql.DB) ([]models.Preference, error) {
	rows, err := db.Query(`SELECT id, topic, weight, occurrences, updated_at FROM preferences WHERE weight < 0 ORDER BY weight ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prefs []models.Preference
	for rows.Next() {
		var p models.Preference
		if err := rows.Scan(&p.ID, &p.Topic, &p.Weight, &p.Occurrences, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}

func GetAllPreferences(db *sql.DB) ([]models.Preference, error) {
	rows, err := db.Query(`SELECT id, topic, weight, occurrences, updated_at FROM preferences ORDER BY weight ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prefs []models.Preference
	for rows.Next() {
		var p models.Preference
		if err := rows.Scan(&p.ID, &p.Topic, &p.Weight, &p.Occurrences, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}

func UpsertPreference(db *sql.DB, topic string, weight float64) error {
	_, err := db.Exec(`INSERT INTO preferences (topic, weight, occurrences, updated_at)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(topic) DO UPDATE SET
			occurrences = occurrences + 1,
			updated_at = excluded.updated_at`,
		topic, weight, time.Now())
	return err
}

func DeletePreference(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM preferences WHERE id = ?`, id)
	return err
}

// Seed default sources

func SeedDefaultSources(db *sql.DB) error {
	defaults := []struct{ name, email string }{
		{"ByteByteGo", "bytebytego@substack.com"},
		{"The Pragmatic Engineer", "pragmaticengineer@substack.com"},
		{"The Pragmatic Engineer", "pragmaticengineer+the-pulse@substack.com"},
		{"The Pragmatic Engineer", "pragmaticengineer+deepdives@substack.com"},
		{"Architecture Weekly", "architecture-weekly@substack.com"},
	}
	for _, d := range defaults {
		if _, err := db.Exec(`INSERT OR IGNORE INTO newsletter_sources (name, sender_email) VALUES (?, ?)`, d.name, d.email); err != nil {
			return err
		}
	}
	return nil
}
