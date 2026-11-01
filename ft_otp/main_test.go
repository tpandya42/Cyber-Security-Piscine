package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHOTP(t *testing.T) {
	secret := []byte("12345678901234567890")

	expected := []string{
		"755224",
		"287082",
		"359152",
		"969429",
		"338314",
		"254676",
		"287922",
		"162583",
		"399871",
		"520489",
	}
	for counter, want := range expected {
		got, err := hotp(secret, uint64(counter))

		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf(
				"counter %d: expected %s, got %s",
				counter, want, got,
			)
		}
	}
}

func TestHOTPFormat(t *testing.T) {
	secret := []byte("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	for counter := uint64(0); counter < 100; counter++ {
		got, err := hotp(secret, counter)
		if err != nil {
			t.Fatalf("counter %d: unexpected error: %v", counter, err)
		}
		if len(got) != 6 {
			t.Errorf("counter %d: expected 6 digits, got %q (len %d)", counter, got, len(got))
		}
		for _, c := range got {
			if c < '0' || c > '9' {
				t.Errorf("counter %d: expected digit, got %q", counter, got)
				break
			}
		}
	}
}

func TestTOTPFormat(t *testing.T) {
	secret := []byte("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	got, err := totp(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 6 {
		t.Errorf("expected 6 digits, got %q (len %d)", got, len(got))
	}
	for _, c := range got {
		if c < '0' || c > '9' {
			t.Errorf("expected digit, got %q", got)
			break
		}
	}
}

func TestParseArgsValid(t *testing.T) {
	mode, filename, err := parseArgs([]string{"-g", "key.hex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "g" || filename != "key.hex" {
		t.Errorf("expected (g, key.hex), got (%q, %q)", mode, filename)
	}

	mode, filename, err = parseArgs([]string{"-k", "ft_otp.key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "k" || filename != "ft_otp.key" {
		t.Errorf("expected (k, ft_otp.key), got (%q, %q)", mode, filename)
	}
}

func TestParseArgsInvalid(t *testing.T) {
	_, _, err := parseArgs([]string{"-x", "file"})
	if err == nil {
		t.Error("expected error for invalid flag, got nil")
	}

	_, _, err = parseArgs([]string{})
	if err == nil {
		t.Error("expected error for no args, got nil")
	}

	_, _, err = parseArgs([]string{"-g"})
	if err == nil {
		t.Error("expected error for missing filename after -g, got nil")
	}

	_, _, err = parseArgs([]string{"-k"})
	if err == nil {
		t.Error("expected error for missing filename after -k, got nil")
	}
}

func TestReadKeyValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.hex")
	if err := os.WriteFile(path, []byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	decoded, err := readKey(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(decoded))
	}
}

func TestReadKeyTooShort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short.hex")
	if err := os.WriteFile(path, []byte("0123456789abcdef"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := readKey(path)
	if err == nil {
		t.Error("expected error for short key, got nil")
	}
	if !strings.Contains(err.Error(), "64 hexadecimal characters") {
		t.Errorf("expected '64 hexadecimal characters' message, got %q", err.Error())
	}
}

func TestReadKeyTooLong(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.hex")
	if err := os.WriteFile(path, []byte(strings.Repeat("0", 66)), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := readKey(path)
	if err == nil {
		t.Error("expected error for long key, got nil")
	}
}

func TestReadKeyNonHex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonhex.hex")
	if err := os.WriteFile(path, []byte("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := readKey(path)
	if err == nil {
		t.Error("expected error for non-hex key, got nil")
	}
}
