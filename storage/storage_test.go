package storage

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStorage_AddRule(t *testing.T) {
	tests := []struct {
		name string

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
			s := NewStorage()
			l := slog.Default()
			s.AddRule(tt.shortPath, tt.targetURL, tt.ttl, l)
			rule := s.GetRule(tt.shortPath, l)

			assert.NotNil(t, rule, "Rule should be created")
			assert.Equal(t, tt.targetURL, rule.TargetURL)
			assert.NotZero(t, rule.CreatedAt)
		})
	}
}

func TestStorage_DeleteRule(t *testing.T) {
	tests := []struct {
		name string

		shortPath string
		targetURL string
	}{
		{
			name:      "Deleting test",
			shortPath: "/google",
			targetURL: "https://google.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStorage()
			l := slog.Default()

			s.AddRule(tt.shortPath, tt.targetURL, 0, l)
			s.DeleteRule(tt.shortPath, l)
			rule := s.GetRule(tt.shortPath, l)
			assert.Nil(t, rule, "Rule should be deleted")
		})
	}
}

func TestStorage_GetRule(t *testing.T) {
	tests := []struct {
		name string

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
			s := NewStorage()
			l := slog.Default()
			s.AddRule(tt.shortPath, tt.targetURL, 0, l)

			rule := s.GetRule(tt.wantFind, l)
			if tt.wantNil {
				assert.Nil(t, rule, "Rule should be nil")
			} else {
				assert.NotNil(t, rule, "Rule should be created")
			}

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
			s := NewStorage()
			l := slog.Default()
			s.AddRule(tt.shortPath, tt.targetURL, 0, l)

			// Добавляем хиты если нужно
			for i := 0; i < tt.hitsCount; i++ {
				s.IncrementStats(tt.shortPath, l)
			}

			stats := s.GetStats(tt.wantFind, l)

			if tt.wantNil {
				assert.Nil(t, stats, "Stats should be nil")
				return
			}

			assert.NotNil(t, stats, "Stats should exist")
			assert.Equal(t, int64(tt.hitsCount), stats.Hits, "Hits count mismatch")

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
		wantPanic      bool
	}{
		{
			name:           "Increment stats for existing rule",
			shortPath:      "/google",
			targetURL:      "https://google.com",
			incrementFor:   "/google",
			incrementCount: 3,
			wantPanic:      false,
		},
		{
			name:           "Increment stats for non-existent rule (should not panic)",
			shortPath:      "/google",
			targetURL:      "https://google.com",
			incrementFor:   "/nonexistent",
			incrementCount: 1,
			wantPanic:      false,
		},
		{
			name:           "Increment stats multiple times",
			shortPath:      "/github",
			targetURL:      "https://github.com",
			incrementFor:   "/github",
			incrementCount: 10,
			wantPanic:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStorage()
			l := slog.Default()
			s.AddRule(tt.shortPath, tt.targetURL, 0, l)

			// Проверяем что нет паники
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("IncrementStats panicked: %v", r)
					}
				}
			}()

			// Инкрементируем статистику
			for i := 0; i < tt.incrementCount; i++ {
				s.IncrementStats(tt.incrementFor, l)
			}

			// Проверяем результат только для существующего правила
			if tt.incrementFor == tt.shortPath {
				stats := s.GetStats(tt.shortPath, l)
				assert.NotNil(t, stats)
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
			s := NewStorage()
			l := slog.Default()

			// Добавляем правило только если указан путь
			if tt.shortPath != "" {
				s.AddRule(tt.shortPath, tt.targetURL, tt.ttl, l)
			}

			// Ждем если нужно
			if tt.sleep > 0 {
				time.Sleep(tt.sleep)
			}

			expired := s.RuleExpired(tt.checkPath, l)
			assert.Equal(t, tt.want, expired, "RuleExpired result mismatch")
		})
	}
}
