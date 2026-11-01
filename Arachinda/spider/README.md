# Arachnida — Exercise 1: Spider

A recursive web image scraper written in Go. Given a seed URL, it downloads all images
from the page, optionally recurses into linked pages up to a configurable depth, and
saves the images to a local directory.

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

`spider` is a command-line tool that fetches an HTML page, extracts image URLs
(`<img src>`), filters them by supported extensions, downloads the image bytes to
disk, and — when recursion is enabled — follows `<a href>` links to crawl deeper
pages up to a user-specified depth limit.

It was built as part of the Cybersecurity Piscine "Arachnida" project at 42 Berlin,
which introduces web scraping and metadata handling.

## Theory

### Web Scraping Fundamentals

Web scraping is the automated extraction of data from websites. In this project we
scrape two kinds of resources:

1. **HTML documents** — parsed to find image references and hyperlinks.
2. **Binary image files** — downloaded and saved to the local filesystem.

### Same-Origin Policy (Self-Implemented)

To avoid accidentally crawling the entire internet, recursion is restricted to the
same host as the seed URL. This is a manual implementation of a same-origin style
guardrail — the program compares the `Host` component of each discovered link
against the seed URL's host before recursing.

### Depth-Limited Crawling

Crawling is bounded by a depth parameter:

| Depth | Meaning |
|-------|---------|
| `0`   | Seed URL only |
| `1`   | Seed + pages directly linked from seed |
| `N`   | Seed + up to `N` link-hops away |

This prevents infinite recursion on circular link structures and keeps runtime
predictable.

### Visited-Set Deduplication

A `map[string]bool` tracks every URL already fetched. Before processing a page, the
crawler checks whether it has been visited. This guarantees termination even when
pages contain circular links or multiple paths to the same resource.

### Image Format Filtering

Only the following extensions are downloaded (per subject requirements):

- `.jpg` / `.jpeg`
- `.png`
- `.gif`
- `.bmp`

All other `src` attributes are ignored.

## Features

- Fetch HTML via HTTP GET with `User-Agent: Spider` and a 15-second timeout
- Parse HTML with `golang.org/x/net/html`
- Resolve relative URLs to absolute URLs using `url.URL.ResolveReference`
- Extract `<img src>` and filter by supported image extensions
- Download images to a configurable directory (default `./data/`)
- Optional recursive crawling (`-r`) with depth limit (`-l`, default `5`)
- Same-host restriction for recursion
- Visited-set deduplication to prevent infinite loops
- Configurable output path (`-p`)

## Project Structure

```
spider/
├── main.go      # Single-file source: arg parsing, HTTP client, HTML parser,
│                # crawler, image downloader
├── go.mod       # Module definition (module spider)
└── go.sum       # Dependency checksums
```

## Architecture & In-Depth Explanation

### `conf` struct

Holds all runtime configuration parsed from command-line arguments:

```go
type conf struct {
    url        string
    depth      int
    recursion  bool
    path       string
}
```

### `parseArgs`

Parses `os.Args[1:]`. Supports:

- Positional URL (must start with `http://` or `https://` and include a host)
- `-r` — enable recursive crawling
- `-l N` — maximum recursion depth (default `5`, must be non-negative)
- `-p PATH` — output directory (default `./data/`)

Returns a validation error for missing URL, unknown flags, invalid depth, or
multiple URLs.

### `getPage`

Performs an HTTP GET request:

1. Creates `http.Request` with `User-Agent: Spider`
2. Uses `http.Client` with `15s` timeout
3. Verifies `StatusCode == 200 OK`
4. Reads the full body and returns it as a string

### HTML Parsing (`parseHTML`, `findImgs`, `extractImages`)

Uses `golang.org/x/net/html` to tokenize the document into a node tree.

- `findImgs` performs a depth-first traversal of the DOM tree.
- When it encounters an `` node with a `src` attribute, it resolves the
  URL relative to the page's base URL and checks the file extension.
- `extractImages` is the public wrapper that parses HTML and returns the
  filtered list of absolute image URLs.

### Image Downloader (`downloadImage`)

1. Creates the output directory with `os.MkdirAll(savePath, 0755)`
2. Issues an HTTP GET to the image URL (same UA and timeout as page fetch)
3. Verifies `200 OK`
4. Derives a filename from the URL path via `path.Base`
5. Writes bytes with `os.WriteFile(filePath, data, 0644)`

If the URL yields no usable basename, the download is skipped with an error
message.

### Link Extraction (`findLinks`, `extractLinks`)

Mirrors `findImgs` but targets `<a href>` attributes instead of `src`.
Returns absolute URLs for every hyperlink on the page.

### Recursive Crawler (`crawl`)

The core recursive function:

```go
func crawl(pageURL string, depth int, visited map[string]bool, seedURL string, cfg conf)
```

Logic flow:

1. **Depth guard** — return if `depth > cfg.depth`
2. **Visited check** — return if URL already processed; mark as visited
3. **Fetch page** — `getPage(pageURL)`
4. **Extract + download images** — `extractImages` then `downloadImage` for each
5. **If recursion enabled**:
   - Extract links via `extractLinks`
   - Filter to same-host as `seedURL` via `sameHost`
   - Recurse into each link at `depth + 1`

### `sameHost`

Compares `url.URL.Host` of two URLs. Only links on the same host as the seed URL
are followed during recursion.

### `main`

1. Parses arguments into `cfg`
2. Initializes `visited := make(map[string]bool)`
3. Calls `crawl(cfg.url, 0, visited, cfg.url, cfg)`

## Installation

### Prerequisites

- Go 1.27.2 or later
- Internet access (to fetch the `golang.org/x/net` dependency)

### Build

```bash
cd Arachinda/spider
go mod download
go build -o spider .
```

This produces a `spider` binary in the current directory.

### Run from source

```bash
cd Arachinda/spider
go run . <URL>
```

## Usage

```bash
./spider [-rlp] URL
```

| Flag | Description | Default |
|------|-------------|---------|
| (none) | Seed URL | Required |
| `-r` | Enable recursive crawling | `false` |
| `-l N` | Maximum recursion depth | `5` |
| `-p PATH` | Output directory | `./data/` |

## Examples

### Single page, default output
```bash
./spider https://example.com
```
Downloads all supported images from `https://example.com` into `./data/`.

### Recursive, depth 2, custom path
```bash
./spider -r -l 2 -p ./images https://example.com
```
Crawls `example.com` and same-host links up to 2 hops deep, saving to `./images`.

### Recursive, default depth 5
```bash
./spider -r https://example.com
```

### Invalid usage (no URL)
```bash
./spider                      # error: no URL provided
./spider not-a-url            # error: invalid url
./spider -l -1 https://x.com  # error: invalid depth
./spider -x https://x.com     # error: unknown option: -x
```

## Testing

### Manual test on a safe page

```bash
cd Arachinda/spider
go build -o spider .
mkdir -p /tmp/spider-test
./spider -p /tmp/spider-test https://example.com
ls /tmp/spider-test
```

### Recursive test

```bash
./spider -r -l 1 -p /tmp/spider-test-rec https://example.com
ls -R /tmp/spider-test-rec
```

### Verify argument validation

```bash
./spider                      # error: no URL provided
./spider not-a-url            # error: invalid url
./spider -l -1 https://x.com  # error: invalid depth
./spider -x https://x.com     # error: unknown option: -x
```

## What We Learned

- **HTTP clients in Go**: `http.NewRequest`, `http.Client`, timeouts, custom headers
- **HTML parsing**: `golang.org/x/net/html` node tree traversal (`FirstChild`, `NextSibling`)
- **URL resolution**: `url.URL.ResolveReference` for converting relative `src`/`href` to absolute URLs
- **File I/O**: `os.MkdirAll`, `os.WriteFile`, `path.Base`, `path.Ext`
- **Recursion + state management**: `map[string]bool` visited set, depth counting, same-host filtering
- **Error handling**: per-image download failures are logged but do not abort the crawl
- **Command-line parsing**: manual flag parsing with validation (no external flag library)

## Edge Cases & Limitations

| Issue | Behavior |
|-------|----------|
| Circular links | Visited set prevents infinite loops |
| Deep recursion | Limited by `-l` (default 5) |
| External domains | Skipped during recursion (same-host only) |
| Missing filename in URL | Download skipped with stderr warning |
| Duplicate basenames | Later download overwrites earlier file |
| Non-200 responses | Logged to stderr, crawl continues |
| Large HTML bodies | No longer printed to stdout (removed in final version) |

