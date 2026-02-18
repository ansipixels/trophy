package render

import "github.com/ansipixels/trophy/math3d"

func buildTexturedTriangle(mesh MeshRenderer, face [3]int, transform math3d.Mat4) Triangle {
	p0, n0, uv0 := mesh.GetVertex(face[0])
	p1, n1, uv1 := mesh.GetVertex(face[1])
	p2, n2, uv2 := mesh.GetVertex(face[2])
	v0 := transform.MulVec3(p0)
	v1 := transform.MulVec3(p1)
	v2 := transform.MulVec3(p2)
	wn0 := transform.MulVec3Dir(n0).Normalize()
	wn1 := transform.MulVec3Dir(n1).Normalize()
	wn2 := transform.MulVec3Dir(n2).Normalize()
	return Triangle{
		V: [3]Vertex{
			{Position: v0, Normal: wn0, UV: uv0, Color: RGB(255, 255, 255)},
			{Position: v1, Normal: wn1, UV: uv1, Color: RGB(255, 255, 255)},
			{Position: v2, Normal: wn2, UV: uv2, Color: RGB(255, 255, 255)},
		},
	}
}

func buildGouraudTriangle(mesh MeshRenderer, face [3]int, transform math3d.Mat4, color Color) Triangle {
	p0, n0, _ := mesh.GetVertex(face[0])
	p1, n1, _ := mesh.GetVertex(face[1])
	p2, n2, _ := mesh.GetVertex(face[2])
	v0 := transform.MulVec3(p0)
	v1 := transform.MulVec3(p1)
	v2 := transform.MulVec3(p2)
	wn0 := transform.MulVec3Dir(n0).Normalize()
	wn1 := transform.MulVec3Dir(n1).Normalize()
	wn2 := transform.MulVec3Dir(n2).Normalize()
	return Triangle{
		V: [3]Vertex{
			{Position: v0, Normal: wn0, Color: color},
			{Position: v1, Normal: wn1, Color: color},
			{Position: v2, Normal: wn2, Color: color},
		},
	}
}
