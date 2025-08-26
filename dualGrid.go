package dualgrid

import (
	"errors"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type (
	TileType uint8
	Material [16]*ebiten.Image
	DualGrid struct {
		Materials             []Material
		DefaultMaterial       TileType
		TileSize              int
		WorldGrid             Grid
		GridWidth, GridHeight int
	}
)

var (
	TilemapDimensionError = errors.New("Tilemap Image isnt the right dimension")
	TextureDimensionError = errors.New("Texture Image isnt the right dimension")
	MaskDimensionError    = errors.New("Mask isnt the right dimension")

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
func (g *DualGrid) AddMaterialFromTilemap(tilemapImage *ebiten.Image) error {

	if tilemapImage.Bounds().Dx() != 4*g.TileSize || tilemapImage.Bounds().Dy() != 4*g.TileSize {
		return TilemapDimensionError
	}

	// reorder for bitmask indexing
	newMaterial := Material{}
	for i := range 16 {
		x := (i % 4) * g.TileSize
		y := (i / 4) * g.TileSize

		newMaterial[tileOrder[i]] = ebiten.NewImageFromImage(tilemapImage.SubImage(image.Rect(x, y, x+g.TileSize, y+g.TileSize)).(*ebiten.Image))
	}
	g.Materials = append(g.Materials, newMaterial)
	return nil
}

// Take a base texture, a 4x4 mask and add the Material to the Dualgrid
func (g *DualGrid) AddMaterialFromMask(textureImage, maskImage *ebiten.Image) error {
	if textureImage.Bounds().Dx() != g.TileSize || textureImage.Bounds().Dy() != g.TileSize {
		return TextureDimensionError
	}
	if maskImage.Bounds().Dx() != 4*g.TileSize || maskImage.Bounds().Dy() != 4*g.TileSize {
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

	// grab the base material, multiply by the mask to "stamp out" the shape
	tempImage := ebiten.NewImage(4*g.TileSize, 4*g.TileSize)
	for i := range 16 {
		x := (i % 4) * g.TileSize
		y := (i / 4) * g.TileSize

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(x), float64(y))
		tempImage.DrawImage(textureImage, opts)
	}
	tempImage.DrawImage(maskImage, multiplyOpts)

	return g.AddMaterialFromTilemap(tempImage)
}

// Really need to be refactored for partial render (camera)
func (g DualGrid) DrawTo(img *ebiten.Image, left, top int) {
	img.Clear()

	var widthInTile int = img.Bounds().Dx() / g.TileSize
	var heightInTile int = img.Bounds().Dy() / g.TileSize
	var tileStartX int = left / g.TileSize
	var tileStartY int = top / g.TileSize
	var offsetX float64 = float64(left % g.TileSize)
	var offsetY float64 = float64(top % g.TileSize)

	var tileX, tileY int
	var tl, tr, bl, br TileType
	var matType TileType
	var matTypeMask = make([]bool, len(g.Materials))
	var bitmask int

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

			// reset mask
			for i := range matTypeMask {
				matTypeMask[i] = false
			}

			matTypeMask[tl] = true
			matTypeMask[tr] = true
			matTypeMask[bl] = true
			matTypeMask[br] = true

			// draw up to 4 images per tile for better layering
			matType = 0
			for i := range len(g.Materials) {
				if matTypeMask[i] == false {
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

				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(x*g.TileSize)-offsetX, float64(y*g.TileSize)-offsetY)
				img.DrawImage(g.Materials[matType][bitmask], opts)
			}
		}
	}
}
