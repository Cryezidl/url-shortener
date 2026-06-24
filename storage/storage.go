package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

type RedirectRule struct {
	TargetURL string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

type RuleStats struct {
	Hits         int64
	LastAccessed time.Time
}

type Storage struct {
	db *sql.DB
}

const initTableSql = `
CREATE TABLE IF NOT EXISTS rules (
    short_path TEXT PRIMARY KEY,
    target_url TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME
);

CREATE TABLE IF NOT EXISTS stats (
    short_path TEXT PRIMARY KEY REFERENCES rules(short_path) ON DELETE CASCADE,
    hits INTEGER NOT NULL DEFAULT 0,
    last_accessed DATETIME
);
`

func NewStorage(storagePath string) (*Storage, error) {
	db, err := sql.Open("sqlite", storagePath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err = db.Exec(initTableSql); err != nil {
		return nil, fmt.Errorf("can't initialize database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) IncrementStats(shortPath string, log *slog.Logger) {
	_, err := s.db.Exec(
		`UPDATE stats SET hits = hits + 1, last_accessed = ? WHERE short_path = ?`,
		time.Now(), shortPath,
	)
	if err != nil {
		log.Error("failed to increment stats", "func", "IncrementStats", "err", err)
	}
}

func (s *Storage) RuleExpired(shortPath string, log *slog.Logger) bool {
	rule := s.GetRule(shortPath, log)
	if rule == nil {
		return true
	}
	return rule.ExpiresAt != nil && time.Now().After(*rule.ExpiresAt)
}

func (s *Storage) AddRule(shortPath, targetURL string, ttl time.Duration, log *slog.Logger) {
	now := time.Now()
	var expiresAt *time.Time
	if ttl > 0 {
		t := now.Add(ttl)
		expiresAt = &t
	}

	_, err := s.db.Exec(
		`INSERT INTO rules (short_path, target_url, created_at, expires_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(short_path) DO UPDATE SET target_url=excluded.target_url, expires_at=excluded.expires_at`,
		shortPath, targetURL, now, expiresAt,
	)
	if err != nil {
		log.Error("failed to add rule", "func", "AddRule", "err", err)
		return
	}

	_, err = s.db.Exec(
		`INSERT INTO stats (short_path, hits, last_accessed) VALUES (?, 0, NULL)
		 ON CONFLICT(short_path) DO NOTHING`,
		shortPath,
	)
	if err != nil {
		log.Error("failed to init stats", "func", "AddRule", "err", err)
	}
}

func (s *Storage) DeleteRule(shortPath string, log *slog.Logger) {
	if _, err := s.db.Exec(`DELETE FROM rules WHERE short_path = ?`, shortPath); err != nil {
		log.Error("failed to delete rule", "func", "DeleteRule", "err", err)
	}
}

func (s *Storage) GetRule(shortPath string, log *slog.Logger) *RedirectRule {
	row := s.db.QueryRow(
		`SELECT target_url, created_at, expires_at FROM rules WHERE short_path = ?`,
		shortPath,
	)

	var rule RedirectRule
	var expiresAt sql.NullTime
	if err := row.Scan(&rule.TargetURL, &rule.CreatedAt, &expiresAt); err != nil {
		if err != sql.ErrNoRows {
			log.Error("failed to get rule", "func", "GetRule", "err", err)
		}
		return nil
	}
	if expiresAt.Valid {
		rule.ExpiresAt = &expiresAt.Time
	}
	return &rule
}

func (s *Storage) GetStats(shortPath string, log *slog.Logger) *RuleStats {
	row := s.db.QueryRow(
		`SELECT hits, last_accessed FROM stats WHERE short_path = ?`,
		shortPath,
	)

	var stats RuleStats
	var lastAccessed sql.NullTime
	if err := row.Scan(&stats.Hits, &lastAccessed); err != nil {
		if err != sql.ErrNoRows {
			log.Error("failed to get stats", "func", "GetStats", "err", err)
		}
		return nil
	}
	if lastAccessed.Valid {
		stats.LastAccessed = lastAccessed.Time
	}
	return &stats
}
