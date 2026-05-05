// Package db provides photo storage utilities for plant records.
// It handles saving, retrieving, generating thumbnails, and deleting
// photo files associated with plant serial numbers.
//
// Photos are stored in the user's home directory under ~/.gardener/photos/<serial>/.
// Each photo is saved with a unique timestamped filename to support multiple images per plant.
package db

import (
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const PhotosDir = "photos"

// GetPhotoDir returns the absolute path to the directory where photos
// for the given plant serial number are stored.
//
// The serial number is sanitized to prevent path traversal attacks:
// only alphanumeric characters, underscores, and hyphens are preserved.
// All other characters are replaced with underscores.
//
// Example: GetPhotoDir("PKG-001") -> "/home/user/.gardener/photos/PKG-001"
func GetPhotoDir(serial string) string {
	home, _ := os.UserHomeDir()
	// Sanitize serial to prevent path traversal: allow only safe characters.
	safeSerial := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, serial)
	return filepath.Join(home, ".gardener", PhotosDir, safeSerial)
}

// SavePhoto copies a photo file from the given source path into the
// application's photo storage for the specified plant serial number.
//
// This is a legacy function that saves a single photo as "main.<ext>".
// For new uploads supporting multiple photos per plant, use SavePhotoFile instead.
//
// Parameters:
//   - serial: the plant's unique serial number (used for directory naming)
//   - sourcePath: absolute path to the source image file
//
// Returns:
//   - The saved filename (e.g., "main.jpg"), or empty string if sourcePath is empty
//   - An error if any file operation fails
func SavePhoto(serial, sourcePath string) (string, error) {
	if sourcePath == "" {
		return "", nil
	}

	// Create the plant's photo directory if it doesn't exist.
	destDir := GetPhotoDir(serial)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Preserve the original file extension; default to .jpg if none.
	ext := filepath.Ext(sourcePath)
	if ext == "" {
		ext = ".jpg"
	}
	filename := "main" + ext
	destPath := filepath.Join(destDir, filename)

	// Read and write the file content.
	input, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return "", err
	}

	// Generate a thumbnail for quick preview in the UI.
	GenerateThumbnail(sourcePath, filepath.Join(destDir, "thumb.jpg"), 150)

	return filename, nil
}

// GenerateThumbnail creates a resized JPEG thumbnail of the source image.
//
// The thumbnail maintains the original aspect ratio and is scaled so that
// its largest dimension does not exceed maxSize pixels.
//
// Parameters:
//   - srcPath: path to the source image file
//   - dstPath: path where the thumbnail will be written
//   - maxSize: maximum width or height in pixels (the other dimension scales proportionally)
//
// Errors during decoding, scaling, or writing are silently ignored;
// the function returns without creating a thumbnail if any step fails.
func GenerateThumbnail(srcPath, dstPath string, maxSize int) {
	file, err := os.Open(srcPath)
	if err != nil {
		return
	}
	defer file.Close()

	src, _, err := image.Decode(file)
	if err != nil {
		return
	}

	// Calculate scaled dimensions while preserving aspect ratio.
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > height {
		if width > maxSize {
			height = height * maxSize / width
			width = maxSize
		}
	} else {
		if height > maxSize {
			width = width * maxSize / height
			height = maxSize
		}
	}

	// Scale the image using bilinear interpolation for quality.
	thumb := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(thumb, thumb.Rect, src, bounds, draw.Over, nil)

	out, err := os.Create(dstPath)
	if err != nil {
		return
	}
	defer out.Close()
	jpeg.Encode(out, thumb, &jpeg.Options{Quality: 85})
}

// DeletePhotos removes the entire photo directory for the given plant serial number.
//
// This deletes all photos and thumbnails associated with the plant.
// If the directory does not exist, the function returns nil (no error).
//
// Parameters:
//   - serial: the plant's unique serial number
//
// Returns:
//   - An error if removal fails (e.g., permission denied), or nil on success
func DeletePhotos(serial string) error {
	dir := GetPhotoDir(serial)
	return os.RemoveAll(dir)
}

// SavePhotoFile saves an uploaded photo from an io.Reader into the application's
// photo storage with a unique, timestamped filename.
//
// This function is designed for handling multipart form uploads where multiple
// photos can be associated with a single plant record. Each photo receives a
// unique name to avoid collisions: "main_<nanoseconds_timestamp>.<ext>".
//
// A thumbnail is also generated for the saved photo.
//
// Parameters:
//   - serial: the plant's unique serial number (used for directory naming)
//   - src: io.Reader containing the image data (e.g., from http.FormFile)
//   - originalFilename: the original filename from the upload (used to extract extension)
//
// Returns:
//   - The unique saved filename (e.g., "main_1714912345678901234.jpg")
//   - An error if directory creation, file writing, or thumbnail generation fails
func SavePhotoFile(serial string, src io.Reader, originalFilename string) (string, error) {
	destDir := GetPhotoDir(serial)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(originalFilename)
	if ext == "" {
		ext = ".jpg"
	}

	// Generate a unique filename using nanosecond timestamp to prevent collisions.
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("main_%d%s", timestamp, ext)
	destPath := filepath.Join(destDir, filename)

	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	if err != nil {
		return "", err
	}

	// Generate a thumbnail for the newly saved photo.
	thumbName := strings.TrimSuffix(filename, ext) + "_thumb" + ext
	GenerateThumbnail(destPath, filepath.Join(destDir, thumbName), 150)

	return filename, nil
}
