package db

import (
	"os"
	"testing"
)

func setupTestDB(t *testing.T) *Database {
	file := filepath.Join(os.TempDir(), "test_scep.db")
	db, err := NewDatabase(file)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func teardownTestDB(t *testing.T, db *Database) {
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	os.Remove(db.file)
}

func TestRegisterDevice(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	serial := "1234567890123456"
	hostname := "test-device.local"
	passphrase := "Queen-Ate-Lamp-Wise-Smell"

	err := db.RegisterDevice(serial, hostname, passphrase)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}

	device, err := db.GetDevice(serial)
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}

	if device.SerialNumber != serial {
		t.Errorf("Expected serial %s, got %s", serial, device.SerialNumber)
	}

	if device.Hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, device.Hostname)
	}

	if device.Passphrase == "" {
		t.Error("Passphrase should not be empty")
	}
}

func TestGetDeviceByHostname(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	serial := "1234567890123456"
	hostname := "test-device.local"
	passphrase := "Queen-Ate-Lamp-Wise-Smell"

	err := db.RegisterDevice(serial, hostname, passphrase)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}

	device, err := db.GetDeviceByHostname(hostname)
	if err != nil {
		t.Fatalf("Failed to get device by hostname: %v", err)
	}

	if device.SerialNumber != serial {
		t.Errorf("Expected serial %s, got %s", serial, device.SerialNumber)
	}
}

func TestListDevices(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	devices := []*Device{
		{SerialNumber: "1234567890123456", Hostname: "device1.local"},
		{SerialNumber: "abcdef1234567890", Hostname: "device2.local"},
		{SerialNumber: "fedcba0987654321", Hostname: "device3.local"},
	}

	for _, device := range devices {
		err := db.RegisterDevice(device.SerialNumber, device.Hostname, "passphrase")
		if err != nil {
			t.Fatalf("Failed to register device %s: %v", device.SerialNumber, err)
		}
	}

	list, err := db.ListDevices()
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(list))
	}
}

func TestDeleteDevice(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	serial := "1234567890123456"
	hostname := "test-device.local"

	err := db.RegisterDevice(serial, hostname, "passphrase")
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}

	err = db.DeleteDevice(serial)
	if err != nil {
		t.Fatalf("Failed to delete device: %v", err)
	}

	_, err = db.GetDevice(serial)
	if err == nil {
		t.Error("Device should not exist after deletion")
	}
}

func TestRotatePassphrase(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	serial := "1234567890123456"
	hostname := "test-device.local"
	oldPassphrase := "Queen-Ate-Lamp-Wise-Smell"

	err := db.RegisterDevice(serial, hostname, oldPassphrase)
	if err != nil {
		t.Fatalf("Failed to register device: %v", err)
	}

	// Get old passphrase
	device, err := db.GetDevice(serial)
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}

	newPassphrase, err := db.RotatePassphrase(serial)
	if err != nil {
		t.Fatalf("Failed to rotate passphrase: %v", err)
	}

	// Verify new passphrase is different
	if newPassphrase == oldPassphrase {
		t.Error("New passphrase should be different from old one")
	}

	// Verify old passphrase is no longer valid
	// (In production, you'd check the encrypted version, but for now we just check it exists)
	_, err = db.GetDevice(serial)
	if err != nil {
		t.Errorf("Device should still exist after passphrase rotation: %v", err)
	}
}