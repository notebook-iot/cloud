package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/notebook-iot/cloud/internal/context"
)

type Telemetry struct {
	DeviceID          string    `json:"device_id"`
	API_Key           string    `json:"api_key"`
	JSON              string    `json:"json"`
	WiFi_Network_Name string    `json:"wifi_network_name"`
	Up_time           int64     `json:"up_time"`
	Latency           int64     `json:"latency"`
	Mac_Address       string    `json:"mac_address"`
	Temperature       float64   `json:"temperature"`
	Timestamp         time.Time `json:"timestamp"`
}

func Ingest(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	var data Telemetry
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ctx.Logger.Error("Failed to decode JSON", "err", err)
		return err
	}

	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	query := `INSERT INTO sensor_data 
	(device_id, api_key, json, wifi_network_name, up_time, latency, mac_address, 
	temperature, timestamp) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = ctx.DB.Exec(query, data.DeviceID, data.API_Key, data.JSON, data.WiFi_Network_Name,
		data.Up_time, data.Latency, data.Mac_Address, data.Temperature, data.Timestamp)
	if err != nil {
		ctx.Logger.Error("Failed to insert data into database", "err", err)
		return err
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Data recorded"))

	return nil
}

func ValidateDevice(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	var incoming struct {
		DeviceID string `json:"device_id"`
		APIKey   string `json:"api_key"`
		MAC      string `json:"mac_address"`
	}
	// Decode incoming JSON...

	// 1. Hash the incoming API Key to compare with DB
	hash := sha256.Sum256([]byte(incoming.APIKey))
	hashedIncoming := hex.EncodeToString(hash[:])

	// 2. Fetch the stored record
	var dbMac string
	var status string
	err := ctx.DB.QueryRow("SELECT COALESCE(mac_address, ''), status FROM devices WHERE device_id = $1 AND hashed_api_key = $2",
		incoming.DeviceID, hashedIncoming).Scan(&dbMac, &status)

	if err != nil {
		// FAIL: Key/ID combo doesn't exist
		ctx.Logger.Warn("Unauthorized access attempt", "id", incoming.DeviceID)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	// 3. Perform the "Approval" Check (Hardware Locking)
	if dbMac == "" {
		// First time this key is used! Lock it to this physical MAC address.
		_, err = ctx.DB.Exec("UPDATE devices SET mac_address = $1, status = 'approved' WHERE device_id = $2",
			incoming.MAC, incoming.DeviceID)
		ctx.Logger.Info("New device approved and locked to MAC", "id", incoming.DeviceID, "mac", incoming.MAC)
	} else if dbMac != incoming.MAC {
		// FAIL: The key is valid, but the MAC address is different (Cloning detected)
		ctx.Logger.Error("Hardware mismatch!", "id", incoming.DeviceID, "expected", dbMac, "got", incoming.MAC)
		http.Error(w, "Forbidden: Invalid Hardware", http.StatusForbidden)
		return nil
	}

	// All clear! Proceed to store data.
	return nil
}
