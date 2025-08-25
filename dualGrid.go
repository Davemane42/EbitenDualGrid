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
func (g DualGrid) DrawTo(img *ebiten.Image) {
	var xPos, yPos float64
	var tl, tr, bl, br TileType
	var matType TileType
	var matTypeMask = make([]bool, len(g.Materials))
	var bitmask int
	for x := range g.GridWidth + 1 {
		xPos = float64(x * g.TileSize)
		for y := range g.GridHeight + 1 {
			yPos = float64(y * g.TileSize)

			tl = g.DefaultMaterial
			tr = g.DefaultMaterial
			bl = g.DefaultMaterial
			br = g.DefaultMaterial

			// If inbound set corners to grid value
			if x >= 1 && y >= 1 {
				tl = g.WorldGrid[x-1][y-1]
			}
			if x < g.GridWidth && y >= 1 {
				tr = g.WorldGrid[x][y-1]
			}
			if x >= 1 && y < g.GridHeight {
				bl = g.WorldGrid[x-1][y]
			}
			if x < g.GridWidth && y < g.GridHeight {
				br = g.WorldGrid[x][y]
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
				opts.GeoM.Translate(xPos, yPos)
				img.DrawImage(g.Materials[matType][bitmask], opts)

			}
		}
	}
}
