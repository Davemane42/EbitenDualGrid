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
	}

	worldWidth  int
	worldHeight int
)

type Game struct {
	DualGrid dualgrid.DualGrid
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()
	MouseWorldX := max(0, min((mx-int(g.DualGrid.TileSize/2))/g.DualGrid.TileSize, g.DualGrid.GridWidth-1))
	MouseWorldY := max(0, min((my-int(g.DualGrid.TileSize/2))/g.DualGrid.TileSize, g.DualGrid.GridHeight-1))

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.DualGrid.WorldGrid[MouseWorldX][MouseWorldY] = dualgrid.TileType(selectedMaterial)
		updateDualGridImage = true
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		g.DualGrid.WorldGrid[MouseWorldX][MouseWorldY] = g.DualGrid.DefaultMaterial
		updateDualGridImage = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.DualGrid.DefaultMaterial = dualgrid.TileType(selectedMaterial)
		g.DualGrid.WorldGrid = dualgrid.NewGridWithValue(g.DualGrid.GridWidth, g.DualGrid.GridHeight, g.DualGrid.DefaultMaterial)
		updateDualGridImage = true
	}

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
		g.DualGrid.DrawTo(dualGridImage)
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
						float32(xPos+1), float32(yPos+1),
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

					vector.DrawFilledCircle(screen, float32(xPos-cornerOffset), float32(yPos-cornerOffset), 1, materialsColors[tl], false)
					vector.DrawFilledCircle(screen, float32(xPos+cornerOffset), float32(yPos-cornerOffset), 1, materialsColors[tr], false)
					vector.DrawFilledCircle(screen, float32(xPos-cornerOffset), float32(yPos+cornerOffset), 1, materialsColors[bl], false)
					vector.DrawFilledCircle(screen, float32(xPos+cornerOffset), float32(yPos+cornerOffset), 1, materialsColors[br], false)
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
			for x, img := range mat {
				xPos = g.DualGrid.TileSize*x + x
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(xPos), float64(yPos))
				vector.DrawFilledRect(screen, float32(xPos), float32(yPos), float32(g.DualGrid.TileSize), float32(g.DualGrid.TileSize), color.Black, false)
				screen.DrawImage(img, opts)
			}
		}
	}

	// Current Texture Preview
	yPos := screen.Bounds().Dy() - g.DualGrid.TileSize - 1
	vector.StrokeRect(screen, 1, float32(yPos), float32(g.DualGrid.TileSize)+1, float32(g.DualGrid.TileSize)+1, 1, materialsColors[selectedMaterial], false)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(1, float64(yPos))
	screen.DrawImage(g.DualGrid.Materials[selectedMaterial][15], opts)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return worldWidth, worldHeight
}

func main() {
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
	newDualGrid.AddMaterial(materials[0], rockMask)  // Rock
	newDualGrid.AddMaterial(materials[1], rockMask)  // Dirt
	newDualGrid.AddMaterial(materials[2], grassMask) // DarkGrass
	newDualGrid.AddMaterial(materials[3], softMask)  // Grass

	game := &Game{
		DualGrid: newDualGrid,
	}

	worldWidth = (newDualGrid.GridWidth + 1) * newDualGrid.TileSize
	worldHeight = (newDualGrid.GridHeight + 1) * newDualGrid.TileSize

	dualGridImage = ebiten.NewImage(worldWidth, worldHeight)

	fmt.Print("EbitenDualGrid Info:\n",
		"  MouseWheel scrolls trough available materials\n",
		"  Left/Right Click Destroy and place the selected material\n",
		"  \"R\" reset the grid with the selected material as the default\n",
		"  \"G\" Display the grid\n",
		"  \"C\" Display the grid true values\n",
		"  \"M\" Display the computed materials\n",
	)

	ebiten.SetWindowSize(worldWidth*3, worldHeight*3)
	ebiten.SetWindowTitle("DualGrid")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
