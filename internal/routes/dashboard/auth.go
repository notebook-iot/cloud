package dashboard

import (
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/notebook-iot/cloud/internal/context"
)

func Login(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/pages/login.html")
		if err != nil {
			return err
		}
		return tmpl.Execute(w, nil)
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		adminUser := os.Getenv("ADMIN_USERNAME")
		adminPass := os.Getenv("ADMIN_PASSWORD")

		if username == adminUser && password == adminPass {
			http.SetCookie(w, &http.Cookie{
				Name:     "auth_session",
				Value:    "authenticated",
				Path:     "/",
				Expires:  time.Now().Add(24 * time.Hour),
				HttpOnly: true,
			})

			http.Redirect(w, r, "/", http.StatusSeeOther)

			return nil
		}

		tmpl, err := template.ParseFiles("templates/pages/login.html")
		if err != nil {
			return err
		}

		return tmpl.Execute(w, map[string]interface{}{
			"Error": "Invalid username or password",
		})
	}

	return nil
}

func AuthMiddleware(ctx *context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_session")
			if err != nil || cookie.Value != "authenticated" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
