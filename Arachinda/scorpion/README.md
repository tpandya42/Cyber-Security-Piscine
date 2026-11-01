# Arachnida — Exercise 2: Scorpion

An image metadata analyzer written in Go. Given one or more image files, it
displays filesystem attributes, image dimensions, and EXIF metadata (camera make,
model, capture date, etc.).

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

`scorpion` is a command-line metadata extractor for image files. It accepts one or
more file paths, validates supported extensions, and prints for each file:

- File size and modification time
- Image dimensions and format
- EXIF tags (Make, Model, DateTime, DateTimeOriginal, DateTimeDigitized, and more)

It was built as part of the Cybersecurity Piscine "Arachnida" project at 42 Berlin,
which introduces web scraping and metadata handling. Metadata is "data about data"
— information embedded in files that can reveal details about their creation,
author, device, and more.

## Theory

### What is EXIF?

EXIF (Exchangeable Image File Format) is a standard that specifies formats for
images, sound, and ancillary tags used by digital cameras, scanners, and other
systems. EXIF data is embedded directly inside image files (commonly in JPEG and
TIFF) and includes:

- **Camera information**: Make, Model, lens details
- **Capture information**: DateTimeOriginal (when the photo was taken)
- **Camera settings**: Exposure time, f-number, ISO speed
- **Geolocation**: GPS latitude/longitude (if enabled on the device)

### Why metadata matters in cybersecurity

Metadata can reveal sensitive information:

- A photo's `DateTimeOriginal` can establish a timeline
- `Make`/`Model` can identify the device used
- GPS coordinates can reveal physical locations
- Software versions in metadata can indicate unpatched tools

Attackers and defenders alike use metadata analysis during reconnaissance and
forensics.

### Supported Formats

| Format | EXIF Support | Notes |
|--------|-------------|-------|
| JPEG / JPG | Full | Most common carrier of EXIF |
| PNG | Limited | Some PNGs store EXIF-like chunks |
| GIF | Minimal | Usually no EXIF |
| BMP | Minimal | Usually no EXIF |

If a file has no EXIF data, scorpion reports "(no EXIF data found)" instead of
erroring.

### Dependencies

- `github.com/bep/imagemeta` — EXIF decoding for multiple image formats
- Go standard library: `image`, `os`, `path/filepath`, `strings`, `time`, `fmt`

## Features

- Accepts multiple files: `scorpion FILE1 [FILE2 ...]`
- Extension filter: `.jpg/.jpeg/.png/.gif/.bmp`
- Filesystem metadata: size, modification time
- Image metadata: dimensions (W×H), format
- EXIF metadata via `github.com/bep/imagemeta`
- Ordered EXIF output (common tags first, then remaining tags)
- Graceful error handling: one bad file does not abort the batch
- Single-file implementation (`main.go` only)

## Project Structure

```
scorpion/
├── main.go      # Single-file source: CLI, metadata extraction, display
├── go.mod       # Module definition (module scorpion)
└── go.sum       # Dependency checksums
```

## Architecture & In-Depth Explanation

### `isSupported`

Validates that a file path ends with one of the supported extensions (case
insensitive). Unsupported files are skipped with a stderr warning.

### `imageFormatFromExt`

Maps a file extension to an `imagemeta.ImageFormat` constant so the EXIF decoder
knows how to interpret the file:

- `.jpg`/`.jpeg` → `imagemeta.JPEG`
- `.png` → `imagemeta.PNG`
- `.gif` → `imagemeta.GIF`
- `.bmp` → `imagemeta.TIFF` (fallback)

### `processFile`

The core per-file processor. Steps:

1. **Open file** — `os.Open(path)`, deferred `f.Close()`
2. **Filesystem stat** — `f.Stat()` → size and modification time
3. **Print header** — path, size, mod time (RFC3339 format)
4. **Decode image config** — `image.DecodeConfig(f)` reads only the header to get
   width, height, and format without decoding pixels
5. **Rewind** — `f.Seek(0, 0)` to reset the reader for EXIF parsing
6. **Decode EXIF** — `imagemeta.Decode` with a `HandleTag` callback that
   collects EXIF tags into `exifTags map[string]string`
7. **Print EXIF** — common tags (`Make`, `Model`, `DateTime`, etc.) first, then
   any remaining tags alphabetically; or "(no EXIF data found)" if empty

### `imagemeta.Decode` Options

The imagemeta library uses a streaming decoder with callback-based tag handling:

```go
imagemeta.Decode(imagemeta.Options{
    R:          f,
    ImageFormat: imageFormatFromExt(path),
    Sources:    imagemeta.EXIF,
    HandleTag: func(ti imagemeta.TagInfo) error {
        if ti.Source == imagemeta.EXIF {
            exifTags[ti.Tag] = fmt.Sprintf("%v", ti.Value)
        }
        return nil
    },
})
```

- `R` — an `io.ReadSeeker` (the open file)
- `ImageFormat` — tells the decoder which format to expect
- `Sources` — bitmask; we request only `EXIF`
- `HandleTag` — called once per discovered tag; `ti` is a `TagInfo` struct with
  `Source`, `Tag`, `Namespace`, and `Value` fields

### `main`

1. Validates that at least one argument was provided
2. Iterates over all arguments:
   - Skips unsupported extensions
   - Calls `processFile` and logs errors to stderr without stopping

## Installation

### Prerequisites

- Go 1.27.2 or later
- Internet access (to fetch `github.com/bep/imagemeta`)

### Build

```bash
cd Arachinda/scorpion
go mod tidy
go build -o scorpion .
```

This produces a `scorpion` binary and updates `go.sum`.

### Run from source

```bash
cd Arachinda/scorpion
go run . <FILE1> [<FILE2> ...]
```

## Usage

```bash
./scorpion FILE1 [FILE2 ...]
```

No flags — all arguments are treated as input file paths.

## Examples

### Single file
```bash
./scorpion photo.jpg
```

Output:
```
=== photo.jpg ===
Size: 245760 bytes
ModTime: 2024-01-15T14:30:00+01:00
Dimensions: 800x600
Format: jpeg
EXIF:
  Make: Apple
  Model: iPhone 14
  DateTimeOriginal: 2024-01-15 14:30:00
```

### Multiple files
```bash
./scorpion photo.jpg screenshot.png snapshot.gif
```

### Unsupported file (skipped)
```bash
./scorpion document.pdf
# Skipping unsupported file: document.pdf
```

### No arguments
```bash
./scorpion
# Usage: scorpion FILE1 [FILE2 ...]
```

## Testing

### Manual test with sample images

```bash
cd Arachinda/scorpion
go build -o scorpion .

# JPEG with potential EXIF
./scorpion /path/to/photo.jpg

# PNG (usually no EXIF)
./scorpion /path/to/image.png

# Multiple files
./scorpion a.jpg b.png c.gif
```

### Verify error handling

```bash
./scorpion /nonexistent.jpg
# Error processing /nonexistent.jpg: open file: open /nonexistent.jpg: no such file or directory

./scorpion main.go
# Skipping unsupported file: main.go
```

## What We Learned

- **Image decoding**: `image.DecodeConfig` reads headers only (fast, low memory)
- **EXIF parsing**: `github.com/bep/imagemeta` streaming decoder with `HandleTag`
  callback; `TagInfo` struct fields (`Source`, `Tag`, `Namespace`, `Value`)
- **File metadata**: `os.Stat` for size and modification time
- **Reader management**: `os.Open`, `defer f.Close()`, `f.Seek(0, 0)` to rewind
- **Map usage**: `make(map[string]string)` to collect tags before ordered printing
- **Error resilience**: per-file error handling prevents batch abort on single failure
- **Go modules**: `go mod tidy` to fetch and record dependencies

## Edge Cases & Limitations

| Issue | Behavior |
|-------|----------|
| No EXIF data | Prints "(no EXIF data found)" |
| Corrupt image | `image.DecodeConfig` returns error; EXIF decode may also fail |
| Unsupported extension | Skipped with stderr warning |
| Missing file | Error logged to stderr, continues to next file |
| BMP format | Mapped to TIFF decoder; EXIF support is limited |
| Large files | `image.DecodeConfig` reads only header; imagemeta streams metadata |

