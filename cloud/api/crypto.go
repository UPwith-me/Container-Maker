package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// encryptCredentialData encrypts credential data using AES-256-GCM
func encryptCredentialData(data map[string]string, key string) (string, error) {
	// Serialize data to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Derive 32-byte key from secret using SHA-256
	keyHash := sha256.Sum256([]byte(key))

	// Create AES cipher
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to create nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptCredentialData decrypts credential data encrypted with AES-256-GCM
func decryptCredentialData(encryptedData string, key string) (map[string]string, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		// Try to parse as plain JSON for backward compatibility
		return parsePlainCredential(encryptedData)
	}

	// Derive key
	keyHash := sha256.Sum256([]byte(key))

	// Create AES cipher
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum length
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and decrypt
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Parse JSON
	var data map[string]string
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted data: %w", err)
	}

	return data, nil
}

// parsePlainCredential attempts to parse plain text credential (for backward compatibility)
func parsePlainCredential(data string) (map[string]string, error) {
	// Try JSON format first
	var result map[string]string
	if err := json.Unmarshal([]byte(data), &result); err == nil {
		return result, nil
	}

	// Try to parse Go map format: map[key:value key2:value2]
	// This is a simplified parser for backward compatibility
	result = make(map[string]string)

	// For now, return empty map if we can't parse
	// In production, we'd implement a proper parser or migrate old data
	return result, nil
}
