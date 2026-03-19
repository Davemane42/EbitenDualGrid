package dualgrid

import (
	"errors"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	TilemapDimensionError = errors.New("Tilemap Image isnt the right dimension")
	TextureDimensionError = errors.New("Texture Image isnt the right dimension")
	MaskDimensionError    = errors.New("Mask Image isnt the right dimension")
)

// Material holds the pre-processed texture strip and variant data for one tile type.
//
//	Texture:
//		Is an horizontal strip where each "slot" is TileSize wide
//		First 16 slots are the computed texture, followed by any variant tiles.
type Material struct {
	TileSize   int
	TileCount  int
	Texture    *ebiten.Image
	VarientMap VarientMap
}

// VarientMap maps a bitmask index (0–15) to a list of alternate tile indices within
// the material's texture strip. Used to add visual variety to specific tile shapes.
//
//	bitmask is a 4bit number 0b0000 Top-Left, Top-Right, Bottom-Left and Bottom-Right
//	dualgrid.VarientMap{
//		3:  {17}, bitmask index 3 0b0011 has an extra variant an material "slot" 17
//		5:  {16}, index 5  0b0101 -> slot 16
//		10: {19}, index 10 0b1010 -> slot 19
//		12: {18}, index 12 0b1100 -> slot 18
//	}
type VarientMap [16][]int

// NewMaterialFromTilemap takes a 4x4 tilemap and builds a Material.
func NewMaterialFromTilemap(tileSize int, tilemapImage *ebiten.Image, varientMap VarientMap) (Material, error) {
	if tilemapImage.Bounds().Dx() != 4*tileSize || tilemapImage.Bounds().Dy() < 4*tileSize {
		return Material{}, TilemapDimensionError
	}
	if tilemapImage.Bounds().Dy() > 4*tileSize && tilemapImage.Bounds().Dy()%tileSize != 0 {
		return Material{}, TilemapDimensionError
	}

	var varientCount int
	for _, v := range varientMap {
		varientCount += len(v)
	}

	m := Material{}
	m.TileCount = 16 + varientCount
	m.Texture = ebiten.NewImage(m.TileCount*tileSize, tileSize)
	m.VarientMap = VarientMap{}

	// reorder tiles for bitmap indexing (different order from what i found online, dont remember why)
	tileOrder := []int{2, 5, 11, 3, 9, 7, 15, 14, 4, 12, 13, 10, 0, 1, 6, 8}
	var opts ebiten.DrawImageOptions
	for i := range 16 {
		x := (i % 4) * tileSize
		y := (i / 4) * tileSize

		opts.GeoM.Reset()
		opts.GeoM.Translate(float64(tileOrder[i]*tileSize), 0)

		m.Texture.DrawImage(tilemapImage.SubImage(image.Rect(x, y, x+tileSize, y+tileSize)).(*ebiten.Image), &opts)
	}
	var i = 16
	for k, varient := range varientMap {
		if len(varient) == 0 {
			continue
		}
		// store the "default" tile as the first varient
		m.VarientMap[k] = append(m.VarientMap[k], k)

		for _, varientIndex := range varient {
			m.VarientMap[k] = append(m.VarientMap[k], i)

			x := (varientIndex % 4) * tileSize
			y := (varientIndex / 4) * tileSize

			opts.GeoM.Reset()
			opts.GeoM.Translate(float64(i*tileSize), 0)

			m.Texture.DrawImage(tilemapImage.SubImage(image.Rect(x, y, x+tileSize, y+tileSize)).(*ebiten.Image), &opts)

			i++
		}
	}

	return m, nil
}

// NewMaterialFromMask takes a base texture and a 4x4 mask and builds a Material.
func NewMaterialFromMask(tileSize int, textureImage, maskImage *ebiten.Image, varientMap VarientMap) (Material, error) {
	if textureImage.Bounds().Dx() != tileSize || textureImage.Bounds().Dy() != tileSize {
		return Material{}, TextureDimensionError
	}
	if maskImage.Bounds().Dx() != 4*tileSize || maskImage.Bounds().Dy() < 4*tileSize {
		return Material{}, MaskDimensionError
	}
	if maskImage.Bounds().Dy() > 4*tileSize && maskImage.Bounds().Dy()%tileSize != 0 {
		return Material{}, MaskDimensionError
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

	maskTileHeight := maskImage.Bounds().Dy() / tileSize

	// grab the base material, multiply by the mask to "stamp out" the shape
	tempImage := ebiten.NewImage(maskImage.Bounds().Dx(), maskImage.Bounds().Dy())
	var stampOpts ebiten.DrawImageOptions
	for i := range maskTileHeight * 4 {
		x := (i % 4) * tileSize
		y := (i / 4) * tileSize

		stampOpts.GeoM.Reset()
		stampOpts.GeoM.Translate(float64(x), float64(y))
		tempImage.DrawImage(textureImage, &stampOpts)
	}
	tempImage.DrawImage(maskImage, multiplyOpts)

	mat, err := NewMaterialFromTilemap(tileSize, tempImage, varientMap)
	tempImage.Dispose()
	return mat, err
}
