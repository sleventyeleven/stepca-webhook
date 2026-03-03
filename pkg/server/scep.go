package server

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/smallstep/certificates/webhook"
	"github.com/smallstep/webhooks/pkg/db"
	"github.com/smallstep/webhooks/pkg/scep"
)

// SCEPChallengeHandler handles SCEP challenge validation
type SCEPChallengeHandler struct {
	db *db.Database
}

// NewSCEPChallengeHandler creates a new SCEP challenge handler
func NewSCEPChallengeHandler(database *db.Database) *SCEPChallengeHandler {
	return &SCEPChallengeHandler{db: database}
}

// AuthorizeSCEP handles SCEP challenge authorization
// URL format: /scep/challenge/{serial_number}
func (h *SCEPChallengeHandler) AuthorizeSCEP(w http.ResponseWriter, r *http.Request) {
	wrb, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	// Extract serial number from URL path
	_, serial := path.Split(r.URL.Path)

	// Validate serial format
	if len(serial) != 16 {
		http.Error(w, "Invalid serial number format", http.StatusBadRequest)
		return
	}

	// Get the device from database
	device, err := h.db.GetDevice(serial)
	if err != nil {
		log.Printf("Device not found: %s, error: %v", serial, err)
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Extract challenge from the X509CertificateRequest
	csr := wrb.X509CertificateRequest
	if csr == nil {
		http.Error(w, "Missing X509 certificate request", http.StatusBadRequest)
		return
	}

	// Extract SCEP challenge from CSR attributes
	challenge, ok := extractSCEPChallenge(csr)
	if !ok {
		log.Printf("No SCEP challenge found in CSR for serial %s", serial)
		http.Error(w, "Missing SCEP challenge", http.StatusBadRequest)
		return
	}

	// Validate the challenge
	allow := scep.ValidateSCEPChallenge(device.Passphrase, serial, challenge)

	err = json.NewEncoder(w).Encode(webhook.ResponseBody{Allow: allow})
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	log.Printf("SCEP challenge validation for %s: allow=%t, challenge=%s", serial, allow, challenge)
}

// extractSCEPChallenge extracts the SCEP challenge from the CSR attributes
func extractSCEPChallenge(csr *webhook.X509CertificateRequest) (string, bool) {
	// SCEP challenge is typically in the Subject Alternative Name extension
	// as a challenge password (OID 1.3.6.1.5.5.7.8.4)

	if csr.Attributes == nil {
		return "", false
	}

	for _, attr := range csr.Attributes {
		if attr.Type == "1.3.6.1.5.5.7.8.4" {
			if challenge, ok := attr.Value.(string); ok {
				return challenge, true
			}
		}
	}

	// Alternative: check if challenge is in CSR fields
	// This depends on how the CSR is constructed
	if csr.Raw != nil {
		// In production, you might need to parse the CSR bytes to extract challenge
		// For now, return false as challenge is typically in the attributes
	}

	return "", false
}

// RegisterDeviceHandler handles registration of new devices
// URL format: /scep/register
func (h *SCEPChallengeHandler) RegisterDeviceHandler(w http.ResponseWriter, r *http.Request) {
	wrb, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var req struct {
		SerialNumber string `json:"serial_number"`
		Hostname     string `json:"hostname"`
		Passphrase   string `json:"passphrase"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println(err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SerialNumber == "" || req.Hostname == "" || req.Passphrase == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if len(req.SerialNumber) != 16 {
		http.Error(w, "Serial number must be exactly 16 characters", http.StatusBadRequest)
		return
	}

	err := h.db.RegisterDevice(req.SerialNumber, req.Hostname, req.Passphrase)
	if err != nil {
		log.Printf("Failed to register device: %v", err)
		http.Error(w, "Failed to register device", http.StatusInternalServerError)
		return
	}

	log.Printf("Device registered: serial=%s, hostname=%s", req.SerialNumber, req.Hostname)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Device registered successfully",
	})
}

// GetChallengeHandler generates and returns the SCEP challenge for a device
// URL format: /scep/challenge/{serial_number}
func (h *SCEPChallengeHandler) GetChallengeHandler(w http.ResponseWriter, r *http.Request) {
	wrb, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	_, serial := path.Split(r.URL.Path)

	if len(serial) != 16 {
		http.Error(w, "Invalid serial number format", http.StatusBadRequest)
		return
	}

	device, err := h.db.GetDevice(serial)
	if err != nil {
		log.Printf("Device not found: %s, error: %v", serial, err)
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Generate the encrypted challenge
	encryptedChallenge, err := scep.EncryptSCEPChallenge(device.Passphrase, serial)
	if err != nil {
		log.Printf("Failed to encrypt challenge: %v", err)
		http.Error(w, "Failed to generate challenge", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"encrypted_challenge": encryptedChallenge,
	})
}

// ListDevicesHandler lists all registered devices
func (h *SCEPChallengeHandler) ListDevicesHandler(w http.ResponseWriter, r *http.Request) {
	wrb, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	devices, err := h.db.ListDevices()
	if err != nil {
		log.Printf("Failed to list devices: %v", err)
		http.Error(w, "Failed to list devices", http.StatusInternalServerError)
		return
	}

	// Return sanitized device data (remove passphrase)
	type DeviceInfo struct {
		SerialNumber string `json:"serial_number"`
		Hostname     string `json:"hostname"`
		CreatedAt    string `json:"created_at"`
	}

	var result []DeviceInfo
	for _, device := range devices {
		result = append(result, DeviceInfo{
			SerialNumber: device.SerialNumber,
			Hostname:     device.Hostname,
			CreatedAt:    device.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeleteDeviceHandler removes a device from the database
func (h *SCEPChallengeHandler) DeleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	wrb, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	_, serial := path.Split(r.URL.Path)

	if len(serial) != 16 {
		http.Error(w, "Invalid serial number format", http.StatusBadRequest)
		return
	}

	err := h.db.DeleteDevice(serial)
	if err != nil {
		log.Printf("Failed to delete device: %v", err)
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	log.Printf("Device deleted: %s", serial)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Device deleted successfully",
	})
}

// RotatePassphraseHandler rotates the passphrase for a device
func (h *SCEPChallengeHandler) RotatePassphraseHandler(w http.ResponseWriter, r *http.Request) {
	wrb, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	_, serial := path.Split(r.URL.Path)

	if len(serial) != 16 {
		http.Error(w, "Invalid serial number format", http.StatusBadRequest)
		return
	}

	newPassphrase, err := h.db.RotatePassphrase(serial)
	if err != nil {
		log.Printf("Failed to rotate passphrase: %v", err)
		http.Error(w, "Failed to rotate passphrase", http.StatusInternalServerError)
		return
	}

	log.Printf("Passphrase rotated for device: %s", serial)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "success",
		"message":      "Passphrase rotated successfully",
		"new_passphrase": newPassphrase,
	})
}

// authenticate performs webhook authentication
func (h *SCEPChallengeHandler) authenticate(w http.ResponseWriter, r *http.Request) (*webhook.RequestBody, bool) {
	id := r.Header.Get("X-Smallstep-Webhook-ID")
	if id == "" {
		http.Error(w, "Missing X-Smallstep-Webhook-ID header", http.StatusBadRequest)
		return nil, false
	}

	// For SCEP handlers, we'll use a simpler authentication
	// In production, use the same authentication as in server.go
	if _, ok := h.webhookIDsToSecrets[id]; !ok {
		log.Printf("Missing webhook secret for %s", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil, false
	}

	return h.authenticateRequest(r)
}

// webhookIDsToSecrets maps webhook IDs to authentication secrets
var webhookIDsToSecrets = map[string]Secret{}

// authenticateRequest performs the actual authentication logic
func (h *SCEPChallengeHandler) authenticateRequest(r *http.Request) (*webhook.RequestBody, bool) {
	// This is a simplified version - use the authenticate function from server.go
	// for production use
	return nil, false
}

// SetupSCEPHandlers sets up all SCEP-related HTTP handlers
func SetupSCEPHandlers(s *http.Server, database *db.Database) {
	handler := NewSCEPChallengeHandler(database)

	// Register endpoints
	http.HandleFunc("/scep/challenge/", handler.AuthorizeSCEP)
	http.HandleFunc("/scep/register", handler.RegisterDeviceHandler)
	http.HandleFunc("/scep/challenge/get/", handler.GetChallengeHandler)
	http.HandleFunc("/scep/devices", handler.ListDevicesHandler)
	http.HandleFunc("/scep/device/", handler.DeleteDeviceHandler)
	http.HandleFunc("/scep/device/rotate/", handler.RotatePassphraseHandler)

	log.Println("SCEP challenge handlers registered")
}