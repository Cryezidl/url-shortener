package storage

import (
	"log/slog"
	"sync"
	"time"
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
	rules map[string]*RedirectRule
	stats map[string]*RuleStats
	mu    sync.RWMutex
}

func NewStorage() *Storage {
	rules := make(map[string]*RedirectRule)
	stats := make(map[string]*RuleStats)
	return &Storage{rules: rules, stats: stats}
}

func (s *Storage) IncrementStats(shortPath string, log *slog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stats, ok := s.stats[shortPath]; ok {
		stats.Hits++
		stats.LastAccessed = time.Now()
		log.Debug("Stats were increased", "func", "IncrementStats", "shortPath", shortPath)
	}
}

func (s *Storage) RuleExpired(shortPath string, log *slog.Logger) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.rules[shortPath]; ok {

		if val.ExpiresAt != nil && time.Now().After(*val.ExpiresAt) {
			log.Debug("Rule has expired", "func", "RuleExpired", "shortPath", shortPath)
			return true
		}
	}
	return false
}

func (s *Storage) AddRule(shortPath, targetURL string, ttl time.Duration, log *slog.Logger) {
	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}

	rule := &RedirectRule{
		TargetURL: targetURL,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[shortPath] = rule

	s.stats[shortPath] = &RuleStats{
		Hits:         0,
		LastAccessed: time.Time{},
	}

	log.Debug("Rule was added", "func", "AddRule", "shortPath", shortPath, "targetURL", targetURL, "ttl in minutes", ttl.String())
}

func (s *Storage) DeleteRule(shortPath string, log *slog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val, ok := s.rules[shortPath]; ok {
		log.Debug("Deleting rule", "func", "DeleteRule", "shortPath", shortPath, "targetRule", val.TargetURL)
		delete(s.rules, shortPath)
	}
}

func (s *Storage) GetRule(shortPath string, log *slog.Logger) *RedirectRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.rules[shortPath]; ok {
		log.Debug("Found rule", "func", "GetRule", "shortPath", shortPath, "targetRule", val.TargetURL)
		rule := &RedirectRule{
			TargetURL: val.TargetURL,
			CreatedAt: val.CreatedAt,
			ExpiresAt: val.ExpiresAt,
		}
		return rule
	}
	log.Debug("Rule not found", "func", "GetRule", "shortPath", shortPath)
	return nil
}

func (s *Storage) GetStats(shortPath string, log *slog.Logger) *RuleStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.stats[shortPath]; ok {
		log.Debug("Found stats", "func", "GetStats", "shortPath", shortPath, "Hits", val.Hits)
		return val
	}
	log.Debug("Stats not found", "func", "GetStats", "shortPath", shortPath)
	return nil
}
