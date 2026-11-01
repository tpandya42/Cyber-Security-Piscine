package main

import (
	//	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func parseArgs(args []string) (string, string, error) {

	if len(args) != 2 {
		return "", "", fmt.Errorf("invalid number of arguments")
	}

	var mode string
	var filename string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-g":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("missing filename for -g")
			}
			mode = "g"
			filename = args[i+1]
			return mode, filename, nil
		case "-k":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("missing filename for -k")
			}
			mode = "k"
			filename = args[i+1]
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
	if len(key) != 64 {
		return nil, fmt.Errorf("key must be 64 hexadecimal characters.")
	}
	// Decode hexadecimal
	decoded, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	return decoded, nil
}

func saveEncryptedKey(encrypted []byte) error {
	err := os.WriteFile("ft_otp.key", encrypted, 0600)
	if err != nil {
		return fmt.Errorf("save encrypted key: %w", err)
	}
	return nil
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

// for -g implementation
func generateKeyFile(filename string) error {
	// step 1: read original hexadecimal key
	secret, err := readKey(filename)
	if err != nil {
		return err
	}

	// temporary encryption TODO: replace with persistent management later
	encryptionKey := make([]byte, 32)

	// step 2: encrypt the secret
	encrypted, err := encryptKey(secret, encryptionKey)
	if err != nil {
		return err
	}

	// step 3: save encrypted secret
	err = saveEncryptedKey(encrypted)
	if err != nil {
		return err
	}
	fmt.Println("Key was successfully saved in ft_otp.key.")
	return nil

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

	if len(encrypted) < noncesize+gcm.Overhead() {
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

/*
Implementing HTOP (RFC 4226)
It uses : HMAC-SHA1

	8-byte counter
	Dynamic trunation
	modulo 1,000,000 to produce six digits
*/
func hotp(secret []byte, counter uint64) (string, error) {
	// step 1: convert counter into 8 byte big-endian representation
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)

	// step 2: calculate HMAC-SHA1
	mac := hmac.New(sha1.New, secret)
	mac.Write(counterBytes[:])
	hash := mac.Sum(nil)

	// step 3: Dynamic Truncation
	offset := hash[len(hash)-1] & 0x0f

	binaryCode := binary.BigEndian.Uint32(hash[offset : offset+4])
	binaryCode &= 0x7fffffff
	// step 4: generate a six digit OTP
	otp := binaryCode % 1000000

	return fmt.Sprintf("%06d", otp), nil

}

func totp(secret []byte) (string, error) {
	timestamp := time.Now().Unix()
	counter := uint64(timestamp / 30) // TOTP time step is 30 seconds
	return hotp(secret, counter)
}

func generateOTP(filename string) error {
	encrypted, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read encrypted key: %w", err)
	}

	encryptionKey := make([]byte, 32)
	secret, err := decryptKey(encrypted, encryptionKey)
	if err != nil {
		return err
	}
	otp, err := totp(secret)
	if err != nil {
		return err
	}
	fmt.Println(otp)
	return nil
}

func main() {
	mode, filename, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "./ft_otp: error: %s\n", err)
		os.Exit(1)
	}
	switch mode {
	case "g":
		err = generateKeyFile(filename)
	case "k":
		err = generateOTP(filename)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "./ft_otp: error: %s\n", err)
		os.Exit(1)
	}
}
