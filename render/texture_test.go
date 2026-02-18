package render

import (
	"testing"
)

func TestNewTexture(t *testing.T) {
	tex := NewTexture(64, 64)
	if tex.Width != 64 || tex.Height != 64 {
		t.Errorf("Expected 64x64, got %dx%d", tex.Width, tex.Height)
	}
	if len(tex.Pixels) != 64*64 {
		t.Errorf("Expected %d pixels, got %d", 64*64, len(tex.Pixels))
	}
}

func TestCheckerTexture(t *testing.T) {
	white := RGB(255, 255, 255)
	black := RGB(0, 0, 0)
	tex := NewCheckerTexture(64, 64, 8, white, black)
	// Check a white cell (0,0 -> 7,7)
	c := tex.GetPixel(4, 4)
	if c != white {
		t.Errorf("Expected white at (4,4), got %v", c)
	}
	// Check a black cell (8,0 -> 15,7)
	c = tex.GetPixel(12, 4)
	if c != black {
		t.Errorf("Expected black at (12,4), got %v", c)
	}
}

func TestTextureSampleNearest(t *testing.T) {
	tex := NewTexture(2, 2)
	tex.SetPixel(0, 0, RGB(255, 0, 0))   // Red at top-left
	tex.SetPixel(1, 0, RGB(0, 255, 0))   // Green at top-right
	tex.SetPixel(0, 1, RGB(0, 0, 255))   // Blue at bottom-left
	tex.SetPixel(1, 1, RGB(255, 255, 0)) // Yellow at bottom-right
	tex.FilterMode = FilterNearest
	// Sample corners (V is flipped, so V=1 is image Y=0)
	tests := []struct {
		u, v     float64
		expected Color
		name     string
	}{
		{0.01, 0.99, RGB(255, 0, 0), "top-left (red)"},
		{0.99, 0.99, RGB(0, 255, 0), "top-right (green)"},
		{0.01, 0.01, RGB(0, 0, 255), "bottom-left (blue)"},
		{0.99, 0.01, RGB(255, 255, 0), "bottom-right (yellow)"},
	}
	for _, tt := range tests {
		c := tex.Sample(tt.u, tt.v)
		if c != tt.expected {
			t.Errorf("Sample(%v, %v) = %v, want %v (%s)", tt.u, tt.v, c, tt.expected, tt.name)
		}
	}
}

func TestTextureWrapRepeat(t *testing.T) {
	tex := NewTexture(2, 2)
	tex.SetPixel(0, 0, RGB(255, 0, 0))
	tex.WrapU = WrapRepeat
	tex.WrapV = WrapRepeat
	tex.FilterMode = FilterNearest
	// U=1.01 should wrap to U=0.01
	c1 := tex.Sample(0.01, 0.99)
	c2 := tex.Sample(1.01, 0.99)
	if c1 != c2 {
		t.Errorf("Wrap repeat failed: Sample(0.01, 0.99)=%v != Sample(1.01, 0.99)=%v", c1, c2)
	}
}

func TestTextureWrapClamp(t *testing.T) {
	tex := NewTexture(2, 2)
	tex.SetPixel(0, 0, RGB(255, 0, 0)) // Red top-left
	tex.SetPixel(1, 0, RGB(0, 255, 0)) // Green top-right
	tex.WrapU = WrapClamp
	tex.WrapV = WrapClamp
	tex.FilterMode = FilterNearest
	// U=-0.5 should clamp to U=0
	c := tex.Sample(-0.5, 0.99)
	expected := RGB(255, 0, 0)
	if c != expected {
		t.Errorf("Wrap clamp failed: Sample(-0.5, 0.99)=%v, want %v (red)", c, expected)
	}
	// U=1.5 should clamp to U=1
	c = tex.Sample(1.5, 0.99)
	expected = RGB(0, 255, 0)
	if c != expected {
		t.Errorf("Wrap clamp failed: Sample(1.5, 0.99)=%v, want %v (green)", c, expected)
	}
}

func TestMultiplyColor(t *testing.T) {
	c := RGB(200, 100, 50)
	result := MultiplyColor(c, 0.5)
	if result.R != 100 || result.G != 50 || result.B != 25 {
		t.Errorf("MultiplyColor failed: got %v", result)
	}
	// Test clamping
	result = MultiplyColor(c, 2.0)
	if result.R != 255 {
		t.Errorf("MultiplyColor should clamp to 255, got %d", result.R)
	}
}

func TestModulateColor(t *testing.T) {
	white := RGB(255, 255, 255)
	red := RGB(255, 0, 0)
	result := ModulateColor(white, red)
	if result != red {
		t.Errorf("ModulateColor(white, red) = %v, want %v", result, red)
	}
	half := RGB(128, 128, 128)
	result = ModulateColor(half, white)
	// 128 * 255 / 255 = 128
	if result.R != 128 || result.G != 128 || result.B != 128 {
		t.Errorf("ModulateColor(half, white) = %v, want gray", result)
	}
}

func TestLerpColor(t *testing.T) {
	black := RGB(0, 0, 0)
	white := RGB(255, 255, 255)
	// Midpoint should be gray
	mid := lerpColor(black, white, 0.5)
	if mid.R != 127 || mid.G != 127 || mid.B != 127 {
		t.Errorf("lerpColor midpoint = %v, want gray(127)", mid)
	}
	// Endpoints
	start := lerpColor(black, white, 0.0)
	if start != black {
		t.Errorf("lerpColor(0.0) = %v, want black", start)
	}
	end := lerpColor(black, white, 1.0)
	if end != white {
		t.Errorf("lerpColor(1.0) = %v, want white", end)
	}
}
