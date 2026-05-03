package dashboard

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/notebook-iot/cloud/internal/context"
)

func CreateAndExportKey(w http.ResponseWriter, r *http.Request, ctx *context.Context) {
	// 1. Generate Unique API Key and Device ID
	rawKey := hex.EncodeToString(generateBytes(16))
	deviceID := "iot-dev-" + hex.EncodeToString(generateBytes(4))

	// 2. Hash for Database (Security)
	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := hex.EncodeToString(hash[:])

	// 3. Save to Database with 'pending' status
	_, err := ctx.DB.Exec("INSERT INTO devices (device_id, hashed_api_key, status) VALUES ($1, $2, 'pending')",
		deviceID, hashedKey)
	if err != nil {
		ctx.Logger.Error("DB Insert failed", "err", err)
		return
	}

	// 4. Generate the CSV string in the required NVS format
	csvContent := strings.Builder{}
	csvContent.WriteString("key,type,encoding,value\n")
	csvContent.WriteString("secrets,namespace,,\n")
	csvContent.WriteString("wifi_ssid,data,string,YourSSID\n")
	csvContent.WriteString("wifi_pass,data,string,YourPassword\n")
	csvContent.WriteString(fmt.Sprintf("api_key,data,string,%s\n", rawKey))
	csvContent.WriteString(fmt.Sprintf("device_id,data,string,%s\n", deviceID))

	// 5. Serve as a downloadable file
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", deviceID))
	w.Header().Set("Content-Type", "text/csv")
	w.Write([]byte(csvContent.String()))
}

func generateBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}
