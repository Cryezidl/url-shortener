package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/Cryezidl/url-shortener/pkg"
	"github.com/Cryezidl/url-shortener/storage"
)

type RedirectHandler struct {
	dataStorage *storage.Storage
	logger      *slog.Logger
}

func NewRedirecHandler(dataStorage *storage.Storage, logger *slog.Logger) *RedirectHandler {
	return &RedirectHandler{dataStorage: dataStorage, logger: logger}
}

/*
GET "shortpath" -> GetRule
*/

func (h *RedirectHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortPath := strings.TrimPrefix(r.URL.Path, "/")
	rule := h.dataStorage.GetRule(shortPath, h.logger)
	if rule == nil {
		pkg.RespondWithError(w, http.StatusNotFound, "unknown shortname", h.logger)
		return
	}

	if h.dataStorage.RuleExpired(shortPath, h.logger) {
		pkg.RespondWithError(w, http.StatusNotFound, "rule was expired", h.logger)
		return
	}
	h.dataStorage.IncrementStats(shortPath, h.logger)
	http.Redirect(w, r, rule.TargetURL, http.StatusFound)
}
