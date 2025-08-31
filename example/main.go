package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	dualgrid "github.com/davemane42/EbitenDualGrid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	showCorners  bool
	showGrid     bool
	showTextures bool

	updateDualGridImage bool = true
	dualGridImage       *ebiten.Image

	selectedMaterial int
	materialsColors  = []color.Color{
		color.RGBA{139, 155, 180, 255}, // Gray
		color.RGBA{115, 62, 57, 255},   // Brown
		color.RGBA{38, 92, 66, 255},    // Dark Green
		color.RGBA{62, 137, 72, 255},   // Green
		color.RGBA{254, 174, 52, 255},  // Orange
	}

	worldWidth  int
	worldHeight int
	cursorX     int
	cursorY     int
	cameraX     int
	cameraY     int
)

type Game struct {
	DualGrid dualgrid.DualGrid
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()
	MouseWorldX := (mx + cameraX - int(g.DualGrid.TileSize/2)) / g.DualGrid.TileSize
	MouseWorldY := (my + cameraY - int(g.DualGrid.TileSize/2)) / g.DualGrid.TileSize

	// Pan
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		cursorX, cursorY = ebiten.CursorPosition()
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		newCursorX, newCursorY := ebiten.CursorPosition()
		if newCursorX != cursorX || newCursorY != cursorY {
			cameraX -= newCursorX - cursorX
			cameraY -= newCursorY - cursorY
			cursorX, cursorY = newCursorX, newCursorY
			updateDualGridImage = true
		}
	}

	// Place and destroy Material
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.DualGrid.IsInbound(MouseWorldX, MouseWorldY) {
		g.DualGrid.WorldGrid[MouseWorldX][MouseWorldY] = dualgrid.TileType(selectedMaterial)
		updateDualGridImage = true
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) && g.DualGrid.IsInbound(MouseWorldX, MouseWorldY) {
		g.DualGrid.WorldGrid[MouseWorldX][MouseWorldY] = g.DualGrid.DefaultMaterial
		updateDualGridImage = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.DualGrid.DefaultMaterial = dualgrid.TileType(selectedMaterial)
		g.DualGrid.WorldGrid = dualgrid.NewGridWithValue(g.DualGrid.GridWidth, g.DualGrid.GridHeight, g.DualGrid.DefaultMaterial)
		updateDualGridImage = true
	}

	// Select Material
	if _, y := ebiten.Wheel(); y != 0 {
		if y > 0 {
			selectedMaterial = (selectedMaterial + 1) % len(g.DualGrid.Materials)
		} else {
			selectedMaterial -= 1
			if selectedMaterial < 0 {
				selectedMaterial = len(g.DualGrid.Materials) - 1
			}
		}
	}

	if updateDualGridImage {
		updateDualGridImage = false
		g.DualGrid.DrawTo(dualGridImage, cameraX, cameraY)
	}

	// Debug stuff
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		showCorners = !showCorners
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		showGrid = !showGrid
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		showTextures = !showTextures
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Clear()
	screen.DrawImage(dualGridImage, &ebiten.DrawImageOptions{})

	// Debug Draw
	if showGrid || showCorners {
		var xPos, yPos int
		var xStart, yStart = g.DualGrid.TileSize / 2, g.DualGrid.TileSize / 2
		var tl, tr, bl, br dualgrid.TileType
		cornerOffset := 2
		for x := range g.DualGrid.GridWidth + 1 {
			xPos = xStart + x*g.DualGrid.TileSize
			for y := range g.DualGrid.GridHeight + 1 {
				yPos = yStart + y*g.DualGrid.TileSize
				// Display grid
				if showGrid && x < g.DualGrid.GridWidth && y < g.DualGrid.GridHeight {
					vector.StrokeRect(
						screen,
						float32(xPos+1-cameraX), float32(yPos+1-cameraY),
						float32(g.DualGrid.TileSize-1), float32(g.DualGrid.TileSize-1),
						1,
						materialsColors[g.DualGrid.WorldGrid[x][y]],
						false,
					)
				}
				// Display grid true value
				if showCorners {
					tl = g.DualGrid.DefaultMaterial
					tr = g.DualGrid.DefaultMaterial
					bl = g.DualGrid.DefaultMaterial
					br = g.DualGrid.DefaultMaterial

					if x >= 1 && y >= 1 {
						tl = g.DualGrid.WorldGrid[x-1][y-1]
					}
					if x < g.DualGrid.GridWidth && y >= 1 {
						tr = g.DualGrid.WorldGrid[x][y-1]
					}
					if x >= 1 && y < g.DualGrid.GridHeight {
						bl = g.DualGrid.WorldGrid[x-1][y]
					}
					if x < g.DualGrid.GridWidth && y < g.DualGrid.GridHeight {
						br = g.DualGrid.WorldGrid[x][y]
					}

					vector.DrawFilledCircle(screen, float32(xPos-cornerOffset-cameraX), float32(yPos-cornerOffset-cameraY), 1, materialsColors[tl], false)
					vector.DrawFilledCircle(screen, float32(xPos+cornerOffset-cameraX), float32(yPos-cornerOffset-cameraY), 1, materialsColors[tr], false)
					vector.DrawFilledCircle(screen, float32(xPos-cornerOffset-cameraX), float32(yPos+cornerOffset-cameraY), 1, materialsColors[bl], false)
					vector.DrawFilledCircle(screen, float32(xPos+cornerOffset-cameraX), float32(yPos+cornerOffset-cameraY), 1, materialsColors[br], false)
				}
			}
		}
	}

	// Display computed textures
	if showTextures {
		matLenght := len(g.DualGrid.Materials)
		vector.DrawFilledRect(screen, 0, 0, float32(g.DualGrid.TileSize*16+16), float32(g.DualGrid.TileSize*matLenght+matLenght+1), color.White, false)
		var xPos, yPos int
		for y, mat := range g.DualGrid.Materials {
			yPos = g.DualGrid.TileSize*y + y + 1
			for x := range mat.TileCount {
				xPos = g.DualGrid.TileSize*x + x
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(xPos), float64(yPos))
				vector.DrawFilledRect(screen, float32(xPos), float32(yPos), float32(g.DualGrid.TileSize), float32(g.DualGrid.TileSize), color.Black, false)
				screen.DrawImage(mat.Texture.SubImage(image.Rect(x*g.DualGrid.TileSize, 0, x*g.DualGrid.TileSize+g.DualGrid.TileSize, g.DualGrid.TileSize)).(*ebiten.Image), opts)
			}
		}
	}

	// Current Texture Preview
	yPos := screen.Bounds().Dy() - g.DualGrid.TileSize - 1
	vector.StrokeRect(screen, 1, float32(yPos), float32(g.DualGrid.TileSize)+1, float32(g.DualGrid.TileSize)+1, 1, materialsColors[selectedMaterial], false)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(1, float64(yPos))
	screen.DrawImage(g.DualGrid.Materials[selectedMaterial].Texture.SubImage(image.Rect(15*g.DualGrid.TileSize, 0, 15*g.DualGrid.TileSize+g.DualGrid.TileSize, g.DualGrid.TileSize)).(*ebiten.Image), opts)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return worldWidth, worldHeight
}

func main() {
	grassTilemap, _, err := ebitenutil.NewImageFromFile("assets/grassTilemap.png")
	if err != nil {
		log.Fatal(err)
	}
	softMask, _, err := ebitenutil.NewImageFromFile("assets/softMask.png")
	if err != nil {
		log.Fatal(err)
	}
	rockMask, _, err := ebitenutil.NewImageFromFile("assets/rockMask.png")
	if err != nil {
		log.Fatal(err)
	}
	grassMask, _, err := ebitenutil.NewImageFromFile("assets/grassMask.png")
	if err != nil {
		log.Fatal(err)
	}
	materialTypes, _, err := ebitenutil.NewImageFromFile("assets/materialTypes.png")
	if err != nil {
		log.Fatal(err)
	}
	tileSize := 16

	// Seperate materialTypes.png into a slice of individual texture
	materials := []*ebiten.Image{}
	materialsCount := int(materialTypes.Bounds().Dx() / tileSize)
	for i := range materialsCount {
		tempMaterial := materialTypes.SubImage(image.Rect(i*tileSize, 0, i*tileSize+tileSize, tileSize)).(*ebiten.Image)
		materials = append(materials, tempMaterial)
	}

	newDualGrid := dualgrid.NewDualGrid(16, 16, tileSize, 3)

	err = newDualGrid.AddMaterialFromMask(materials[0], rockMask, dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	err = newDualGrid.AddMaterialFromMask(materials[1], rockMask, dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	err = newDualGrid.AddMaterialFromMask(
		materials[2],
		grassMask,
		dualgrid.VarientMap{
			3:  {17},
			5:  {16},
			10: {19},
			12: {18},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	err = newDualGrid.AddMaterialFromMask(materials[3], softMask, dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	err = newDualGrid.AddMaterialFromTilemap(grassTilemap, dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}

	game := &Game{
		DualGrid: newDualGrid,
	}

	worldWidth = 30 * newDualGrid.TileSize
	worldHeight = 20 * newDualGrid.TileSize

	// Center Camera on grid center
	cameraX = -(worldWidth - ((newDualGrid.GridWidth + 1) * newDualGrid.TileSize)) / 2
	cameraY = -(worldHeight - ((newDualGrid.GridHeight + 1) * newDualGrid.TileSize)) / 2

	dualGridImage = ebiten.NewImage(worldWidth, worldHeight)

	fmt.Print("EbitenDualGrid Info:\n",
		"  MouseWheel scrolls trough available materials\n",
		"  Left/Right Click Destroy and place the selected material\n",
		"  Middle mouse to pan around\n",
		"  \"R\" reset the grid with the selected material as the default\n",
		"  \"G\" Display the grid\n",
		"  \"C\" Display the grid true values\n",
		"  \"M\" Display the computed materials\n",
	)

	ebiten.SetWindowSize(worldWidth*2, worldHeight*2)
	ebiten.SetWindowTitle("DualGrid")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
