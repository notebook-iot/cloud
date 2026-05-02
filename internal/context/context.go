package context

import (
	"database/sql"
	"html/template"
	"log/slog"
)

type Context struct {
	Logger    *slog.Logger
	Templates *template.Template
	DB        *sql.DB
}
