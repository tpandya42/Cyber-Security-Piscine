package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func parseArgs(args []string) (string, string, error) {
	
	if len(args) != 2 {
		return "","", fmt.Errorf("invalid number of arguments")
	}
	
	var mode string
	var filename string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-g":
			mode = "g"
			filename = args[i + 1]
			return mode, filename, nil
		case "-k":
			mode = "k"
			filename = args[i + 1]
			return mode, filename, nil
		default:
			return "", "", fmt.Errorf("Invalid Input")
		}
	}
	return mode, filename, nil
}

func readKey(filename string) ([]byte, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	key := string(bytes)
	key = strings.TrimSpace(key)
	if len(key) < 64 {
		return nil, fmt.Errorf("less than 64 characters")
	}
	// Decode hexadecimal
	decoded, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	return decoded, nil
}

func encryptKey(secret []byte, encryptionKey []byte) ([]byte, error) {
	// AES-256 needs 32 bytes
	// block -> AES block cipher
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// GCM provides authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	// nonce -> is a value used once with a given encryption key, typically 12 bytes.
	nonce := make([]byte, gcm.NonceSize())
	// rand.Reader provides secure random data.
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// gcm.Seal (encrypts the secret)
	ciphertext := gcm.Seal(nil, nonce, secret, nil)
	
	// combine nonce and ciphertext
	// * The result contains encrypted secret and the authentication tag
	// * Nonce isn't secret, we store it with the ciphertext because decryption requires the same nonce.
	encrypted := append(nonce, ciphertext...)
	return encrypted, nil
}

func decryptKey(encrypted []byte, encryptionKey []byte) ([]byte, error) {

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// GCM provides authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	
	// extract the nonce (we stored it in beginning of the encrypted data)
	noncesize := gcm.NonceSize()

	if len(encrypted) < noncesize + gcm.Overhead() {
		return nil, fmt.Errorf("invalid data")
	}
	nonce := encrypted[:noncesize]
	ciphertext := encrypted[noncesize:]

	// decrypt
	secret, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt key: %w", err)
	}

	return secret, nil

}

func main(){
	secret := []byte("super secret TOTP key")
	encryptionKey := make([]byte, 32)
	encrypted, err := encryptKey(secret, encryptionKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Encryption failed:", err)
		os.Exit(1)
	}
	fmt.Printf("Original secret length: %d bytes\n", len(secret))
	fmt.Printf("Encrypted data length: %d bytes\n", len(encrypted))

	decrypted, err := decryptKey(encrypted, encryptionKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Decryption failed:", err)
		os.Exit(1)
	}

	if !bytes.Equal(secret, decrypted) {
		fmt.Fprintln(os.Stderr, "Test failed: secret don't match")
		os.Exit(1)
	}

	// testing tamper detection
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 1

	_, err = decryptKey(tampered, encryptionKey)
	if err == nil {
		fmt.Fprintln(os.Stderr, "Test failed: tampered data was accepted")
		os.Exit(1)
	}
	fmt.Println("tamper detection test passed\n")
}
