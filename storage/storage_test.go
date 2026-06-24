package storage

import (
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStorage spins up a fresh in-memory sqlite storage for each test.
// ":memory:" keeps tests fast while still exercising real SQL.
func newTestStorage(t *testing.T) (*Storage, *slog.Logger) {
	t.Helper()
	s, err := NewStorage(":memory:")
	require.NoError(t, err, "storage should initialize")
	t.Cleanup(func() { _ = s.db.Close() })
	return s, slog.Default()
}

func TestNewStorage(t *testing.T) {
	s, err := NewStorage(":memory:")
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.NoError(t, s.db.Ping())
	_ = s.db.Close()
}

func TestStorage_AddRule(t *testing.T) {
	tests := []struct {
		name      string
		shortPath string
		targetURL string
		ttl       time.Duration
	}{
		{
			name:      "Add permanent rule",
			shortPath: "/google",
			targetURL: "https://google.com",
			ttl:       0,
		},
		{
			name:      "Add temporary rule",
			shortPath: "/temp",
			targetURL: "https://temp.com",
			ttl:       time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, l := newTestStorage(t)
			s.AddRule(tt.shortPath, tt.targetURL, tt.ttl, l)

			rule := s.GetRule(tt.shortPath, l)
			require.NotNil(t, rule, "rule should be created")
			assert.Equal(t, tt.targetURL, rule.TargetURL)
			assert.NotZero(t, rule.CreatedAt)

			if tt.ttl > 0 {
				assert.NotNil(t, rule.ExpiresAt, "temporary rule should have ExpiresAt")
			} else {
				assert.Nil(t, rule.ExpiresAt, "permanent rule should not have ExpiresAt")
			}
		})
	}
}

func TestStorage_AddRule_Upsert(t *testing.T) {
	s, l := newTestStorage(t)

	s.AddRule("/dup", "https://old.com", 0, l)
	s.AddRule("/dup", "https://new.com", 0, l) // same short path -> should update, not error

	rule := s.GetRule("/dup", l)
	require.NotNil(t, rule)
	assert.Equal(t, "https://new.com", rule.TargetURL, "target should be updated on conflict")
}

func TestStorage_DeleteRule(t *testing.T) {
	s, l := newTestStorage(t)

	s.AddRule("/google", "https://google.com", 0, l)
	s.DeleteRule("/google", l)

	rule := s.GetRule("/google", l)
	assert.Nil(t, rule, "rule should be deleted")
}

func TestStorage_DeleteRule_NonExistent(t *testing.T) {
	s, l := newTestStorage(t)
	// deleting a missing rule should be a no-op, not a panic
	assert.NotPanics(t, func() { s.DeleteRule("/nope", l) })
}

func TestStorage_GetRule(t *testing.T) {
	tests := []struct {
		name      string
		shortPath string
		targetURL string
		wantFind  string
		wantNil   bool
	}{
		{
			name:      "Getting rule",
			shortPath: "/google",
			targetURL: "https://google.com",
			wantFind:  "/google",
			wantNil:   false,
		},
		{
			name:      "Getting non-existent rule",
			shortPath: "/google",
			targetURL: "https://google.com",
			wantFind:  "/nonexistent",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, l := newTestStorage(t)
			s.AddRule(tt.shortPath, tt.targetURL, 0, l)

			rule := s.GetRule(tt.wantFind, l)
			if tt.wantNil {
				assert.Nil(t, rule, "rule should be nil")
				return
			}
			require.NotNil(t, rule, "rule should be found")
			assert.Equal(t, tt.targetURL, rule.TargetURL)
			assert.NotZero(t, rule.CreatedAt)
		})
	}
}

func TestStorage_GetStats(t *testing.T) {
	tests := []struct {
		name      string
		shortPath string
		targetURL string
		wantFind  string
		wantNil   bool
		hitsCount int
	}{
		{
			name:      "Get stats for existing rule",
			shortPath: "/google",
			targetURL: "https://google.com",
			wantFind:  "/google",
			wantNil:   false,
			hitsCount: 0,
		},
		{
			name:      "Get stats for non-existent rule",
			shortPath: "/google",
			targetURL: "https://google.com",
			wantFind:  "/nonexistent",
			wantNil:   true,
			hitsCount: 0,
		},
		{
			name:      "Get stats for rule with hits",
			shortPath: "/github",
			targetURL: "https://github.com",
			wantFind:  "/github",
			wantNil:   false,
			hitsCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, l := newTestStorage(t)
			s.AddRule(tt.shortPath, tt.targetURL, 0, l)

			for i := 0; i < tt.hitsCount; i++ {
				s.IncrementStats(tt.shortPath, l)
			}

			stats := s.GetStats(tt.wantFind, l)
			if tt.wantNil {
				assert.Nil(t, stats, "stats should be nil")
				return
			}

			require.NotNil(t, stats, "stats should exist")
			assert.Equal(t, int64(tt.hitsCount), stats.Hits, "hits count mismatch")

			if tt.hitsCount > 0 {
				assert.NotZero(t, stats.LastAccessed, "LastAccessed should be set")
			}
		})
	}
}

func TestStorage_IncrementStats(t *testing.T) {
	tests := []struct {
		name           string
		shortPath      string
		targetURL      string
		incrementFor   string
		incrementCount int
	}{
		{
			name:           "Increment stats for existing rule",
			shortPath:      "/google",
			targetURL:      "https://google.com",
			incrementFor:   "/google",
			incrementCount: 3,
		},
		{
			name:           "Increment stats for non-existent rule (no-op, no panic)",
			shortPath:      "/google",
			targetURL:      "https://google.com",
			incrementFor:   "/nonexistent",
			incrementCount: 1,
		},
		{
			name:           "Increment stats multiple times",
			shortPath:      "/github",
			targetURL:      "https://github.com",
			incrementFor:   "/github",
			incrementCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, l := newTestStorage(t)
			s.AddRule(tt.shortPath, tt.targetURL, 0, l)

			assert.NotPanics(t, func() {
				for i := 0; i < tt.incrementCount; i++ {
					s.IncrementStats(tt.incrementFor, l)
				}
			})

			if tt.incrementFor == tt.shortPath {
				stats := s.GetStats(tt.shortPath, l)
				require.NotNil(t, stats)
				assert.Equal(t, int64(tt.incrementCount), stats.Hits)
				assert.NotZero(t, stats.LastAccessed)
			}
		})
	}
}

func TestStorage_RuleExpired(t *testing.T) {
	tests := []struct {
		name      string
		shortPath string
		targetURL string
		ttl       time.Duration
		checkPath string
		sleep     time.Duration
		want      bool
	}{
		{
			name:      "Permanent rule never expires",
			shortPath: "/google",
			targetURL: "https://google.com",
			ttl:       0,
			checkPath: "/google",
			sleep:     0,
			want:      false,
		},
		{
			name:      "Temporary rule not expired yet",
			shortPath: "/temp",
			targetURL: "https://temp.com",
			ttl:       time.Hour,
			checkPath: "/temp",
			sleep:     0,
			want:      false,
		},
		{
			name:      "Temporary rule expired after TTL",
			shortPath: "/short",
			targetURL: "https://short.com",
			ttl:       50 * time.Millisecond,
			checkPath: "/short",
			sleep:     100 * time.Millisecond,
			want:      true,
		},
		{
			name:      "Non-existent rule considered expired",
			shortPath: "/google",
			targetURL: "https://google.com",
			ttl:       0,
			checkPath: "/nonexistent",
			sleep:     0,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, l := newTestStorage(t)
			s.AddRule(tt.shortPath, tt.targetURL, tt.ttl, l)

			if tt.sleep > 0 {
				time.Sleep(tt.sleep)
			}

			expired := s.RuleExpired(tt.checkPath, l)
			assert.Equal(t, tt.want, expired, "RuleExpired result mismatch")
		})
	}
}

// TestStorage_Persistence is the key advantage over the old in-memory map:
// data survives reopening the same database file.
func TestStorage_Persistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist_test.db")
	l := slog.Default()

	// First session: write a rule and rack up some hits.
	s1, err := NewStorage(dbPath)
	require.NoError(t, err)
	s1.AddRule("/persist", "https://persist.com", 0, l)
	s1.IncrementStats("/persist", l)
	s1.IncrementStats("/persist", l)
	require.NoError(t, s1.db.Close())

	// Second session: reopen the same file, data should still be there.
	s2, err := NewStorage(dbPath)
	require.NoError(t, err)
	defer s2.db.Close()

	rule := s2.GetRule("/persist", l)
	require.NotNil(t, rule, "rule should survive a restart")
	assert.Equal(t, "https://persist.com", rule.TargetURL)

	stats := s2.GetStats("/persist", l)
	require.NotNil(t, stats)
	assert.Equal(t, int64(2), stats.Hits, "hits should survive a restart")
}
