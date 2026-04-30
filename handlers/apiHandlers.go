package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Cryezidl/url-shortener.git/pkg"
	"github.com/Cryezidl/url-shortener.git/storage"
	"github.com/go-chi/chi/v5"
)

type APIHandler struct {
	dataStorage *storage.Storage
	logger      *slog.Logger
}

func NewAPIHandler(dataStorage *storage.Storage, logger *slog.Logger) *APIHandler {
	return &APIHandler{dataStorage: dataStorage, logger: logger}
}

func (h *APIHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimSpace(chi.URLParam(r, "shortpath"))

	rule := h.dataStorage.GetRule(shortURL, h.logger)
	if rule == nil {
		pkg.RespondWithJSON(w, http.StatusNotFound, "rule not found", h.logger)
		return
	}

	pkg.RespondWithJSON(w, http.StatusFound, rule, h.logger)
}

func (h *APIHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimSpace(chi.URLParam(r, "shortpath"))

	stats := h.dataStorage.GetStats(shortURL, h.logger)
	if stats == nil {
		pkg.RespondWithJSON(w, http.StatusOK, "stats not found", h.logger)
		return
	}

	pkg.RespondWithJSON(w, http.StatusFound, map[string]any{
		"Hits":         stats.Hits,
		"LastAccessed": stats.LastAccessed,
	}, h.logger)
}

func (h *APIHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	type createRequest struct {
		ShortPath string         `json:"shortname"`
		TargetURL string         `json:"targeturl"`
		Ttl       *time.Duration `json:"ttl"`
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.RespondWithJSON(w, http.StatusBadRequest, "failed to decode json", h.logger)
		return
	}
	var ttl time.Duration
	if req.Ttl != nil {
		ttl = *req.Ttl
	} else {
		ttl = 0
	}
	if strings.TrimSpace(req.ShortPath) == "" {
		pkg.RespondWithJSON(w, http.StatusBadRequest, "wrong shortpath", h.logger)
		return
	}

	h.dataStorage.AddRule(req.ShortPath, req.TargetURL, ttl, h.logger)

	w.WriteHeader(http.StatusOK)
}

func (h *APIHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	shortURL := strings.TrimSpace(chi.URLParam(r, "shortpath"))
	h.dataStorage.DeleteRule(shortURL, h.logger)
	w.WriteHeader(http.StatusOK)
}
