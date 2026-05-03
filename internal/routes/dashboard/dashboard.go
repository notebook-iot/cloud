package dashboard

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/notebook-iot/cloud/internal/context"
)

type DashboardStats struct {
	TotalNodes   int
	OnlineNow    int
	AvgLatency   int
	EventsToday  int
	Devices      []DeviceRow
	RecentEvents []Event
}

type Event struct {
	Time        string
	DeviceID    string
	Temperature float64
}

type DeviceRow struct {
	DeviceID   string
	MacAddress string
	Uptime     string
	Network    string
}

var funcs = template.FuncMap{
	"sub": func(a, b int) int {
		return a - b
	},
}

func render(w http.ResponseWriter, page string, data interface{}) error {
	tmpl, err := template.New("layout.html").Funcs(funcs).ParseFiles("templates/layout/layout.html", "templates/pages/"+page)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, "layout.html", data)
}

func Dashboard(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	stats := DashboardStats{}

	// total nodes
	err := ctx.DB.QueryRow("SELECT COUNT(*) FROM devices").Scan(&stats.TotalNodes)
	if err != nil {
		ctx.Logger.Error("Failed to fetch total nodes", "err", err)
	}

	// online now (sent data in last 5 minutes)
	err = ctx.DB.QueryRow("SELECT COUNT(DISTINCT device_id) FROM sensor_data WHERE timestamp > NOW() - INTERVAL '5 minutes'").Scan(&stats.OnlineNow)
	if err != nil {
		ctx.Logger.Error("Failed to fetch online nodes", "err", err)
	}

	// avg latency
	err = ctx.DB.QueryRow("SELECT COALESCE(ROUND(AVG(latency)), 0) FROM sensor_data").Scan(&stats.AvgLatency)
	if err != nil {
		ctx.Logger.Error("Failed to fetch avg latency", "err", err)
	}

	// events today
	err = ctx.DB.QueryRow("SELECT COUNT(*) FROM sensor_data WHERE timestamp > CURRENT_DATE").Scan(&stats.EventsToday)
	if err != nil {
		ctx.Logger.Error("Failed to fetch events today", "err", err)
	}

	// recent devices
	rows, err := ctx.DB.Query(`
		SELECT device_id, mac_address, wifi_network_name 
		FROM (
			SELECT DISTINCT ON (device_id) device_id, mac_address, wifi_network_name, timestamp 
			FROM sensor_data 
			ORDER BY device_id, timestamp DESC
		) sub 
		ORDER BY timestamp DESC 
		LIMIT 5`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d DeviceRow
			if err := rows.Scan(&d.DeviceID, &d.MacAddress, &d.Network); err == nil {
				d.Uptime = "Active" // Placeholder for actual uptime calculation if needed
				stats.Devices = append(stats.Devices, d)
			}
		}
	}

	return render(w, "dashboard.html", stats)
}

type APIKey struct {
	ID        int
	Name      string
	Prefix    string
	Suffix    string
	CreatedAt string
}

func Keys(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	var keys []APIKey
	rows, err := ctx.DB.Query("SELECT id, name, prefix, suffix, created_at FROM api_keys ORDER BY created_at DESC")
	if err != nil {
		ctx.Logger.Error("Failed to fetch API keys", "err", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var k APIKey
			var createdAt time.Time
			if err := rows.Scan(&k.ID, &k.Name, &k.Prefix, &k.Suffix, &createdAt); err == nil {
				k.CreatedAt = createdAt.Format("Jan 02, 2006")
				keys = append(keys, k)
			}
		}
	}

	return render(w, "keys.html", keys)
}

func DeleteKey(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing key ID", http.StatusBadRequest)
		return nil
	}

	_, err := ctx.DB.Exec("DELETE FROM api_keys WHERE id = $1", id)
	if err != nil {
		ctx.Logger.Error("Failed to delete API key", "err", err, "id", id)
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func Devices(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	return render(w, "devices.html", nil)
}

func Events(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	return render(w, "events.html", nil)
}

func Visualization(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	return render(w, "visualization.html", nil)
}

func CreateKey(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}

	rawSecret := hex.EncodeToString(bytes)
	prefix := "nb_live_"
	fullKey := prefix + rawSecret

	hash := sha256.Sum256([]byte(fullKey))
	keyHash := hex.EncodeToString(hash[:])

	suffix := fullKey[len(fullKey)-4:]

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	query := `INSERT INTO api_keys (name, prefix, suffix, key_hash) VALUES ($1, $2, $3, $4)`

	_, err := ctx.DB.Exec(query, hex.EncodeToString(b), prefix, suffix, keyHash)
	if err != nil {
		ctx.Logger.Error("Failed to insert API key", "err", err)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	return json.NewEncoder(w).Encode(map[string]string{
		"raw_key": fullKey,
	})
}
