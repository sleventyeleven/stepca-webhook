package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// Device represents a registered device
type Device struct {
	SerialNumber string `json:"serial_number"`
	Hostname     string `json:"hostname"`
	Passphrase   string `json:"passphrase"` // Encrypted
	CreatedAt    string `json:"created_at"`
}

// Database wraps the SQLite database operations
type Database struct {
	mu      sync.RWMutex
	file    string
	conn    *sql.DB
}

// NewDatabase creates a new database instance
func NewDatabase(file string) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create devices table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS devices (
			serial_number TEXT PRIMARY KEY,
			hostname TEXT NOT NULL,
			passphrase TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create devices table: %v", err)
	}

	// Create challenges table (stores hex-encoded challenges)
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS challenges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			serial_number TEXT NOT NULL,
			encrypted_challenge TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (serial_number) REFERENCES devices(serial_number)
		)
	`); err != nil {
		return nil, fmt.Errorf("failed to create challenges table: %v", err)
	}

	return &Database{
		file: file,
		conn: db,
	}, nil
}

// RegisterDevice registers a new device with a diceware passphrase
func (db *Database) RegisterDevice(serial, hostname, passphrase string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	passphraseHash, err := EncryptPassphrase(passphrase, serial)
	if err != nil {
		return fmt.Errorf("failed to encrypt passphrase: %v", err)
	}

	_, err = db.conn.Exec(`
		INSERT INTO devices (serial_number, hostname, passphrase)
		VALUES (?, ?, ?)
		ON CONFLICT(serial_number) DO UPDATE SET
			hostname = excluded.hostname,
			passphrase = excluded.passphrase,
			updated_at = CURRENT_TIMESTAMP
	`, serial, hostname, passphraseHash)

	if err != nil {
		return fmt.Errorf("failed to register device: %v", err)
	}

	return nil
}

// GetDevice retrieves a device by serial number
func (db *Database) GetDevice(serial string) (*Device, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var device Device
	var passphraseHash string

	err := db.conn.QueryRow(`
		SELECT serial_number, hostname, passphrase FROM devices WHERE serial_number = ?
	`, serial).Scan(&device.SerialNumber, &device.Hostname, &passphraseHash)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found: %s", serial)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query device: %v", err)
	}

	device.Passphrase = passphraseHash
	return &device, nil
}

// GetDeviceByHostname retrieves a device by hostname
func (db *Database) GetDeviceByHostname(hostname string) (*Device, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var device Device
	var passphraseHash string

	err := db.conn.QueryRow(`
		SELECT serial_number, hostname, passphrase FROM devices WHERE hostname = ?
	`, hostname).Scan(&device.SerialNumber, &device.Hostname, &passphraseHash)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found for hostname: %s", hostname)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query device by hostname: %v", err)
	}

	device.Passphrase = passphraseHash
	return &device, nil
}

// ListDevices retrieves all devices
func (db *Database) ListDevices() ([]*Device, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.conn.Query(`
		SELECT serial_number, hostname, passphrase FROM devices
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %v", err)
	}
	defer rows.Close()

	var devices []*Device
	for rows.Next() {
		var device Device
		var passphraseHash string

		if err := rows.Scan(&device.SerialNumber, &device.Hostname, &passphraseHash); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		device.Passphrase = passphraseHash
		devices = append(devices, &device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return devices, nil
}

// DeleteDevice removes a device from the database
func (db *Database) DeleteDevice(serial string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	result, err := db.conn.Exec(`
		DELETE FROM devices WHERE serial_number = ?
	`, serial)
	if err != nil {
		return fmt.Errorf("failed to delete device: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rows == 0 {
		return fmt.Errorf("device not found: %s", serial)
	}

	return nil
}

// RotatePassphrase generates a new diceware passphrase for a device
func (db *Database) RotatePassphrase(serial string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Get current device to extract hostname
	device, err := db.GetDevice(serial)
	if err != nil {
		return "", err
	}

	// Generate new passphrase (simulated - in production use proper diceware)
	// Using a simple passphrase generator for demo
	newPassphrase := generateDicewarePassphrase()

	// Encrypt new passphrase
	passphraseHash, err := EncryptPassphrase(newPassphrase, serial)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt new passphrase: %v", err)
	}

	// Update database
	_, err = db.conn.Exec(`
		UPDATE devices SET passphrase = ?, updated_at = CURRENT_TIMESTAMP
		WHERE serial_number = ?
	`, passphraseHash, serial)

	if err != nil {
		return "", fmt.Errorf("failed to update passphrase: %v", err)
	}

	// Update challenge in challenges table
	_, err = db.conn.Exec(`
		UPDATE challenges SET encrypted_challenge = ?
		WHERE serial_number = ?
	`, passphraseHash, serial)

	if err != nil {
		return "", fmt.Errorf("failed to update challenge: %v", err)
	}

	return newPassphrase, nil
}

// EncryptPassphrase encrypts a passphrase using AES-128 with the serial number as key
func EncryptPassphrase(passphrase, serial string) (string, error) {
	// We use the existing EncryptSCEPChallenge function
	// The database stores the challenge key (encrypted passphrase) as hex or base64
	return EncryptSCEPChallenge(passphrase, serial)
}

// generateDicewarePassphrase generates a random passphrase
// In production, use a proper diceware word list
func generateDicewarePassphrase() string {
	// Simple passphrase generator for demo
	// This should be replaced with proper diceware word list in production
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
	return result[:len(result)-1]
}

// randInt returns a random index
func randInt(max int) int {
	return int(randInt64()) % max
}

// randInt64 returns a random int64
func randInt64() int64 {
	const letterBytes = "0123456789"
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	n := 0
	for _, c := range b {
		n = n*10 + int(c)
	}
	return n
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}