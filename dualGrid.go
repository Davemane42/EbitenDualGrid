package dualgrid

import (
	"errors"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type TileType uint8

type Material [16]*ebiten.Image

type DualGrid struct {
	Materials             []Material
	DefaultMaterial       TileType
	TileSize              int
	WorldGrid             Grid
	GridWidth, GridHeight int
}

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

func (g *DualGrid) AddMaterial(material, mask *ebiten.Image) {
	if material.Bounds().Dx() != g.TileSize || material.Bounds().Dy() != g.TileSize {
		log.Fatal(errors.New("Material isnt the right dimension"))
	}
	if mask.Bounds().Dx() != 4*g.TileSize || mask.Bounds().Dy() != 4*g.TileSize {
		log.Fatal(errors.New("Mask isnt the right dimension"))
	}

	opts := &ebiten.DrawImageOptions{}
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

	// reorder tiles for bitmap indexing (different order from what i found online, dont remember why)
	order := []int{2, 5, 11, 3, 9, 7, 15, 14, 4, 12, 13, 10, 0, 1, 6, 8}

	// for each tile you grab the base material, multiply by the right mask and store in the right order
	newMaterial := Material{}
	tempImage := ebiten.NewImage(g.TileSize, g.TileSize)
	for i := range 16 {
		x := (i % 4) * g.TileSize
		y := (i / 4) * g.TileSize

		finalImage := ebiten.NewImage(g.TileSize, g.TileSize)

		tempImage.DrawImage(material, opts)
		tempImage.DrawImage(mask.SubImage(image.Rect(x, y, x+g.TileSize, y+g.TileSize)).(*ebiten.Image), multiplyOpts)

		finalImage.DrawImage(tempImage, opts)

		newMaterial[order[i]] = finalImage
	}
	g.Materials = append(g.Materials, newMaterial)
}

func (g DualGrid) DrawTo(img *ebiten.Image) {
	var xPos, yPos float64
	var tl, tr, bl, br TileType
	var matType TileType
	var matTypeMask = make([]bool, len(g.Materials))
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
				bitmask := 0b0000
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
