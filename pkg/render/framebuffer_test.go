package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFramebufferSavePNG(t *testing.T) {
	// Create a small framebuffer with a gradient
	fb := NewFramebuffer(100, 100)
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			fb.SetPixel(x, y, RGB(uint8(x*2), uint8(y*2), 128))
		}
	}

	// Save to temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.png")

	err := fb.SavePNG(path)
	if err != nil {
		t.Fatalf("SavePNG failed: %v", err)
	}

	// Verify file exists and has content
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("File is empty")
	}
}

func TestFramebufferToImage(t *testing.T) {
	fb := NewFramebuffer(50, 50)
	fb.SetPixel(10, 20, ColorRed)
	fb.SetPixel(30, 40, ColorGreen)

	img := fb.ToImage()

	if img.Bounds().Dx() != 50 || img.Bounds().Dy() != 50 {
		t.Errorf("Image dimensions wrong: got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Check specific pixels
	r, g, b, a := img.At(10, 20).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("Red pixel wrong: got %d,%d,%d,%d", r>>8, g>>8, b>>8, a>>8)
	}

	r, g, b, a = img.At(30, 40).RGBA()
	if r>>8 != 0 || g>>8 != 255 || b>>8 != 0 {
		t.Errorf("Green pixel wrong: got %d,%d,%d,%d", r>>8, g>>8, b>>8, a>>8)
	}
}
