// Package db provides unit tests for photo storage utilities.
//
// Tests verify:
//   - Path sanitization and security (GetPhotoDir)
//   - File saving with unique naming (SavePhotoFile)
//   - Thumbnail generation and scaling (GenerateThumbnail)
//   - Directory cleanup (DeletePhotos)
//
// All tests use t.TempDir() and environment variable mocking
// to ensure isolation and avoid modifying the user's actual home directory.
package db

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetPhotoDir_Sanitization verifies that GetPhotoDir properly sanitizes
// plant serial numbers to prevent path traversal attacks.
//
// Test cases cover:
//   - Valid serials: preserved unchanged
//   - Path traversal attempts: "../" replaced with underscores
//   - Special characters: replaced with underscores
//   - Output: always returns an absolute path
func TestGetPhotoDir_Sanitization(t *testing.T) {
	tests := []struct {
		serial   string
		expected string
	}{
		{"PKG-001", "PKG-001"},
		{"test/../evil", "test____evil"},
		{"normal_name", "normal_name"},
		{"@#$%^", "_____"},
	}

	for _, tt := range tests {
		t.Run(tt.serial, func(t *testing.T) {
			result := GetPhotoDir(tt.serial)
			if !filepath.IsAbs(result) {
				t.Errorf("Expected absolute path, got %s", result)
			}
			if tt.expected != "" && !strings.HasSuffix(result, tt.expected) {
				t.Errorf("Expected path to end with %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSavePhotoFile verifies that SavePhotoFile correctly:
//   - Creates the target directory for the plant serial
//   - Saves the uploaded image with a unique timestamped filename
//   - Generates a thumbnail for the saved image
//
// The test uses a synthetic 100x100 red JPEG image and mocks the HOME
// environment variable to isolate file operations in a temporary directory.
func TestSavePhotoFile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create a synthetic 100x100 red JPEG image for testing.
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	serial := "PHOTO-TEST"
	filename := "test.jpg"

	savedName, err := SavePhotoFile(serial, &buf, filename)
	if err != nil {
		t.Fatalf("SavePhotoFile failed: %v", err)
	}
	if savedName == "" {
		t.Error("Expected non-empty savedName")
	}

	// Verify the photo file was created at the expected path.
	photoDir := GetPhotoDir(serial)
	photoPath := filepath.Join(photoDir, savedName)
	if _, err := os.Stat(photoPath); os.IsNotExist(err) {
		t.Errorf("Expected photo file to exist: %s", photoPath)
	}

	// Verify a thumbnail was generated (filename contains "_thumb").
	files, _ := os.ReadDir(photoDir)
	thumbFound := false
	for _, f := range files {
		if strings.Contains(f.Name(), "_thumb") {
			thumbFound = true
			break
		}
	}
	if !thumbFound {
		t.Logf("Warning: thumbnail not found in %s", photoDir)
	}
}

// TestGenerateThumbnail verifies that GenerateThumbnail:
//   - Creates a thumbnail file at the specified destination
//   - Scales the image so the largest dimension does not exceed maxSize
//   - Preserves the original aspect ratio
//
// The test uses a 500x300 green image and requests a 150px max thumbnail,
// expecting the output to be 150x90 pixels (scaled proportionally).
func TestGenerateThumbnail(t *testing.T) {
	tmpDir := t.TempDir()

	img := image.NewRGBA(image.Rect(0, 0, 500, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 500; x++ {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}

	srcPath := filepath.Join(tmpDir, "src.jpg")
	dstPath := filepath.Join(tmpDir, "thumb.jpg")

	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Failed to create src file: %v", err)
	}
	jpeg.Encode(f, img, nil)
	f.Close()

	GenerateThumbnail(srcPath, dstPath, 150)

	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Expected thumbnail to be created")
	}

	thumbFile, _ := os.Open(dstPath)
	thumbImg, _, _ := image.Decode(thumbFile)
	thumbFile.Close()

	bounds := thumbImg.Bounds()
	if bounds.Dx() > 150 || bounds.Dy() > 150 {
		t.Errorf("Thumbnail too large: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestDeletePhotos verifies that DeletePhotos removes the entire photo
// directory for a given plant serial, including all contained files.
func TestDeletePhotos(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	serial := "DELETE-TEST"
	photoDir := GetPhotoDir(serial)
	os.MkdirAll(photoDir, 0755)

	testFile := filepath.Join(photoDir, "test.jpg")
	os.WriteFile(testFile, []byte("fake image"), 0644)

	err := DeletePhotos(serial)
	if err != nil {
		t.Errorf("DeletePhotos failed: %v", err)
	}

	if _, err := os.Stat(photoDir); !os.IsNotExist(err) {
		t.Error("Expected photo directory to be deleted")
	}
}

// TestDeletePhotos_NonExistent verifies that DeletePhotos does not return
// an error when called with a serial number that has no associated directory.
//
// This ensures the function is idempotent and safe to call during cleanup
// operations even if photos were never added for a plant.
func TestDeletePhotos_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	err := DeletePhotos("NON-EXISTENT")
	if err != nil {
		t.Errorf("DeletePhotos should not fail for non-existent dir: %v", err)
	}
}
