package main

import (
	"log/slog"
	"net/http"

	"github.com/Cryezidl/url-shortener.git/handlers"
	"github.com/Cryezidl/url-shortener.git/storage"
	"github.com/go-chi/chi/v5"
)

func main() {
	dataStorage := storage.NewStorage()
	logger := slog.Default()
	redirectHandler := handlers.NewRedirecHandler(dataStorage, logger)
	apiHandler := handlers.NewAPIHandler(dataStorage, logger)

	//Настроиваем роуты
	router := chi.NewRouter()

	router.Route("/api", func(r chi.Router) {

		r.Get("/{shortpath}", apiHandler.GetRule)
		r.Get("/{shortpath}/stats", apiHandler.GetStats)

		r.Post("/", apiHandler.CreateRule)
		r.Delete("/{shortpath}", apiHandler.DeleteRule)
	})

	router.Get("/{shortname}", redirectHandler.Redirect)

	http.ListenAndServe(":8080", router)
}
