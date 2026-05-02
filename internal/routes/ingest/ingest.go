package ingest

import (
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
