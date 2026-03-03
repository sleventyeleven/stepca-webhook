package scep

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// SCEP Challenge Validation and Encryption/Decryption

// EncryptSCEPChallenge encrypts a diceware passphrase using AES-128 with the serial number as key
// Formula: SCEPCHALLENGE = base64(AES128(passphrase, serial))
func EncryptSCEPChallenge(passphrase, serial string) (string, error) {
	if len(serial) != 16 {
		return "", fmt.Errorf("serial number must be exactly 16 bytes for AES-128")
	}

	// Convert passphrase to bytes
	passBytes := []byte(passphrase)

	// Create block
	block, err := aes.NewCipher([]byte(serial))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt
	encrypted := gcm.Seal(nonce, nonce, passBytes, nil)

	// Encode as base64
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptSCEPChallenge decrypts an encrypted SCEP challenge using AES-128 with the serial number as key
func DecryptSCEPChallenge(encrypted, serial string) (string, error) {
	if len(serial) != 16 {
		return "", fmt.Errorf("serial number must be exactly 16 bytes for AES-128")
	}

	// Decode base64
	encryptedBytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %v", err)
	}

	// Create block
	block, err := aes.NewCipher([]byte(serial))
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	// Check nonce size
	nonceSize := gcm.NonceSize()
	if len(encryptedBytes) < nonceSize {
		return "", fmt.Errorf("encrypted data too short")
	}

	// Extract nonce
	nonce, ciphertext := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %v", err)
	}

	return string(plaintext), nil
}

// ValidateSCEPChallenge validates if the provided challenge matches the stored encrypted value
func ValidateSCEPChallenge(encrypted, serial, providedChallenge string) bool {
	if serial == "" || encrypted == "" || providedChallenge == "" {
		return false
	}

	// Extract the expected challenge by decrypting
	expectedChallenge, err := DecryptSCEPChallenge(encrypted, serial)
	if err != nil {
		return false
	}

	// Compare challenges
	return expectedChallenge == providedChallenge
}

// GenerateChallengeForDevice generates an encrypted SCEP challenge for a device
func GenerateChallengeForDevice(passphrase, serial string) (string, error) {
	return EncryptSCEPChallenge(passphrase, serial)
}

// GetChallengeKeyFromHex extracts the challenge key from a hex-encoded value
// This is used when the challenge is stored as hex in the database
func GetChallengeKeyFromHex(hexStr, serial string) (string, error) {
	// Try to decode hex
	decrypted, err := DecryptSCEPChallenge(hexStr, serial)
	if err != nil {
		// If hex decode fails, try as base64
		return DecryptSCEPChallenge(hexStr, serial)
	}
	return decrypted, nil
}