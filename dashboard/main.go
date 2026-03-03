package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/smallstep/webhooks/pkg/db"
)

//go:embed templates/*
var templates embed.FS

//go:embed static/*
var static embed.FS

// DashboardHandler handles dashboard requests
type DashboardHandler struct {
	db *db.Database
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(database *db.Database) *DashboardHandler {
	return &DashboardHandler{db: database}
}

func main() {
	// Initialize database
	database, err := db.NewDatabase("scep.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	handler := NewDashboardHandler(database)

	// Setup API routes
	http.HandleFunc("/api/devices", handler.listDevicesHandler)
	http.HandleFunc("/api/register", handler.registerDeviceHandler)
	http.HandleFunc("/api/device/", handler.deleteDeviceHandler)
	http.HandleFunc("/api/device/rotate/", handler.rotatePassphraseHandler)
	http.HandleFunc("/api/config/diceware", handler.getDicewareConfigHandler)

	// Serve templates
	http.Handle("/", http.FileServer(http.FS(templates)))

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	log.Println("Dashboard listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// DeviceResponse represents a device in the API response
type DeviceResponse struct {
	SerialNumber string `json:"serial_number"`
	Hostname     string `json:"hostname"`
	CreatedAt    string `json:"created_at"`
}

// RegisterDeviceRequest represents a device registration request
type RegisterDeviceRequest struct {
	SerialNumber string `json:"serial_number"`
	Hostname     string `json:"hostname"`
	Passphrase   string `json:"passphrase"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// listDevicesHandler returns all registered devices
func (h *DashboardHandler) listDevicesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	devices, err := h.db.ListDevices()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Failed to list devices",
		})
		return
	}

	var result []DeviceResponse
	for _, device := range devices {
		result = append(result, DeviceResponse{
			SerialNumber: device.SerialNumber,
			Hostname:     device.Hostname,
			CreatedAt:    device.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// registerDeviceHandler registers a new device
func (h *DashboardHandler) registerDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Invalid request body",
		})
		return
	}

	if req.SerialNumber == "" || req.Hostname == "" || req.Passphrase == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Missing required fields",
		})
		return
	}

	err := h.db.RegisterDevice(req.SerialNumber, req.Hostname, req.Passphrase)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Failed to register device",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Message: "Device registered successfully",
	})
}

// deleteDeviceHandler removes a device
func (h *DashboardHandler) deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Extract serial number from URL
	_, serial := r.URL.Path[len("/api/device/"):]

	err := h.db.DeleteDevice(serial)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Failed to delete device",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Message: "Device deleted successfully",
	})
}

// rotatePassphraseHandler rotates a device's passphrase
func (h *DashboardHandler) rotatePassphraseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Extract serial number from URL
	_, serial := r.URL.Path[len("/api/device/rotate/"):]

	newPassphrase, err := h.db.RotatePassphrase(serial)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Failed to rotate passphrase",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Status:       "success",
		Message:      "Passphrase rotated successfully",
		NewPassphrase: newPassphrase,
	})
}

// getDicewareConfigHandler returns the diceware configuration
func (h *DashboardHandler) getDicewareConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Generate a random passphrase for demo purposes
	words := []string{
		"Queen", "Ate", "Lamp", "Wise", "Smell",
		"Blue", "Tree", "Sun", "Moon", "Star",
		"Cloud", "Rain", "Wind", "Fire", "Ice",
		"Stone", "Gold", "Silver", "Steel", "Iron",
	}

	var result string
	for i := 0; i < 5; i++ {
		result += words[randInt(len(words))] + "-"
	}
	result = result[:len(result)-1]

	config := map[string]interface{}{
		"passphrase": result,
		"expires_in": 300, // 5 minutes
		"generated_at": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// randInt returns a random index
func randInt(max int) int {
	return int(time.Now().UnixNano()) % max
}