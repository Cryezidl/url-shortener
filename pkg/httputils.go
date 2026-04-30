package pkg

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func RespondWithError(w http.ResponseWriter, code int, err string, log *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := map[string]interface{}{
		"error": err,
		"code":  code,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode error response",
			slog.Any("err", err),
			slog.Int("original_code", code),
		)
		http.Error(w, `{"error":"Internal server error"}`, code)
	}
}

func RespondWithJSON(w http.ResponseWriter, code int, data interface{}, log *slog.Logger) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error("failed to encode json data",
			slog.Any("err", err),
			slog.Int("original_code", code),
		)
		http.Error(w, `{"error":"Internal server error"}`, code)
	}
}
