package main

import (
	"database/sql"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/notebook-iot/cloud/internal/context"
	"github.com/notebook-iot/cloud/internal/routes/dashboard"
	"github.com/notebook-iot/cloud/internal/routes/ingest"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

var logger *slog.Logger
var templates *template.Template

func ParseTemplates(dir string) error {
	tmpl := template.New("").Funcs(template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	})

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".html" {
			_, err = tmpl.ParseFiles(path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	templates = tmpl

	return nil
}

func main() {
	logger = slog.Default()

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			logger.Error("error loading .env", "err", err)

			return
		}
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("error connecting to database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error("failed to set goose dialect", "err", err)
		os.Exit(1)
	}

	if err := goose.Up(db, "cmd/cloud/migrations"); err != nil {
		logger.Error("migration failed", "err", err)
		os.Exit(1)
	}
	logger.Info("migrations applied successfully")

	err = ParseTemplates("./templates")
	if err != nil {
		logger.Error("error parsing templates", "err", err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	context := context.Context{
		Logger:    logger,
		Templates: templates,
		DB:        db,
	}

	r.Post("/ingest", func(w http.ResponseWriter, r *http.Request) {
		handleErr(ingest.Ingest(w, r, &context), w, "ingest")
	})

	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		handleErr(dashboard.Login(w, r, &context), w, "login")
	})

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		handleErr(dashboard.Login(w, r, &context), w, "login")
	})

	r.Group(func(r chi.Router) {
		r.Use(dashboard.AuthMiddleware(&context))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			handleErr(dashboard.Dashboard(w, r, &context), w, "dashboard")
		})

		r.Get("/keys", func(w http.ResponseWriter, r *http.Request) {
			handleErr(dashboard.Keys(w, r, &context), w, "keys")
		})

		r.Get("/devices", func(w http.ResponseWriter, r *http.Request) {
			handleErr(dashboard.Devices(w, r, &context), w, "keys")
		})

		r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
			handleErr(dashboard.Events(w, r, &context), w, "keys")
		})

		r.Get("/visualization", func(w http.ResponseWriter, r *http.Request) {
			handleErr(dashboard.Visualization(w, r, &context), w, "keys")
		})

		r.Post("/api/keys", func(w http.ResponseWriter, r *http.Request) {
			handleErr(dashboard.CreateKey(w, r, &context), w, "create_key")
		})
	})

	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}

func handleErr(err error, w http.ResponseWriter, svc string) {
	if err != nil {
		http.Error(w, "An error occurred while attempting to process your request.", http.StatusInternalServerError)
		logger.Error("error handling route", "svc", svc, "err", err)
	}
}
