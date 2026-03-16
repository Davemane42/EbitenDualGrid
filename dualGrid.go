package dualgrid

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// TileType identifies a material by its index in DualGrid.Materials.
// The value 0 is always the lowest-priority (bottom) layer.
type TileType uint8

// Main struct of the library
type DualGrid struct {
	TileSize        int
	DefaultMaterial TileType
	WorldGrid       Grid
	Materials       []Material
}

func NewDualGrid(width, height, tileSize int, defaultMaterial TileType) DualGrid {
	return DualGrid{
		Materials:       []Material{},
		DefaultMaterial: defaultMaterial,
		TileSize:        tileSize,
		WorldGrid:       NewGridWithValue(width, height, defaultMaterial),
	}
}

// Check if a X, Y coord is inside the bounds of the grid
func (dg DualGrid) IsInbound(x, y int) bool {
	return x >= 0 && y >= 0 && x < dg.WorldGrid.Width && y < dg.WorldGrid.Height
}

// AddMaterial appends a Material to the DualGrid.
func (dg *DualGrid) AddMaterial(m Material) {
	dg.Materials = append(dg.Materials, m)
}

// Render and draw the DualGrid to img from the top-left world coord
func (dg DualGrid) DrawTo(img *ebiten.Image, left, top int) {
	img.Clear()

	widthInTile := img.Bounds().Dx() / dg.TileSize
	heightInTile := img.Bounds().Dy() / dg.TileSize
	tileStartX := left / dg.TileSize
	tileStartY := top / dg.TileSize
	offsetX := float32(left % dg.TileSize)
	offsetY := float32(top % dg.TileSize)
	ts := float32(dg.TileSize)

	var tileX, tileY int
	var tl, tr, bl, br TileType
	var matType TileType
	var matTypeMask [256]bool // TileType is uint8 so max 256 values, no heap alloc per call
	var bitmask int

	// One vertex/index buffer slice per material
	capacity := widthInTile * heightInTile
	vertices := make([][]ebiten.Vertex, len(dg.Materials))
	indices := make([][]uint16, len(dg.Materials))
	for i := range dg.Materials {
		vertices[i] = make([]ebiten.Vertex, 0, capacity*4)
		indices[i] = make([]uint16, 0, capacity*6)
	}

	for x := range widthInTile {
		tileX = tileStartX + x
		if tileX < 0 || tileX >= dg.WorldGrid.Width+1 {
			continue
		}
		for y := range heightInTile {
			tileY = tileStartY + y
			if tileY < 0 || tileY >= dg.WorldGrid.Height+1 {
				continue
			}

			tl = dg.DefaultMaterial
			tr = dg.DefaultMaterial
			bl = dg.DefaultMaterial
			br = dg.DefaultMaterial

			// If inbound set corners to grid value
			if tileX >= 1 && tileY >= 1 {
				tl = dg.WorldGrid.Cells[tileX-1][tileY-1]
			}
			if tileX < dg.WorldGrid.Width && tileY >= 1 {
				tr = dg.WorldGrid.Cells[tileX][tileY-1]
			}
			if tileX >= 1 && tileY < dg.WorldGrid.Height {
				bl = dg.WorldGrid.Cells[tileX-1][tileY]
			}
			if tileX < dg.WorldGrid.Width && tileY < dg.WorldGrid.Height {
				br = dg.WorldGrid.Cells[tileX][tileY]
			}

			matTypeMask[tl] = true
			matTypeMask[tr] = true
			matTypeMask[bl] = true
			matTypeMask[br] = true

			dstX := float32(x)*ts - offsetX
			dstY := float32(y)*ts - offsetY

			// Draw up to 4 layers per tile for "layering"
			for i := range len(dg.Materials) {
				if !matTypeMask[i] {
					continue
				}
				matType = TileType(i)
				bitmask = 0b0000
				if tl == matType || tl > matType {
					bitmask |= 1 << 3
				}
				if tr == matType || tr > matType {
					bitmask |= 1 << 2
				}
				if bl == matType || bl > matType {
					bitmask |= 1 << 1
				}
				if br == matType || br > matType {
					bitmask |= 1 << 0
				}

				// pick a varient using a world-space coords deterministic hash
				if v, ok := dg.Materials[matType].VarientMap[bitmask]; ok {
					tileHash := uint32(tileX)*7919 + uint32(tileY)*6151
					bitmask = v[tileHash%uint32(len(v))]
				}

				srcX := float32(bitmask) * ts
				base := uint16(len(vertices[i]))

				// TL, TR, BL, BR
				vertices[i] = append(vertices[i],
					ebiten.Vertex{DstX: dstX, DstY: dstY, SrcX: srcX, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dstX + ts, DstY: dstY, SrcX: srcX + ts, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dstX, DstY: dstY + ts, SrcX: srcX, SrcY: ts, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dstX + ts, DstY: dstY + ts, SrcX: srcX + ts, SrcY: ts, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
				)
				indices[i] = append(indices[i],
					base, base+1, base+2,
					base+1, base+3, base+2,
				)
			}

			// Reset only the entries that were changed
			matTypeMask[tl] = false
			matTypeMask[tr] = false
			matTypeMask[bl] = false
			matTypeMask[br] = false
		}
	}

	// One draw call per material
	var drawOpts ebiten.DrawTrianglesOptions
	for i, mat := range dg.Materials {
		if len(vertices[i]) == 0 {
			continue
		}
		img.DrawTriangles(vertices[i], indices[i], mat.Texture, &drawOpts)
	}
}
