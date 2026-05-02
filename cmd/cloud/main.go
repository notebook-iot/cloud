package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/notebook-iot/cloud/internal/context"
	"github.com/notebook-iot/cloud/internal/routes/dashboard"
	"github.com/notebook-iot/cloud/internal/routes/ingest"
)

var logger *slog.Logger

func main() {
	logger = slog.Default()

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			logger.Error("error loading .env", "err", err)

			return
		}
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	context := context.Context{
		Logger: logger,
	}

	r.Post("/ingest", func(w http.ResponseWriter, r *http.Request) {
		handleErr(ingest.Ingest(w, r, &context), w, "Error ingesting data")
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		handleErr(dashboard.Dashboard(w, r, &context), w, "Error fetching dashboard")
	})

	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}

func handleErr(err error, w http.ResponseWriter, reason string) {
	if err != nil {
		http.Error(w, reason, http.StatusInternalServerError)
	}
}
