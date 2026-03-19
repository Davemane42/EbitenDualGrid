package dualgrid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"

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
	canvas          *ebiten.Image
	dirty           bool
	// Cached render buffers, reused across frames
	vertices [][]ebiten.Vertex
	indices  [][]uint16
}

func NewDualGrid(width, height, tileSize int, defaultMaterial TileType) DualGrid {
	return DualGrid{
		Materials:       []Material{},
		DefaultMaterial: defaultMaterial,
		TileSize:        tileSize,
		WorldGrid:       NewGridWithValue(width, height, defaultMaterial),
		canvas:          ebiten.NewImage((width+1)*tileSize, (height+1)*tileSize),
		dirty:           true,
	}
}

// SetCell updates a single cell.
func (dg *DualGrid) SetCell(x, y int, t TileType) {
	dg.WorldGrid.Cells[x*dg.WorldGrid.Height+y] = t
}

// GetCell returns the TileType at the given cell.
func (dg *DualGrid) GetCell(x, y int) TileType {
	return dg.WorldGrid.Cells[x*dg.WorldGrid.Height+y]
}

// MarkDirty schedules a full canvas redraw on the next Canvas() call.
// Call this after bulk modifications to WorldGrid.
func (dg *DualGrid) MarkDirty() {
	dg.dirty = true
}

// Canvas returns the cached full-grid rendered image, rebuilding it if dirty.
func (dg *DualGrid) Canvas() *ebiten.Image {
	w, h := dg.WorldGrid.Width, dg.WorldGrid.Height
	fullW := (w + 1) * dg.TileSize
	fullH := (h + 1) * dg.TileSize
	if dg.canvas == nil || dg.canvas.Bounds().Dx() != fullW || dg.canvas.Bounds().Dy() != fullH {
		if dg.canvas != nil {
			dg.canvas.Deallocate()
		}
		dg.canvas = ebiten.NewImage(fullW, fullH)
		dg.dirty = true
	}
	if dg.dirty {
		dg.dirty = false
		dg.DrawTo(dg.canvas, 0, 0)
	}
	return dg.canvas
}

// ViewCanvas renders only the visible world region (viewW×viewH world pixels starting at
// worldLeft,worldTop) into the internal canvas, resizing it if needed.
// The returned image should be drawn to screen using TileOffset() to correct alignment.
// Unlike Canvas(), this always redraws — use it when the viewport moves every frame.
func (dg *DualGrid) ViewCanvas(viewW, viewH, worldLeft, worldTop int) *ebiten.Image {
	if dg.canvas == nil || dg.canvas.Bounds().Dx() != viewW || dg.canvas.Bounds().Dy() != viewH {
		if dg.canvas != nil {
			dg.canvas.Deallocate()
		}
		dg.canvas = ebiten.NewImage(viewW, viewH)
	}
	dg.DrawTo(dg.canvas, worldLeft, worldTop)
	return dg.canvas
}

// Check if a X, Y coord is inside the bounds of the grid
func (dg *DualGrid) IsInbound(x, y int) bool {
	return x >= 0 && y >= 0 && x < dg.WorldGrid.Width && y < dg.WorldGrid.Height
}

// Marshal encodes the DualGrid state to bytes.
//
//	Format: [tileSize uint32][defaultMaterial uint8][numMaterials uint8][width uint32][height uint32][tiles...]
func (dg *DualGrid) Marshal() []byte {
	w, h := dg.WorldGrid.Width, dg.WorldGrid.Height
	buf := make([]byte, 14+w*h)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(dg.TileSize))
	buf[4] = byte(dg.DefaultMaterial)
	buf[5] = byte(len(dg.Materials))
	binary.LittleEndian.PutUint32(buf[6:10], uint32(w))
	binary.LittleEndian.PutUint32(buf[10:14], uint32(h))
	for i, v := range dg.WorldGrid.Cells {
		buf[14+i] = byte(v)
	}
	return buf
}

// Unmarshal loads a DualGrid state from bytes produced by Marshal.
// Returns an error if the encoded metadata does not match the current DualGrid configuration.
//
// If forceResize is true, a grid size mismatch is not an error: the WorldGrid and
// internal canvas are resized to match the saved dimensions instead.
func (dg *DualGrid) Unmarshal(data []byte, forceResize bool) error {
	if len(data) < 14 {
		return errors.New("data too short")
	}
	tileSize := int(binary.LittleEndian.Uint32(data[0:4]))
	defaultMaterial := TileType(data[4])
	numMaterials := int(data[5])
	width := int(binary.LittleEndian.Uint32(data[6:10]))
	height := int(binary.LittleEndian.Uint32(data[10:14]))

	if tileSize != dg.TileSize {
		return fmt.Errorf("tile size mismatch: file has %d, current is %d", tileSize, dg.TileSize)
	}
	if numMaterials != len(dg.Materials) {
		return fmt.Errorf("material count mismatch: file has %d, current is %d", numMaterials, len(dg.Materials))
	}
	if width != dg.WorldGrid.Width || height != dg.WorldGrid.Height {
		if !forceResize {
			return fmt.Errorf("grid size mismatch: file has %dx%d, current is %dx%d", width, height, dg.WorldGrid.Width, dg.WorldGrid.Height)
		}
		dg.WorldGrid = NewGridWithValue(width, height, defaultMaterial)
		if dg.canvas != nil {
			dg.canvas.Deallocate()
		}
		dg.canvas = ebiten.NewImage((width+1)*dg.TileSize, (height+1)*dg.TileSize)
	}
	if len(data) < 14+width*height {
		return errors.New("data truncated")
	}
	dg.DefaultMaterial = defaultMaterial
	for i, v := range data[14 : 14+width*height] {
		dg.WorldGrid.Cells[i] = TileType(v)
	}
	dg.dirty = true
	return nil
}

// AddMaterial appends a Material to the DualGrid.
func (dg *DualGrid) AddMaterial(m Material) {
	dg.Materials = append(dg.Materials, m)
	dg.dirty = true
}

// DrawTo clears img and renders the DualGrid into it from the given top-left world pixel coord.
func (dg *DualGrid) DrawTo(img *ebiten.Image, left, top int) {
	img.Clear()
	dg.renderTo(img, left, top)
}

// RedrawCanvasRegion clears and redraws the tile region at (tileX, tileY) with
// size (tileW x tileH) on the DualGrid's internal canvas.
// Automatically expands by one tile on the right/bottom for dual-grid corner overlap.
func (dg *DualGrid) RedrawCanvasRegion(tileX, tileY, tileW, tileH int) {
	left := tileX * dg.TileSize
	top := tileY * dg.TileSize
	right := (tileX + tileW + 1) * dg.TileSize
	bottom := (tileY + tileH + 1) * dg.TileSize

	b := dg.canvas.Bounds()
	left = max(left, b.Min.X)
	top = max(top, b.Min.Y)
	right = min(right, b.Max.X)
	bottom = min(bottom, b.Max.Y)
	if left >= right || top >= bottom {
		return
	}
	sub := dg.canvas.SubImage(image.Rect(left, top, right, bottom)).(*ebiten.Image)
	dg.renderTo(sub, left, top)
}

func (dg *DualGrid) renderTo(img *ebiten.Image, left, top int) {
	bounds := img.Bounds()
	widthInTile := bounds.Dx() / dg.TileSize
	heightInTile := bounds.Dy() / dg.TileSize
	tileStartX := left / dg.TileSize
	tileStartY := top / dg.TileSize
	offsetX := float32(left % dg.TileSize)
	offsetY := float32(top % dg.TileSize)
	originX := float32(bounds.Min.X)
	originY := float32(bounds.Min.Y)
	ts := float32(dg.TileSize)
	gridW := dg.WorldGrid.Width
	gridH := dg.WorldGrid.Height
	cells := dg.WorldGrid.Cells

	var tileX, tileY int
	var tl, tr, bl, br TileType
	var matType TileType
	var matTypeMask [256]bool // TileType is uint8 so max 256 values, no heap alloc per call
	var bitmask int

	// Reuse cached vertex/index buffers
	numMats := len(dg.Materials)
	if len(dg.vertices) < numMats {
		dg.vertices = make([][]ebiten.Vertex, numMats)
		dg.indices = make([][]uint16, numMats)
	}
	for i := range numMats {
		dg.vertices[i] = dg.vertices[i][:0]
		dg.indices[i] = dg.indices[i][:0]
	}

	for x := range widthInTile {
		tileX = tileStartX + x
		if tileX < 0 || tileX >= gridW+1 {
			continue
		}
		for y := range heightInTile {
			tileY = tileStartY + y
			if tileY < 0 || tileY >= gridH+1 {
				continue
			}

			tl = dg.DefaultMaterial
			tr = dg.DefaultMaterial
			bl = dg.DefaultMaterial
			br = dg.DefaultMaterial

			// If inbound set corners to grid value
			if tileX >= 1 && tileY >= 1 {
				tl = cells[(tileX-1)*gridH+(tileY-1)]
			}
			if tileX < gridW && tileY >= 1 {
				tr = cells[tileX*gridH+(tileY-1)]
			}
			if tileX >= 1 && tileY < gridH {
				bl = cells[(tileX-1)*gridH+tileY]
			}
			if tileX < gridW && tileY < gridH {
				br = cells[tileX*gridH+tileY]
			}

			matTypeMask[tl] = true
			matTypeMask[tr] = true
			matTypeMask[bl] = true
			matTypeMask[br] = true

			dstX := float32(x)*ts - offsetX + originX
			dstY := float32(y)*ts - offsetY + originY

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
				if v := dg.Materials[matType].VarientMap[bitmask]; len(v) > 0 {
					tileHash := uint32(tileX)*7919 + uint32(tileY)*6151
					bitmask = v[tileHash%uint32(len(v))]
				}

				srcX := float32(bitmask) * ts
				base := uint16(len(dg.vertices[i]))

				// TL, TR, BL, BR
				dg.vertices[i] = append(dg.vertices[i],
					ebiten.Vertex{DstX: dstX, DstY: dstY, SrcX: srcX, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dstX + ts, DstY: dstY, SrcX: srcX + ts, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dstX, DstY: dstY + ts, SrcX: srcX, SrcY: ts, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
					ebiten.Vertex{DstX: dstX + ts, DstY: dstY + ts, SrcX: srcX + ts, SrcY: ts, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
				)
				dg.indices[i] = append(dg.indices[i],
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
		if len(dg.vertices[i]) == 0 {
			continue
		}
		img.DrawTriangles(dg.vertices[i], dg.indices[i], mat.Texture, &drawOpts)
	}
}
