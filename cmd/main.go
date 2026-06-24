package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Cryezidl/url-shortener/cfg"
	"github.com/Cryezidl/url-shortener/handlers"
	"github.com/Cryezidl/url-shortener/storage"
	"github.com/go-chi/chi/v5"
)

func main() {
	config := cfg.LoadConfig()
	dataStorage, err := storage.NewStorage(config.DBPath)
	if err != nil {
		panic(fmt.Errorf("failed to initialize storage: %w", err))
	}

	logger := slog.Default()
	redirectHandler := handlers.NewRedirecHandler(dataStorage, logger)
	apiHandler := handlers.NewAPIHandler(dataStorage, logger)

	//Setup router
	router := chi.NewRouter()

	router.Route("/api", func(r chi.Router) {

		r.Get("/{shortpath}", apiHandler.GetRule)
		r.Get("/{shortpath}/stats", apiHandler.GetStats)

		r.Post("/", apiHandler.CreateRule)
		r.Delete("/{shortpath}", apiHandler.DeleteRule)
	})

	router.Get("/{shortname}", redirectHandler.Redirect)

	if err := http.ListenAndServe(":"+config.Port, router); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
