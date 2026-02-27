package dualgrid

import (
	"errors"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type (
	TileType   uint8
	VarientMap map[int][]int
	Material   struct {
		Texture    *ebiten.Image
		TileSize   int
		VarientMap map[int][]int
		TileCount  int
	}
	DualGrid struct {
		Materials             []Material
		DefaultMaterial       TileType
		TileSize              int
		WorldGrid             Grid
		GridWidth, GridHeight int
		Texture               *ebiten.Image
	}
)

var (
	TilemapDimensionError = errors.New("Tilemap Image isnt the right dimension")
	TextureDimensionError = errors.New("Texture Image isnt the right dimension")
	MaskDimensionError    = errors.New("Mask Image isnt the right dimension")

	// reorder tiles for bitmap indexing (different order from what i found online, dont remember why)
	tileOrder = []int{2, 5, 11, 3, 9, 7, 15, 14, 4, 12, 13, 10, 0, 1, 6, 8}
)

func NewDualGrid(width, height, tileSize int, defaultMaterial TileType) DualGrid {
	return DualGrid{
		Materials:       []Material{},
		DefaultMaterial: defaultMaterial,
		TileSize:        tileSize,
		WorldGrid:       NewGridWithValue(width, height, defaultMaterial),
		GridWidth:       width,
		GridHeight:      height,
	}
}

func (g DualGrid) IsInbound(x, y int) bool {
	return x >= 0 && y >= 0 && x < g.GridWidth && y < g.GridHeight
}

// Take a 4x4 tilemap, reorder in into a Material and add it to the Dualgrid
func (g *DualGrid) AddMaterialFromTilemap(tilemapImage *ebiten.Image, varientMap VarientMap) error {

	if tilemapImage.Bounds().Dx() != 4*g.TileSize || tilemapImage.Bounds().Dy() < 4*g.TileSize {
		return TilemapDimensionError
	}
	if tilemapImage.Bounds().Dy() > 4*g.TileSize && tilemapImage.Bounds().Dy()%g.TileSize != 0 {
		return TilemapDimensionError
	}

	//var tilemapTileHeight = int(math.Ceil(float64(tilemapImage.Bounds().Dy() / g.TileSize)))

	var varientCount int
	for _, v := range varientMap {
		varientCount += len(v)
	}

	newMaterial := Material{}
	newMaterial.TileCount = 16 + varientCount
	newMaterial.Texture = ebiten.NewImage(newMaterial.TileCount*g.TileSize, g.TileSize)
	newMaterial.VarientMap = VarientMap{}

	// reorder for bitmask indexing
	var opts ebiten.DrawImageOptions
	for i := range 16 {
		x := (i % 4) * g.TileSize
		y := (i / 4) * g.TileSize

		opts.GeoM.Reset()
		opts.GeoM.Translate(float64(tileOrder[i]*g.TileSize), 0)

		newMaterial.Texture.DrawImage(tilemapImage.SubImage(image.Rect(x, y, x+g.TileSize, y+g.TileSize)).(*ebiten.Image), &opts)
	}
	var i = 16
	for k, varient := range varientMap {
		// store the "default" tile as the first varient
		newMaterial.VarientMap[k] = append(newMaterial.VarientMap[k], k)

		for _, varientIndex := range varient {
			newMaterial.VarientMap[k] = append(newMaterial.VarientMap[k], i)

			x := (varientIndex % 4) * g.TileSize
			y := (varientIndex / 4) * g.TileSize

			opts.GeoM.Reset()
			opts.GeoM.Translate(float64(i*g.TileSize), 0)

			newMaterial.Texture.DrawImage(tilemapImage.SubImage(image.Rect(x, y, x+g.TileSize, y+g.TileSize)).(*ebiten.Image), &opts)

			i++
		}
	}

	g.Materials = append(g.Materials, newMaterial)
	return nil
}

// Take a base texture, a 4x4 mask and add the Material to the Dualgrid
func (g *DualGrid) AddMaterialFromMask(textureImage, maskImage *ebiten.Image, varientMap VarientMap) error {
	if textureImage.Bounds().Dx() != g.TileSize || textureImage.Bounds().Dy() != g.TileSize {
		return TextureDimensionError
	}
	if maskImage.Bounds().Dx() != 4*g.TileSize || maskImage.Bounds().Dy() < 4*g.TileSize {
		return MaskDimensionError
	}
	if maskImage.Bounds().Dy() > 4*g.TileSize && maskImage.Bounds().Dy()%g.TileSize != 0 {
		return MaskDimensionError
	}

	multiplyOpts := &ebiten.DrawImageOptions{
		Blend: ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorZero,
			BlendFactorSourceAlpha:      ebiten.BlendFactorSourceAlpha,
			BlendFactorDestinationRGB:   ebiten.BlendFactorSourceColor,
			BlendFactorDestinationAlpha: ebiten.BlendFactorZero,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		},
	}

	var maskTileHeight = maskImage.Bounds().Dy() / g.TileSize

	// grab the base material, multiply by the mask to "stamp out" the shape
	tempImage := ebiten.NewImage(maskImage.Bounds().Dx(), maskImage.Bounds().Dy())
	var stampOpts ebiten.DrawImageOptions
	for i := range maskTileHeight * 4 {
		x := (i % 4) * g.TileSize
		y := (i / 4) * g.TileSize

		stampOpts.GeoM.Reset()
		stampOpts.GeoM.Translate(float64(x), float64(y))
		tempImage.DrawImage(textureImage, &stampOpts)
	}
	tempImage.DrawImage(maskImage, multiplyOpts)

	return g.AddMaterialFromTilemap(tempImage, varientMap)
}

func (g DualGrid) DrawTo(img *ebiten.Image, left, top int) {
	img.Clear()

	widthInTile := img.Bounds().Dx() / g.TileSize
	heightInTile := img.Bounds().Dy() / g.TileSize
	tileStartX := left / g.TileSize
	tileStartY := top / g.TileSize
	offsetX := float32(left % g.TileSize)
	offsetY := float32(top % g.TileSize)
	ts := float32(g.TileSize)

	var tileX, tileY int
	var tl, tr, bl, br TileType
	var matType TileType
	var matTypeMask [256]bool // TileType is uint8 so max 256 values, no heap alloc per call
	var bitmask int

	// One vertex/index buffer slice per material
	capacity := widthInTile * heightInTile
	vertices := make([][]ebiten.Vertex, len(g.Materials))
	indices := make([][]uint16, len(g.Materials))
	for i := range g.Materials {
		vertices[i] = make([]ebiten.Vertex, 0, capacity*4)
		indices[i] = make([]uint16, 0, capacity*6)
	}

	for x := range widthInTile {
		tileX = tileStartX + x
		if tileX < 0 || tileX >= g.GridWidth+1 {
			continue
		}
		for y := range heightInTile {
			tileY = tileStartY + y
			if tileY < 0 || tileY >= g.GridHeight+1 {
				continue
			}

			tl = g.DefaultMaterial
			tr = g.DefaultMaterial
			bl = g.DefaultMaterial
			br = g.DefaultMaterial

			// If inbound set corners to grid value
			if tileX >= 1 && tileY >= 1 {
				tl = g.WorldGrid[tileX-1][tileY-1]
			}
			if tileX < g.GridWidth && tileY >= 1 {
				tr = g.WorldGrid[tileX][tileY-1]
			}
			if tileX >= 1 && tileY < g.GridHeight {
				bl = g.WorldGrid[tileX-1][tileY]
			}
			if tileX < g.GridWidth && tileY < g.GridHeight {
				br = g.WorldGrid[tileX][tileY]
			}

			matTypeMask[tl] = true
			matTypeMask[tr] = true
			matTypeMask[bl] = true
			matTypeMask[br] = true

			dstX := float32(x)*ts - offsetX
			dstY := float32(y)*ts - offsetY

			// Draw up to 4 layers per tile for "layering"
			for i := range len(g.Materials) {
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
				if v, ok := g.Materials[matType].VarientMap[bitmask]; ok {
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
	for i, mat := range g.Materials {
		if len(vertices[i]) == 0 {
			continue
		}
		img.DrawTriangles(vertices[i], indices[i], mat.Texture, &drawOpts)
	}
}
