package dashboard

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// CreateKey generates a new API key, stores its hash, and returns the raw key.
func CreateKey(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	// 1. Generate 16 cryptographically secure random bytes (32 hex characters)
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}

	// 2. Construct the full raw key
	rawSecret := hex.EncodeToString(bytes)
	prefix := "nb_live_"
	fullKey := prefix + rawSecret

	// 3. Hash the key for database storage
	hash := sha256.Sum256([]byte(fullKey))
	keyHash := hex.EncodeToString(hash[:])

	// 4. Extract the suffix for the UI (e.g., the last 4 characters)
	suffix := fullKey[len(fullKey)-4:]

	// 5. Insert the hash and metadata into the database
	query := `INSERT INTO api_keys (name, prefix, suffix, key_hash) VALUES ($1, $2, $3, $4)`
	// Hardcoding the name "New Key" for this example. You could parse this from a JSON request body.
	_, err := ctx.DB.Exec(query, "New Key", prefix, suffix, keyHash)
	if err != nil {
		ctx.Logger.Error("Failed to insert API key", "err", err)
		return err
	}

	// 6. Return the RAW key to the frontend (this is the ONLY time it is ever transmitted)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(map[string]string{
		"raw_key": fullKey,
	})
}
