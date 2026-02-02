package models

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestSTLLoaderASCII(t *testing.T) {
	// Simple ASCII STL cube (partial - one face)
	asciiSTL := `solid cube
  facet normal 0 0 -1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 1 1 0
    endloop
  endfacet
  facet normal 0 0 -1
    outer loop
      vertex 0 0 0
      vertex 1 1 0
      vertex 0 1 0
    endloop
  endfacet
endsolid cube`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load ASCII STL: %v", err)
	}

	if mesh.Name != "cube" {
		t.Errorf("Name = %q, want %q", mesh.Name, "cube")
	}

	if mesh.TriangleCount() != 2 {
		t.Errorf("TriangleCount = %d, want 2", mesh.TriangleCount())
	}

	// Should have 4 unique vertices (square)
	if mesh.VertexCount() != 4 {
		t.Errorf("VertexCount = %d, want 4 (deduplicated)", mesh.VertexCount())
	}
}

func TestSTLLoaderBinary(t *testing.T) {
	// Create a simple binary STL with one triangle
	var buf bytes.Buffer

	// 80-byte header
	header := make([]byte, 80)
	copy(header, []byte("Binary STL test"))
	buf.Write(header)

	// Triangle count (1)
	binary.Write(&buf, binary.LittleEndian, uint32(1))

	// Triangle: normal + 3 vertices + attribute
	// Normal: 0, 0, 1
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(1))

	// Vertex 1: 0, 0, 0
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))

	// Vertex 2: 1, 0, 0
	binary.Write(&buf, binary.LittleEndian, float32(1))
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))

	// Vertex 3: 0, 1, 0
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(1))
	binary.Write(&buf, binary.LittleEndian, float32(0))

	// Attribute byte count
	binary.Write(&buf, binary.LittleEndian, uint16(0))

	loader := NewSTLLoader()
	mesh, err := loader.LoadBytes(buf.Bytes(), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load binary STL: %v", err)
	}

	if mesh.TriangleCount() != 1 {
		t.Errorf("TriangleCount = %d, want 1", mesh.TriangleCount())
	}

	if mesh.VertexCount() != 3 {
		t.Errorf("VertexCount = %d, want 3", mesh.VertexCount())
	}

	// Check normal
	v := mesh.Vertices[0]
	if v.Normal.Z != 1.0 {
		t.Errorf("Normal.Z = %f, want 1.0", v.Normal.Z)
	}
}

func TestSTLDetection(t *testing.T) {
	// ASCII should not be detected as binary
	ascii := []byte("solid test\nfacet normal 0 0 1\n")
	if isBinarySTL(ascii) {
		t.Error("ASCII STL detected as binary")
	}

	// Binary with matching size should be detected
	var buf bytes.Buffer
	buf.Write(make([]byte, 80))                              // header
	binary.Write(&buf, binary.LittleEndian, uint32(0))       // 0 triangles
	if !isBinarySTL(buf.Bytes()) {
		t.Error("Binary STL not detected")
	}
}

func TestSTLVertexDeduplication(t *testing.T) {
	// Two triangles sharing an edge
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 1 0
    endloop
  endfacet
  facet normal 0 0 1
    outer loop
      vertex 1 0 0
      vertex 1 1 0
      vertex 0 1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// 2 triangles = 6 vertices, but 2 are shared, so 4 unique
	if mesh.VertexCount() != 4 {
		t.Errorf("VertexCount = %d, want 4 (deduplicated)", mesh.VertexCount())
	}

	if mesh.TriangleCount() != 2 {
		t.Errorf("TriangleCount = %d, want 2", mesh.TriangleCount())
	}
}

func TestSTLSmoothNormals(t *testing.T) {
	// Two triangles at 90 degrees
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 0 1
    endloop
  endfacet
  facet normal 0 -1 0
    outer loop
      vertex 0 0 0
      vertex 0 0 1
      vertex 0 -1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	loader.SmoothNormals = true
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Find the shared vertex at origin
	for _, v := range mesh.Vertices {
		if v.Position.X == 0 && v.Position.Y == 0 && v.Position.Z == 0 {
			// Shared vertex should have averaged normal
			// Between (0,0,1) and (0,-1,0), normalized
			expectedLen := math.Sqrt(v.Normal.X*v.Normal.X + v.Normal.Y*v.Normal.Y + v.Normal.Z*v.Normal.Z)
			if math.Abs(expectedLen-1.0) > 0.001 {
				t.Errorf("Smooth normal not normalized: length = %f", expectedLen)
			}
		}
	}
}

func TestSTLBounds(t *testing.T) {
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex -1 -2 -3
      vertex 4 5 6
      vertex 0 0 0
    endloop
  endfacet
endsolid test`

	mesh, err := LoadSTL("")
	_ = mesh
	_ = err

	loader := NewSTLLoader()
	mesh, err = loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	min, max := mesh.GetBounds()

	if min.X != -1 || min.Y != -2 || min.Z != -3 {
		t.Errorf("BoundsMin = %v, want (-1, -2, -3)", min)
	}

	if max.X != 4 || max.Y != 5 || max.Z != 6 {
		t.Errorf("BoundsMax = %v, want (4, 5, 6)", max)
	}
}
