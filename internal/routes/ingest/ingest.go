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

	if err = ValidateDevice(w, r, ctx, &data); err != nil {
		return nil
	}

	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	query := `INSERT INTO sensor_data
	(device_id, api_key, json_payload, wifi_network_name, up_time, latency, mac_address,
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

func ValidateDevice(w http.ResponseWriter, r *http.Request, ctx *context.Context, data *Telemetry) error {
	hash := sha256.Sum256([]byte(data.API_Key))
	hashedIncoming := hex.EncodeToString(hash[:])

	var dbID string
	var dbMac string
	var status string

	err := ctx.DB.QueryRow("SELECT device_id, COALESCE(mac_address, ''), status FROM devices WHERE device_id = $1 AND hashed_api_key = $2",
		data.DeviceID, hashedIncoming).Scan(&dbID, &dbMac, &status)

	if err != nil {
		ctx.Logger.Info("Device ID not found, trying MAC fallback", "id", data.DeviceID, "mac", data.Mac_Address)

		err = ctx.DB.QueryRow("SELECT device_id, COALESCE(mac_address, ''), status FROM devices WHERE mac_address = $1 AND hashed_api_key = $2",
			data.Mac_Address, hashedIncoming).Scan(&dbID, &dbMac, &status)

		if err == nil {
			ctx.Logger.Info("Device matched via MAC fallback", "mac", data.Mac_Address, "canonical_id", dbID)
			// update the incoming data to use the canonical ID found in the database
			data.DeviceID = dbID
		}
	}

	if err != nil {
		ctx.Logger.Warn("Unauthorized access attempt", "id", data.DeviceID, "mac", data.Mac_Address, "err", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return err
	}

	if dbMac == "" {
		_, err = ctx.DB.Exec("UPDATE devices SET mac_address = $1, status = 'approved' WHERE device_id = $2",
			data.Mac_Address, data.DeviceID)

		ctx.Logger.Info("New device approved and locked to MAC", "id", data.DeviceID, "mac", data.Mac_Address)
	} else if dbMac != data.Mac_Address {
		ctx.Logger.Error("Hardware mismatch!", "id", data.DeviceID, "expected", dbMac, "got", data.Mac_Address)
		http.Error(w, "Forbidden: Invalid Hardware", http.StatusForbidden)

		return http.ErrAbortHandler
	}

	return nil
}
