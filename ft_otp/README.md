# ft_otp

A One-Time Password (OTP) generator written in Go, implementing HOTP (RFC 4226) and TOTP (RFC 6238) standards. Supports key generation/encryption and OTP production from an encrypted key file.

## Table of Contents

- [Overview](#overview)
- [Theory](#theory)
- [Features](#features)
- [Project Structure](#project-structure)
- [Architecture & In-Depth Explanation](#architecture--in-depth-explanation)
- [Installation](#installation)
- [Usage](#usage)
- [Examples](#examples)
- [Testing](#testing)
- [What We Learned](#what-we-learned)
- [Edge Cases & Limitations](#edge-cases--limitations)
- [License](#license)

## Overview

`ft_otp` is a command-line tool that implements time-based and counter-based one-time password algorithms. It operates in two modes:

1. **`-g` (generate)** — reads a hexadecimal secret key from a file, encrypts it with AES-256-GCM, and stores the encrypted key in `ft_otp.key`.
2. **`-k` (key/OTP)** — reads the encrypted key file, decrypts the secret, and generates a 6-digit OTP using the TOTP algorithm.

The project was built as part of the Cybersecurity Piscine at 42 Berlin, introducing symmetric encryption, HMAC-based authentication, and time-synchronized one-time passwords.

## Theory

### One-Time Passwords (OTP)

An OTP is a password that is valid for only one login session or transaction. Unlike static passwords, OTPs expire quickly, making them resistant to replay attacks.

### HOTP — HMAC-based One-Time Password (RFC 4226)

HOTP generates an OTP from a shared secret key and a moving factor (a counter):

1. Compute `HMAC-SHA1(secret, counter)` where counter is an 8-byte big-endian integer.
2. Apply **dynamic truncation** to the 20-byte SHA-1 hash to extract a 31-bit integer.
3. Take the result modulo 1,000,000 to produce a 6-digit code.

### TOTP — Time-based One-Time Password (RFC 6238)

TOTP extends HOTP by replacing the counter with a time-based value:

```
counter = floor(current_unix_timestamp / time_step)
```

The default time step is **30 seconds**, meaning the OTP changes every 30 seconds. Both client and server must be roughly time-synchronized.

### AES-256-GCM Encryption

The secret key is encrypted at rest using AES-256 in Galois/Counter Mode (GCM):

- **AES-256**: Symmetric block cipher with a 256-bit (32-byte) key.
- **GCM**: Authenticated encryption mode that provides both confidentiality and integrity. It produces a ciphertext plus an authentication tag.
- **Nonce**: A 12-byte value used once per encryption. It is not secret but must never repeat with the same key. It is prepended to the ciphertext so decryption can recover it.

### HMAC-SHA1

HMAC (Hash-based Message Authentication Code) combines a cryptographic hash (SHA-1) with a secret key to produce a message authentication code. In HOTP, it binds the counter to the secret so that only someone with the key can generate valid codes.

## Features

- **`-g <keyfile>`**: Read a hex key (min 64 chars), encrypt with AES-256-GCM, save to `ft_otp.key`
- **`-k <keyfile>`**: Decrypt `ft_otp.key`, compute TOTP, print 6-digit OTP
- **HOTP implementation** per RFC 4226 with dynamic truncation
- **TOTP implementation** per RFC 6238 with 30-second time steps
- **Unit tests** with RFC 4226 test vectors
- Single-file implementation (`main.go`)
- Zero external dependencies (Go standard library only)

## Project Structure

```
ft_otp/
├── main.go        # Single-file source: CLI, key management, AES-GCM, HOTP/TOTP
├── main_test.go   # Unit tests for HOTP (RFC 4226 test vectors)
├── go.mod         # Module definition (module ft_otp)
├── go.sum         # Dependency checksums
└── README.md      # This file
```

## Architecture & In-Depth Explanation

### `parseArgs`

Parses exactly two command-line arguments:

| Flag | Mode | Description |
|------|------|-------------|
| `-g <file>` | `"g"` | Generate/encrypt key from `<file>` |
| `-k <file>` | `"k"` | Generate OTP from encrypted key file |

Any other input returns an error.

### `readKey`

Reads a key file and validates it:

1. Reads raw bytes with `os.ReadFile`
2. Trims whitespace
3. Requires at least 64 characters
4. Decodes the hex string into raw bytes via `hex.DecodeString`

Returns the decoded secret key bytes.

### `encryptKey`

Encrypts the secret using AES-256-GCM:

1. Creates an AES cipher block from a 32-byte encryption key
2. Wraps it in GCM mode
3. Generates a random 12-byte nonce from `crypto/rand`
4. Seals (encrypts) the secret: `ciphertext = gcm.Seal(nil, nonce, secret, nil)`
5. Prepends the nonce to the ciphertext for storage

Returns `nonce || ciphertext`.

### `saveEncryptedKey`

Writes the encrypted blob to `ft_otp.key` with file permissions `0600` (owner read/write only).

### `generateKeyFile` (`-g` mode)

Orchestrates key generation:

1. `readKey(filename)` — load and validate the hex secret
2. `encryptKey(secret, encryptionKey)` — AES-256-GCM encrypt
3. `saveEncryptedKey(encrypted)` — persist to `ft_otp.key`

> **Note**: The encryption key is currently a 32-byte zero value (`make([]byte, 32)`), marked as temporary in the code. For production, this should be derived from a user-supplied passphrase or stored separately.

### `decryptKey`

Reverses `encryptKey`:

1. Recreates the AES-GCM block from the encryption key
2. Splits the input into nonce (first 12 bytes) and ciphertext (remainder)
3. Validates minimum length: `noncesize + gcm.Overhead()`
4. Calls `gcm.Open` to decrypt and authenticate

Returns the original secret bytes on success.

### `generateOTP` (`-k` mode)

Orchestrates OTP generation:

1. Reads `ft_otp.key` from disk
2. `decryptKey(encrypted, encryptionKey)` — recover the secret
3. `totp(secret)` — compute the current TOTP
4. Prints the 6-digit OTP to stdout

### `hotp` (RFC 4226)

```go
func hotp(secret []byte, counter uint64) (string, error)
```

1. Packs `counter` into 8 bytes, big-endian
2. Computes `HMAC-SHA1(secret, counterBytes)`
3. Dynamic truncation:
   - `offset = last_byte & 0x0f`
   - Extract 4 bytes starting at `offset`
   - Clear the most significant bit: `& 0x7fffffff`
4. `otp = binaryCode % 1000000`
5. Returns zero-padded 6-digit string

### `totp` (RFC 6238)

```go
func totp(secret []byte) (string, error)
```

1. Gets current Unix timestamp
2. `counter = timestamp / 30` (30-second time step)
3. Delegates to `hotp(secret, counter)`

### `main`

1. Parses args into `(mode, filename)`
2. Switch on mode:
   - `"g"` → `generateKeyFile(filename)`
   - `"k"` → `generateOTP(filename)`
3. On error, prints to stderr and exits with code 1

## Installation

### Prerequisites

- Go 1.27.3 or later
- No external dependencies (standard library only)

### Build

```bash
cd ft_otp
go build -o ft_otp .
```

### Run from source

```bash
go run . -g keyfile.txt
go run . -k ft_otp.key
```

## Usage

```bash
./ft_otp -g <keyfile>   # Generate encrypted key
./ft_otp -k <keyfile>   # Generate OTP from encrypted key
```

| Flag | Argument | Description |
|------|----------|-------------|
| `-g` | keyfile | Path to a file containing a hex key (min 64 chars) |
| `-k` | keyfile | Path to the encrypted key file (`ft_otp.key`) |

## Examples

### Step 1: Create a hex key file

```bash
echo "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2" > mykey.txt
```

The key must be at least 64 hexadecimal characters (32 bytes when decoded).

### Step 2: Generate the encrypted key file

```bash
./ft_otp -g mykey.txt
# Key was successfully save in ft_otp.key
```

This creates `ft_otp.key` containing the AES-256-GCM encrypted secret (nonce + ciphertext + auth tag).

### Step 3: Generate an OTP

```bash
./ft_otp -k ft_otp.key
# 755224
```

Prints a 6-digit time-based OTP. The code changes every 30 seconds.

### Invalid usage

```bash
./ft_otp
# Error: invalid number of arguments

./ft_otp -x mykey.txt
# Error: Invalid Input

./ft_otp -g shortkey.txt
# Error: less than 64 characters
```

## Testing

### Run unit tests

```bash
cd ft_otp
go test -v -run TestHOTP
```

The test `TestHOTP` validates the `hotp` function against RFC 4226 test vectors:

| Counter | Expected OTP |
|---------|-------------|
| 0 | 755224 |
| 1 | 287082 |
| 2 | 359152 |
| 3 | 969429 |
| 4 | 338314 |
| 5 | 254676 |
| 6 | 287922 |
| 7 | 162583 |
| 8 | 399871 |
| 9 | 520489 |

### Full test suite

```bash
go test ./...
```

### Manual end-to-end test

```bash
# Create key file
echo "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" > testkey.txt

# Generate encrypted key
./ft_otp -g testkey.txt

# Generate OTP
./ft_otp -k ft_otp.key

# Verify output is 6 digits
./ft_otp -k ft_otp.key | grep -E '^[0-9]{6}$'
```

## What We Learned

- **Symmetric encryption**: AES-256 block cipher, GCM authenticated encryption mode, nonce management
- **Key derivation basics**: Reading hex keys, validating minimum length, decoding to bytes
- **HMAC**: `crypto/hmac` with SHA-1 for message authentication
- **HOTP algorithm**: Counter-based OTP with dynamic truncation per RFC 4226
- **TOTP algorithm**: Time-based counter (Unix timestamp / 30 seconds) per RFC 6238
- **Binary encoding**: `encoding/binary` for big-endian uint64/uint32 conversion
- **File permissions**: `os.WriteFile` with `0600` for sensitive key material
- **Secure random**: `crypto/rand` for nonce generation
- **Go testing**: Table-driven test with RFC standard test vectors
- **CLI design**: Manual argument parsing with mode-based dispatch

## Edge Cases & Limitations

| Issue | Behavior |
|-------|----------|
| Wrong argument count | Error: "invalid number of arguments" |
| Unknown flag | Error: "Invalid Input" |
| Key file < 64 chars | Error: "less than 64 characters" |
| Non-hex key content | Error from `hex.DecodeString` |
| Missing `ft_otp.key` | Error reading file |
| Wrong encryption key | `gcm.Open` authentication failure |
| Clock skew | TOTP depends on system time; mismatched clocks produce invalid codes |
| Encryption key is hardcoded zeros | Currently `make([]byte, 32)` — not derived from user input; should be replaced with proper key management in production |
| Single OTP output | Prints to stdout only; no QR code or provisioning URI support |
