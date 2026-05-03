package ingest

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/notebook-iot/cloud/internal/context"
)

type Telemetry struct {
	DeviceID          string    `json:"device_id"`
	API_Key           string    `json:"api_key"`
	JSONPayload       string    `json:"json_payload"`
	WiFi_Network_Name string    `json:"wifi_network_name"`
	Up_time           int64     `json:"up_time"`
	Latency           int64     `json:"latency"`
	Mac_Address       string    `json:"mac_address"`
	Temperature       float64   `json:"temperature"`
	Timestamp         time.Time `json:"timestamp"`
}

func Ingest(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	// Ensure table exists (safeguard against migration issues)
	_, _ = ctx.DB.Exec(`CREATE TABLE IF NOT EXISTS devices (
		id SERIAL PRIMARY KEY,
		device_id TEXT UNIQUE NOT NULL,
		mac_address TEXT,
		hashed_api_key TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)

	var data Telemetry
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ctx.Logger.Error("Failed to decode JSON", "err", err)
		return err
	}

	if err = ValidateDevice(w, r, ctx, &data); err != nil {
		return nil
	}

	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	// Try to extract temperature from JSONPayload if it's 0
	if data.Temperature == 0 && data.JSONPayload != "" {
		var extra struct {
			Temp float64 `json:"temp"`
		}
		if err := json.Unmarshal([]byte(data.JSONPayload), &extra); err == nil {
			data.Temperature = extra.Temp
		}
	}

	query := `INSERT INTO sensor_data
	(device_id, api_key, json_payload, wifi_network_name, up_time, latency, mac_address,
	temperature, timestamp)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = ctx.DB.Exec(query, data.DeviceID, data.API_Key, data.JSONPayload, data.WiFi_Network_Name,
		data.Up_time, data.Latency, data.Mac_Address, data.Temperature, data.Timestamp)
	if err != nil {
		ctx.Logger.Error("Failed to insert data into database", "err", err)
		return err
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Data recorded"))

	return nil
}

func ValidateDevice(w http.ResponseWriter, r *http.Request, ctx *context.Context, data *Telemetry) error {
	hash := sha256.Sum256([]byte(data.API_Key))
	hashedIncoming := hex.EncodeToString(hash[:])

	// 1. Verify API Key is a valid fleet key
	var keyName string
	err := ctx.DB.QueryRow("SELECT name FROM api_keys WHERE key_hash = $1", hashedIncoming).Scan(&keyName)
	if err != nil {
		ctx.Logger.Warn("Unauthorized API key used", "id", data.DeviceID, "mac", data.Mac_Address)
		http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
		return err
	}

	var dbID string
	var dbMac string
	var status string

	// 2. Check if device exists
	err = ctx.DB.QueryRow("SELECT device_id, COALESCE(mac_address, ''), status FROM devices WHERE device_id = $1",
		data.DeviceID).Scan(&dbID, &dbMac, &status)

	if err == sql.ErrNoRows {
		// Auto-provision if it's a new device but has a valid fleet key
		ctx.Logger.Info("Provisioning new device", "id", data.DeviceID, "mac", data.Mac_Address)
		_, err = ctx.DB.Exec("INSERT INTO devices (device_id, mac_address, hashed_api_key, status) VALUES ($1, $2, $3, 'approved')",
			data.DeviceID, data.Mac_Address, hashedIncoming)
		if err != nil {
			ctx.Logger.Error("Failed to provision device", "err", err)
			return err
		}
		return nil
	} else if err != nil {
		ctx.Logger.Error("Database error during validation", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return err
	}

	// 3. Existing device validation
	if dbMac == "" {
		_, err = ctx.DB.Exec("UPDATE devices SET mac_address = $1, status = 'approved', hashed_api_key = $2 WHERE device_id = $3",
			data.Mac_Address, hashedIncoming, data.DeviceID)
		ctx.Logger.Info("Device approved and locked to MAC", "id", data.DeviceID, "mac", data.Mac_Address)
	} else if dbMac != data.Mac_Address {
		ctx.Logger.Error("Hardware mismatch!", "id", data.DeviceID, "expected", dbMac, "got", data.Mac_Address)
		http.Error(w, "Forbidden: Invalid Hardware", http.StatusForbidden)
		return http.ErrAbortHandler
	}

	return nil
}
