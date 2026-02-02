package models

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/taigrr/trophy/pkg/math3d"
)

// STLLoader loads STL (stereolithography) files in both ASCII and binary formats.
type STLLoader struct {
	// Options
	SmoothNormals bool // If true, average normals per-vertex for smooth shading
}

// NewSTLLoader creates a new STL loader with default settings.
func NewSTLLoader() *STLLoader {
	return &STLLoader{
		SmoothNormals: false,
	}
}

// LoadFile loads an STL file from disk.
func (l *STLLoader) LoadFile(path string) (*Mesh, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read STL file: %w", err)
	}

	return l.LoadBytes(data, path)
}

// LoadBytes parses STL from a byte slice.
func (l *STLLoader) LoadBytes(data []byte, name string) (*Mesh, error) {
	if isBinarySTL(data) {
		return l.loadBinary(data, name)
	}
	return l.loadASCII(data, name)
}

// Load parses STL from a reader.
// Note: This reads the entire content into memory to detect format.
func (l *STLLoader) Load(r io.Reader, name string) (*Mesh, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read STL data: %w", err)
	}
	return l.LoadBytes(data, name)
}

// isBinarySTL detects if the data is binary STL format.
// Binary STL starts with 80-byte header, then 4-byte triangle count.
// ASCII STL starts with "solid".
func isBinarySTL(data []byte) bool {
	if len(data) < 84 {
		return false
	}

	// Check if it starts with "solid" (ASCII format)
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if bytes.HasPrefix(trimmed, []byte("solid")) {
		// Could still be binary if "solid" appears in header
		// Check if triangle count matches file size
		triCount := binary.LittleEndian.Uint32(data[80:84])
		expectedSize := 84 + triCount*50
		if uint32(len(data)) == expectedSize {
			return true
		}
		return false
	}

	return true
}

// loadBinary parses binary STL format.
func (l *STLLoader) loadBinary(data []byte, name string) (*Mesh, error) {
	if len(data) < 84 {
		return nil, fmt.Errorf("binary STL too short: %d bytes", len(data))
	}

	// Skip 80-byte header
	triCount := binary.LittleEndian.Uint32(data[80:84])

	expectedSize := 84 + triCount*50
	if uint32(len(data)) < expectedSize {
		return nil, fmt.Errorf("binary STL truncated: expected %d bytes, got %d", expectedSize, len(data))
	}

	mesh := NewMesh(name)

	// Vertex deduplication map
	vertexMap := make(map[math3d.Vec3]int)

	offset := 84
	for i := uint32(0); i < triCount; i++ {
		// Read normal (3 floats = 12 bytes)
		normal := math3d.V3(
			float64(readFloat32LE(data[offset:])),
			float64(readFloat32LE(data[offset+4:])),
			float64(readFloat32LE(data[offset+8:])),
		)
		offset += 12

		// Read 3 vertices (9 floats = 36 bytes)
		var faceVerts [3]int
		for v := 0; v < 3; v++ {
			pos := math3d.V3(
				float64(readFloat32LE(data[offset:])),
				float64(readFloat32LE(data[offset+4:])),
				float64(readFloat32LE(data[offset+8:])),
			)
			offset += 12

			// Deduplicate vertex
			if idx, exists := vertexMap[pos]; exists {
				faceVerts[v] = idx
			} else {
				idx := len(mesh.Vertices)
				mesh.Vertices = append(mesh.Vertices, MeshVertex{
					Position: pos,
					Normal:   normal,
				})
				vertexMap[pos] = idx
				faceVerts[v] = idx
			}
		}

		// Skip 2-byte attribute byte count
		offset += 2

		mesh.Faces = append(mesh.Faces, Face{
			V:        faceVerts,
			Material: -1,
		})
	}

	mesh.CalculateBounds()

	if l.SmoothNormals {
		mesh.CalculateSmoothNormals()
	}

	return mesh, nil
}

// readFloat32LE reads a little-endian float32 from a byte slice.
func readFloat32LE(data []byte) float32 {
	bits := binary.LittleEndian.Uint32(data)
	return math.Float32frombits(bits)
}

// loadASCII parses ASCII STL format.
func (l *STLLoader) loadASCII(data []byte, name string) (*Mesh, error) {
	mesh := NewMesh(name)

	// Vertex deduplication map
	vertexMap := make(map[math3d.Vec3]int)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0

	var currentNormal math3d.Vec3
	var faceVerts []int
	inFacet := false
	inLoop := false

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch strings.ToLower(fields[0]) {
		case "solid":
			if len(fields) > 1 {
				mesh.Name = fields[1]
			}

		case "facet":
			if len(fields) >= 5 && strings.ToLower(fields[1]) == "normal" {
				nx, err := strconv.ParseFloat(fields[2], 64)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid normal x: %w", lineNum, err)
				}
				ny, err := strconv.ParseFloat(fields[3], 64)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid normal y: %w", lineNum, err)
				}
				nz, err := strconv.ParseFloat(fields[4], 64)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid normal z: %w", lineNum, err)
				}
				currentNormal = math3d.V3(nx, ny, nz).Normalize()
			}
			inFacet = true
			faceVerts = nil

		case "outer":
			if len(fields) >= 2 && strings.ToLower(fields[1]) == "loop" {
				inLoop = true
			}

		case "vertex":
			if !inFacet || !inLoop {
				return nil, fmt.Errorf("line %d: vertex outside facet/loop", lineNum)
			}
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: vertex needs x y z", lineNum)
			}

			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid vertex x: %w", lineNum, err)
			}
			y, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid vertex y: %w", lineNum, err)
			}
			z, err := strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid vertex z: %w", lineNum, err)
			}

			pos := math3d.V3(x, y, z)

			// Deduplicate vertex
			if idx, exists := vertexMap[pos]; exists {
				faceVerts = append(faceVerts, idx)
			} else {
				idx := len(mesh.Vertices)
				mesh.Vertices = append(mesh.Vertices, MeshVertex{
					Position: pos,
					Normal:   currentNormal,
				})
				vertexMap[pos] = idx
				faceVerts = append(faceVerts, idx)
			}

		case "endloop":
			inLoop = false

		case "endfacet":
			if len(faceVerts) >= 3 {
				mesh.Faces = append(mesh.Faces, Face{
					V:        [3]int{faceVerts[0], faceVerts[1], faceVerts[2]},
					Material: -1,
				})
			}
			inFacet = false
			faceVerts = nil

		case "endsolid":
			// Done

		default:
			// Ignore unknown
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading ASCII STL: %w", err)
	}

	mesh.CalculateBounds()

	if l.SmoothNormals {
		mesh.CalculateSmoothNormals()
	}

	return mesh, nil
}

// LoadSTL is a convenience function to load an STL file with default settings.
func LoadSTL(path string) (*Mesh, error) {
	return NewSTLLoader().LoadFile(path)
}

// LoadSTLSmooth loads an STL file with smooth normals.
func LoadSTLSmooth(path string) (*Mesh, error) {
	loader := NewSTLLoader()
	loader.SmoothNormals = true
	return loader.LoadFile(path)
}
