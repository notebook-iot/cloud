package dashboard

import (
	"html/template"
	"net/http"

	"github.com/notebook-iot/cloud/internal/context"
)

func Dashboard(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	tmpl, err := template.ParseFiles("templates/layout/layout.html", "templates/pages/dashboard.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, "layout.html", nil)
}

func Keys(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	tmpl, err := template.ParseFiles("templates/layout/layout.html", "templates/pages/keys.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, "layout.html", nil)
}

func Devices(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	tmpl, err := template.ParseFiles("templates/layout/layout.html", "templates/pages/devices.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, "layout.html", nil)
}

func Events(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	tmpl, err := template.ParseFiles("templates/layout/layout.html", "templates/pages/events.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, "layout.html", nil)
}

func Visualization(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	tmpl, err := template.ParseFiles("templates/layout/layout.html", "templates/pages/visualization.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, "layout.html", nil)
}
