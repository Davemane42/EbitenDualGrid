package main

import (
	"image"
	"log"

	dualgrid "github.com/davemane42/EbitenDualGrid"
	assets "github.com/davemane42/EbitenDualGrid/example/assets"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	tileSize     = 16
	gridWidth    = 20
	gridHeight   = 14
	screenWidth  = (gridWidth + 1) * tileSize
	screenHeight = (gridHeight + 1) * tileSize
)

// Material indices
const (
	MatRock      dualgrid.TileType = 0
	MatDarkRock  dualgrid.TileType = 1
	MatDarkGrass dualgrid.TileType = 2
	MatGrass     dualgrid.TileType = 3
	MatFlowers   dualgrid.TileType = 4
)

type Game struct {
	dualGrid dualgrid.DualGrid
	canvas   *ebiten.Image
}

func NewGame() *Game {
	// 1. Create a DualGrid — default material is grass (the base ground)
	dg := dualgrid.NewDualGrid(gridWidth, gridHeight, tileSize, MatGrass)

	// 2. Create materials

	// Extract individual material swatches from the materialTypes image
	mats := make([]*ebiten.Image, assets.Images["materialTypes"].Bounds().Dx()/tileSize)
	for i := range mats {
		mats[i] = assets.Images["materialTypes"].SubImage(
			image.Rect(i*tileSize, 0, i*tileSize+tileSize, tileSize),
		).(*ebiten.Image)
	}

	// NewMaterialFromMask "stamps" out the shape from the extracted material using a 4x4 tile mask
	rockMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[0], assets.Images["rockMask"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	dirtMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[1], assets.Images["rockMask"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}

	// VarientMap are used to add diversity so tiles dont repeat as frequently
	// Bitmask is a 4bit number 0b0000 Top-Left, Top-Right, Bottom-Left and Bottom-Right
	darkGrassMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[2], assets.Images["grassMask"], dualgrid.VarientMap{
		3:  {17}, // bitmask index 3 0b0011 has an extra variant an material "slot" 17
		5:  {16}, // index 5  0b0101 -> slot 16
		10: {19}, // index 10 0b1010 -> slot 19
		12: {18}, // index 12 0b1100 -> slot 18
	})
	if err != nil {
		log.Fatal(err)
	}
	grassMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[3], assets.Images["softMask"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}

	// Use NewMaterialFromTilemap if you have an already made Tilemap (unused)
	greenGrass, err := dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["grassTilemap"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}

	// 3. Add the materials
	dg.AddMaterial(rockMat)
	dg.AddMaterial(dirtMat)
	dg.AddMaterial(darkGrassMat)
	dg.AddMaterial(grassMat)
	dg.AddMaterial(greenGrass)

	// Draw diagonal bands:
	for x := range gridWidth {
		for y := range gridHeight {
			d := x - y
			switch {
			case d >= 1 && d <= 5:
				dg.SetCell(x, y, MatRock)
			case d >= -4 && d <= 10:
				dg.SetCell(x, y, MatDarkRock)
			case d >= -9 && d <= 15:
				dg.SetCell(x, y, MatDarkGrass)
			}
		}
	}

	// 4. Render
	canvas := ebiten.NewImage(screenWidth, screenHeight)
	dg.DrawTo(canvas, 0, 0)

	return &Game{dualGrid: dg, canvas: canvas}
}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.canvas, &ebiten.DrawImageOptions{})
}

func (g *Game) Layout(_, _ int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth*3, screenHeight*3)
	ebiten.SetWindowTitle("DualGrid - simple example")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
