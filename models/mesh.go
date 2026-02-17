// Package models provides 3D model loading and representation for Trophy.
package models

import (
	"image"

	"github.com/ansipixels/trophy/math3d"
)

// Mesh represents a 3D mesh with vertices, faces, and materials.
type Mesh struct {
	Name      string
	Vertices  []MeshVertex
	Faces     []Face
	Materials []Material

	// Bounding box (calculated on load)
	BoundsMin math3d.Vec3
	BoundsMax math3d.Vec3
}

// MeshVertex holds all vertex attributes.
type MeshVertex struct {
	Position math3d.Vec3
	Normal   math3d.Vec3
	UV       math3d.Vec2
}

// Face represents a triangle face with vertex indices and material reference.
type Face struct {
	V        [3]int // Indices into Mesh.Vertices
	Material int    // Index into Mesh.Materials (-1 for no material)
}

// Material represents a PBR material from GLTF.
type Material struct {
	Name       string
	BaseColor  [4]float64  // RGBA in 0-1 range
	Metallic   float64     // 0 = dielectric, 1 = metal
	Roughness  float64     // 0 = smooth, 1 = rough
	BaseMap    image.Image // Optional base color texture
	HasTexture bool
}

// NewMesh creates an empty mesh.
func NewMesh(name string) *Mesh {
	return &Mesh{
		Name:      name,
		Vertices:  make([]MeshVertex, 0),
		Faces:     make([]Face, 0),
		BoundsMin: math3d.V3(0, 0, 0),
		BoundsMax: math3d.V3(0, 0, 0),
	}
}

// CalculateBounds computes the axis-aligned bounding box.
func (m *Mesh) CalculateBounds() {
	if len(m.Vertices) == 0 {
		return
	}

	m.BoundsMin = m.Vertices[0].Position
	m.BoundsMax = m.Vertices[0].Position

	for _, v := range m.Vertices[1:] {
		m.BoundsMin = m.BoundsMin.Min(v.Position)
		m.BoundsMax = m.BoundsMax.Max(v.Position)
	}
}

// Center returns the center of the bounding box.
func (m *Mesh) Center() math3d.Vec3 {
	return m.BoundsMin.Add(m.BoundsMax).Scale(0.5)
}

// Size returns the dimensions of the bounding box.
func (m *Mesh) Size() math3d.Vec3 {
	return m.BoundsMax.Sub(m.BoundsMin)
}

// TriangleCount returns the number of triangles.
func (m *Mesh) TriangleCount() int {
	return len(m.Faces)
}

// VertexCount returns the number of vertices.
func (m *Mesh) VertexCount() int {
	return len(m.Vertices)
}

// CalculateNormals computes face normals and assigns them to vertices.
// This is a simple flat-shading approach; for smooth shading, normals
// should be averaged per-vertex.
func (m *Mesh) CalculateNormals() {
	for i := range m.Faces {
		f := &m.Faces[i]
		v0 := m.Vertices[f.V[0]].Position
		v1 := m.Vertices[f.V[1]].Position
		v2 := m.Vertices[f.V[2]].Position

		edge1 := v1.Sub(v0)
		edge2 := v2.Sub(v0)
		normal := edge1.Cross(edge2).Normalize()

		// Assign to vertices (flat shading - each face has its own normal)
		m.Vertices[f.V[0]].Normal = normal
		m.Vertices[f.V[1]].Normal = normal
		m.Vertices[f.V[2]].Normal = normal
	}
}

// CalculateSmoothNormals computes averaged normals for smooth shading.
func (m *Mesh) CalculateSmoothNormals() {
	// Reset all normals
	for i := range m.Vertices {
		m.Vertices[i].Normal = math3d.Zero3()
	}

	// Accumulate face normals per vertex
	for _, f := range m.Faces {
		v0 := m.Vertices[f.V[0]].Position
		v1 := m.Vertices[f.V[1]].Position
		v2 := m.Vertices[f.V[2]].Position

		edge1 := v1.Sub(v0)
		edge2 := v2.Sub(v0)
		normal := edge1.Cross(edge2) // Don't normalize yet

		m.Vertices[f.V[0]].Normal = m.Vertices[f.V[0]].Normal.Add(normal)
		m.Vertices[f.V[1]].Normal = m.Vertices[f.V[1]].Normal.Add(normal)
		m.Vertices[f.V[2]].Normal = m.Vertices[f.V[2]].Normal.Add(normal)
	}

	// Normalize all accumulated normals
	for i := range m.Vertices {
		m.Vertices[i].Normal = m.Vertices[i].Normal.Normalize()
	}
}

// Transform applies a transformation matrix to all vertices.
func (m *Mesh) Transform(mat math3d.Mat4) {
	for i := range m.Vertices {
		m.Vertices[i].Position = mat.MulVec3(m.Vertices[i].Position)
		// Transform normals with inverse transpose (for non-uniform scaling)
		// For now, just use the rotation part
		m.Vertices[i].Normal = mat.MulVec3Dir(m.Vertices[i].Normal).Normalize()
	}
	m.CalculateBounds()
}

// Clone creates a deep copy of the mesh.
func (m *Mesh) Clone() *Mesh {
	clone := &Mesh{
		Name:      m.Name,
		Vertices:  make([]MeshVertex, len(m.Vertices)),
		Faces:     make([]Face, len(m.Faces)),
		Materials: make([]Material, len(m.Materials)),
		BoundsMin: m.BoundsMin,
		BoundsMax: m.BoundsMax,
	}
	copy(clone.Vertices, m.Vertices)
	copy(clone.Faces, m.Faces)
	copy(clone.Materials, m.Materials)
	return clone
}

// GetVertex returns the position, normal, and UV for vertex i.
// Implements render.MeshRenderer interface.
func (m *Mesh) GetVertex(i int) (pos, normal math3d.Vec3, uv math3d.Vec2) {
	v := m.Vertices[i]
	return v.Position, v.Normal, v.UV
}

// GetFace returns the vertex indices for face i.
// Implements render.MeshRenderer interface.
func (m *Mesh) GetFace(i int) [3]int {
	return m.Faces[i].V
}

// GetFaceMaterial returns the material index for face i.
// Returns -1 if no material assigned.
func (m *Mesh) GetFaceMaterial(i int) int {
	return m.Faces[i].Material
}

// GetMaterial returns the material at index i.
// Returns nil if index is out of bounds or -1.
func (m *Mesh) GetMaterial(i int) *Material {
	if i < 0 || i >= len(m.Materials) {
		return nil
	}
	return &m.Materials[i]
}

// MaterialCount returns the number of materials.
func (m *Mesh) MaterialCount() int {
	return len(m.Materials)
}

// GetBounds returns the axis-aligned bounding box.
// Implements render.BoundedMeshRenderer interface.
func (m *Mesh) GetBounds() (min, max math3d.Vec3) {
	return m.BoundsMin, m.BoundsMax
}

// faceKey creates a canonical key for a face by sorting vertex indices.
// Two faces with the same vertices (in any order) will have the same key.
func faceKey(v0, v1, v2 int) [3]int {
	// Sort the three indices
	if v0 > v1 {
		v0, v1 = v1, v0
	}
	if v1 > v2 {
		v1, v2 = v2, v1
	}
	if v0 > v1 {
		v0, v1 = v1, v0
	}
	return [3]int{v0, v1, v2}
}

// DeduplicateFaces removes duplicate faces from the mesh.
// Two faces are considered duplicates if they have the same three vertices
// (regardless of winding order). When duplicates are found, only the first
// occurrence is kept.
// Returns the number of faces removed.
func (m *Mesh) DeduplicateFaces() int {
	if len(m.Faces) == 0 {
		return 0
	}

	seen := make(map[[3]int]bool)
	kept := make([]Face, 0, len(m.Faces))

	for _, f := range m.Faces {
		key := faceKey(f.V[0], f.V[1], f.V[2])
		if !seen[key] {
			seen[key] = true
			kept = append(kept, f)
		}
	}

	removed := len(m.Faces) - len(kept)
	m.Faces = kept
	return removed
}

// RemoveInternalFaces removes pairs of coplanar faces that face opposite directions.
// These typically occur at the boundaries of merged/combined meshes where internal
// geometry should be removed. For each pair of faces sharing the same vertices but
// with opposite normals, both faces are removed.
// Returns the number of faces removed.
func (m *Mesh) RemoveInternalFaces() int {
	if len(m.Faces) == 0 {
		return 0
	}

	// Group faces by their vertex key
	type faceInfo struct {
		index  int
		normal math3d.Vec3
	}
	groups := make(map[[3]int][]faceInfo)

	for i, f := range m.Faces {
		// Calculate face normal
		v0 := m.Vertices[f.V[0]].Position
		v1 := m.Vertices[f.V[1]].Position
		v2 := m.Vertices[f.V[2]].Position
		edge1 := v1.Sub(v0)
		edge2 := v2.Sub(v0)
		normal := edge1.Cross(edge2).Normalize()

		key := faceKey(f.V[0], f.V[1], f.V[2])
		groups[key] = append(groups[key], faceInfo{index: i, normal: normal})
	}

	// Mark faces for removal - pairs with opposing normals
	toRemove := make(map[int]bool)
	for _, faceList := range groups {
		if len(faceList) < 2 {
			continue
		}

		// Check all pairs for opposing normals
		for i := range faceList {
			if toRemove[faceList[i].index] {
				continue
			}
			for j := i + 1; j < len(faceList); j++ {
				if toRemove[faceList[j].index] {
					continue
				}
				// Check if normals are roughly opposite (dot product close to -1)
				dot := faceList[i].normal.Dot(faceList[j].normal)
				if dot < -0.99 {
					// These faces are coplanar and facing opposite directions
					// This is internal geometry - remove both
					toRemove[faceList[i].index] = true
					toRemove[faceList[j].index] = true
					break
				}
			}
		}
	}

	if len(toRemove) == 0 {
		return 0
	}

	// Build new face list without removed faces
	kept := make([]Face, 0, len(m.Faces)-len(toRemove))
	for i, f := range m.Faces {
		if !toRemove[i] {
			kept = append(kept, f)
		}
	}

	removed := len(m.Faces) - len(kept)
	m.Faces = kept
	return removed
}

// CleanMesh performs all mesh cleanup operations:
// 1. Remove degenerate faces (zero area)
// 2. Remove internal faces (coplanar opposing pairs) - must come before dedup!
// 3. Remove duplicate faces
// 4. Remove unreferenced vertices
// Returns the total number of faces removed.
func (m *Mesh) CleanMesh() int {
	removed := 0

	// Remove degenerate faces first
	removed += m.RemoveDegenerateFaces()

	// Remove internal geometry (coplanar opposing faces) BEFORE dedup
	// because DeduplicateFaces would remove one of the pair before we can detect it
	removed += m.RemoveInternalFaces()

	// Remove exact duplicates (same vertices, same direction)
	removed += m.DeduplicateFaces()

	// Clean up unreferenced vertices
	m.RemoveUnreferencedVertices()

	return removed
}

// RemoveDegenerateFaces removes faces with zero or near-zero area.
// Returns the number of faces removed.
func (m *Mesh) RemoveDegenerateFaces() int {
	if len(m.Faces) == 0 {
		return 0
	}

	const minArea = 1e-10
	kept := make([]Face, 0, len(m.Faces))

	for _, f := range m.Faces {
		// Check for duplicate vertex indices
		if f.V[0] == f.V[1] || f.V[1] == f.V[2] || f.V[0] == f.V[2] {
			continue
		}

		// Check for near-zero area using cross product magnitude
		v0 := m.Vertices[f.V[0]].Position
		v1 := m.Vertices[f.V[1]].Position
		v2 := m.Vertices[f.V[2]].Position
		edge1 := v1.Sub(v0)
		edge2 := v2.Sub(v0)
		cross := edge1.Cross(edge2)
		area := cross.Len() * 0.5

		if area > minArea {
			kept = append(kept, f)
		}
	}

	removed := len(m.Faces) - len(kept)
	m.Faces = kept
	return removed
}

// RemoveUnreferencedVertices removes vertices that are not referenced by any face.
// This compacts the vertex array and updates face indices accordingly.
func (m *Mesh) RemoveUnreferencedVertices() {
	if len(m.Faces) == 0 || len(m.Vertices) == 0 {
		return
	}

	// Mark referenced vertices
	referenced := make([]bool, len(m.Vertices))
	for _, f := range m.Faces {
		referenced[f.V[0]] = true
		referenced[f.V[1]] = true
		referenced[f.V[2]] = true
	}

	// Build compacted vertex list and index mapping
	newIndex := make([]int, len(m.Vertices))
	newVertices := make([]MeshVertex, 0, len(m.Vertices))
	for i, v := range m.Vertices {
		if referenced[i] {
			newIndex[i] = len(newVertices)
			newVertices = append(newVertices, v)
		}
	}

	// Update face indices
	for i := range m.Faces {
		m.Faces[i].V[0] = newIndex[m.Faces[i].V[0]]
		m.Faces[i].V[1] = newIndex[m.Faces[i].V[1]]
		m.Faces[i].V[2] = newIndex[m.Faces[i].V[2]]
	}

	m.Vertices = newVertices
}
