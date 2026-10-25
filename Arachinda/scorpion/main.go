package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bep/imagemeta"
)

var supportedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
}

func isSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExts[ext]
}

func imageFormatFromExt(path string) imagemeta.ImageFormat {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return imagemeta.JPEG
	case ".png":
		return imagemeta.PNG
	case ".gif":
		return imagemeta.GIF
	case ".bmp":
		return imagemeta.TIFF // BMP uses TIFF decoder
	default:
		return imagemeta.JPEG
	}
}

func processFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	fmt.Printf("=== %s ===\n", path)
	fmt.Printf("Size: %d bytes\n", info.Size())
	fmt.Printf("ModTime: %s\n", info.ModTime().Format(time.RFC3339))

	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		fmt.Printf("Dimensions: (unable to decode image header)\n")
	} else {
		fmt.Printf("Dimensions: %dx%d\n", cfg.Width, cfg.Height)
		fmt.Printf("Format: %s\n", format)
	}

	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("seek file: %w", err)
	}

	exifTags := make(map[string]string)
	_, err = imagemeta.Decode(imagemeta.Options{
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

	if len(exifTags) > 0 {
		fmt.Printf("EXIF:\n")
		order := []string{"Make", "Model", "DateTime", "DateTimeOriginal", "DateTimeDigitized"}
		for _, k := range order {
			if v, ok := exifTags[k]; ok {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
		// Print remaining tags not in the ordered list
		seen := make(map[string]bool)
		for _, k := range order {
			seen[k] = true
		}
		for k, v := range exifTags {
			if !seen[k] {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
	} else {
		fmt.Printf("EXIF: (no EXIF data found)\n")
	}

	fmt.Println()
	return nil
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: scorpion FILE1 [FILE2 ...]")
		os.Exit(1)
	}

	for _, arg := range args {
		if !isSupported(arg) {
			fmt.Fprintf(os.Stderr, "Skipping unsupported file: %s\n", arg)
			continue
		}
		if err := processFile(arg); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", arg, err)
		}
	}
}
