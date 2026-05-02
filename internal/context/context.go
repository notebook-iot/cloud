package context

import (
	"html/template"
	"log/slog"
)

type Context struct {
	Logger    *slog.Logger
	Templates *template.Template
}

// timescale integration and installing packages
