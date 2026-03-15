package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"

	dualgrid "github.com/davemane42/EbitenDualGrid"
	assets "github.com/davemane42/EbitenDualGrid/example/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Mode int

const (
	ModeDungeon Mode = iota
	ModeNature
)

var (
	showCorners  bool
	showGrid     bool
	showTextures bool

	updateDualGridImage = true
	dualGridImage       *ebiten.Image

	selectedMaterial int
	materialsColors  []color.Color

	cursorX int
	cursorY int
	cameraX int
	cameraY int

	tileSize    = 16
	worldWidth  = 30 * tileSize
	worldHeight = 20 * tileSize

	currentMode     Mode
	currentModeName string
)

type Game struct {
	DualGrid dualgrid.DualGrid
}

func setupDungeonMode() dualgrid.DualGrid {
	dg := dualgrid.NewDualGrid(16, 16, tileSize, 2)
	dg.WorldGrid.FillRect(4, 4, 8, 8, 0)  // main room
	dg.WorldGrid.FillRect(4, 4, 8, 1, 1)  // main room wall
	dg.WorldGrid.FillRect(6, 2, 4, 12, 0) // vertical corridor
	dg.WorldGrid.FillRect(6, 2, 4, 1, 1)  // vertical corridor wall
	dg.WorldGrid.FillRect(2, 6, 12, 4, 0) // horizontal corridor
	dg.WorldGrid.FillRect(2, 6, 2, 1, 1)  // horizontal corridor left wall
	dg.WorldGrid.FillRect(12, 6, 2, 1, 1) // horizontal corridor right wall

	if err := dg.AddMaterialFromTilemap(assets.Images["floor"], dualgrid.VarientMap{}); err != nil {
		log.Fatal(err)
	}
	if err := dg.AddMaterialFromTilemap(assets.Images["wall"], dualgrid.VarientMap{}); err != nil {
		log.Fatal(err)
	}
	if err := dg.AddMaterialFromTilemap(assets.Images["topWall"], dualgrid.VarientMap{}); err != nil {
		log.Fatal(err)
	}

	materialsColors = []color.Color{
		color.RGBA{255, 255, 255, 255}, // White
		color.RGBA{139, 155, 180, 255}, // Gray
		color.RGBA{254, 174, 52, 255},  // Orange
	}
	currentModeName = "Dungeon"
	return dg
}

func setupNatureMode() dualgrid.DualGrid {
	mats := []*ebiten.Image{}
	for i := range assets.Images["materialTypes"].Bounds().Dx() / tileSize {
		mats = append(mats, assets.Images["materialTypes"].SubImage(image.Rect(i*tileSize, 0, i*tileSize+tileSize, tileSize)).(*ebiten.Image))
	}

	dg := dualgrid.NewDualGrid(16, 16, tileSize, 3)

	if err := dg.AddMaterialFromMask(mats[0], assets.Images["rockMask"], dualgrid.VarientMap{}); err != nil {
		log.Fatal(err)
	}
	if err := dg.AddMaterialFromMask(mats[1], assets.Images["softMask"], dualgrid.VarientMap{}); err != nil {
		log.Fatal(err)
	}
	if err := dg.AddMaterialFromMask(mats[2], assets.Images["grassMask"], dualgrid.VarientMap{
		3:  {17},
		5:  {16},
		10: {19},
		12: {18},
	}); err != nil {
		log.Fatal(err)
	}
	if err := dg.AddMaterialFromMask(mats[3], assets.Images["softMask"], dualgrid.VarientMap{}); err != nil {
		log.Fatal(err)
	}
	if err := dg.AddMaterialFromTilemap(assets.Images["grassTilemap"], dualgrid.VarientMap{}); err != nil {
		log.Fatal(err)
	}

	materialsColors = []color.Color{
		color.RGBA{139, 155, 180, 255}, // Gray
		color.RGBA{115, 62, 57, 255},   // Brown
		color.RGBA{38, 92, 66, 255},    // Dark Green
		color.RGBA{62, 137, 72, 255},   // Green
		color.RGBA{254, 174, 52, 255},  // Orange
	}
	currentModeName = "Nature"

	return dg
}

func (g *Game) switchMode(mode Mode) {
	currentMode = mode
	selectedMaterial = 0
	switch mode {
	case ModeDungeon:
		g.DualGrid = setupDungeonMode()
	case ModeNature:
		g.DualGrid = setupNatureMode()
	}
	// Reset camera to center
	cameraX = -(worldWidth - ((g.DualGrid.GridWidth + 1) * g.DualGrid.TileSize)) / 2
	cameraY = -(worldHeight - ((g.DualGrid.GridHeight + 1) * g.DualGrid.TileSize)) / 2
	updateDualGridImage = true
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

	// Switch mode
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.switchMode((currentMode + 1) % 2)
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
	for i, key := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
		if i < len(g.DualGrid.Materials) && inpututil.IsKeyJustPressed(key) {
			selectedMaterial = i
		}
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

	// Save
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		filename := "assets/" + currentModeName + ".bin"
		if err := os.WriteFile(filename, g.DualGrid.WorldGrid.Marshal(), 0644); err != nil {
			log.Println("save failed:", err)
		} else {
			log.Println("grid saved to", filename)
		}
	}

	// Load
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		filename := "assets/" + currentModeName + ".bin"
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Println("load failed:", err)
		} else {
			grid, err := dualgrid.Unmarshal(data)
			if err != nil {
				log.Println("load failed:", err)
			} else {
				g.DualGrid.WorldGrid = grid
				updateDualGridImage = true
				log.Println("grid loaded from", filename)
			}
		}
	}

	if updateDualGridImage {
		updateDualGridImage = false
		g.DualGrid.DrawTo(dualGridImage, cameraX, cameraY)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{24, 20, 37, 255})
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

	// Mode indicator (bottom right)
	modeText := "Mode: " + currentModeName + " [Tab]"
	ebitenutil.DebugPrintAt(screen, modeText, screen.Bounds().Dx()-len(modeText)*6-8, screen.Bounds().Dy()-12-8)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return worldWidth, worldHeight
}

func main() {
	game := &Game{}
	dualGridImage = ebiten.NewImage(worldWidth, worldHeight)

	game.switchMode(ModeDungeon)

	fmt.Print("EbitenDualGrid Info:\n",
		"  Tab              Switch between Dungeon and Nature mode\n",
		"  1-9              Select material by number\n",
		"  MouseWheel       Scroll through available materials\n",
		"  Left Click       Place selected material\n",
		"  Right Click      Erase (place default material)\n",
		"  Middle Mouse     Pan around\n",
		"  R                Reset grid with selected material as default\n",
		"  S                Save grid to grid.bin\n",
		"  L                Load grid from grid.bin\n",
		"  G                Display the grid\n",
		"  C                Display the grid true values\n",
		"  M                Display the computed materials\n",
	)

	ebiten.SetWindowSize(worldWidth*2, worldHeight*2)
	ebiten.SetWindowTitle("DualGrid")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
