package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"runtime"

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

	updateDualGridImage = true
	dualGridImage       *ebiten.Image

	selectedMaterial int
	materialsColors  []color.Color

	tileSize    = 16
	worldWidth  = 30 * tileSize
	worldHeight = 20 * tileSize

	currentMode    Mode
	currentModeIdx int

	sourceDir = func() string {
		_, f, _, _ := runtime.Caller(0)
		return filepath.Dir(f)
	}()
)

type Game struct {
	DualGrid dualgrid.DualGrid
	Camera   Camera
}

func (g *Game) switchMode(mode Mode) {
	currentMode = mode
	selectedMaterial = 0
	g.DualGrid = mode.Setup()
	gridW := (g.DualGrid.WorldGrid.Width + 1) * g.DualGrid.TileSize
	gridH := (g.DualGrid.WorldGrid.Height + 1) * g.DualGrid.TileSize
	g.Camera.CenterOn(worldWidth, worldHeight, gridW, gridH)
	updateDualGridImage = true
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()
	wx, wy := g.Camera.ScreenToWorld(mx, my, worldWidth, worldHeight)
	MouseWorldX := (wx - g.DualGrid.TileSize/2) / g.DualGrid.TileSize
	MouseWorldY := (wy - g.DualGrid.TileSize/2) / g.DualGrid.TileSize

	g.Camera.Update()
	if g.Camera.Draging {
		updateDualGridImage = true
	}

	// Place and destroy Material
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.DualGrid.IsInbound(MouseWorldX, MouseWorldY) {
		g.DualGrid.WorldGrid.Cells[MouseWorldX][MouseWorldY] = dualgrid.TileType(selectedMaterial)
		updateDualGridImage = true
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) && g.DualGrid.IsInbound(MouseWorldX, MouseWorldY) {
		g.DualGrid.WorldGrid.Cells[MouseWorldX][MouseWorldY] = g.DualGrid.DefaultMaterial
		updateDualGridImage = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.DualGrid.DefaultMaterial = dualgrid.TileType(selectedMaterial)
		g.DualGrid.WorldGrid = dualgrid.NewGridWithValue(g.DualGrid.WorldGrid.Width, g.DualGrid.WorldGrid.Height, g.DualGrid.DefaultMaterial)
		updateDualGridImage = true
	}

	// Switch mode
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		currentModeIdx = (currentModeIdx + 1) % len(modes)
		g.switchMode(modes[currentModeIdx])
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
		filename := filepath.Join(sourceDir, currentMode.GetName()+".bin")
		data := g.DualGrid.WorldGrid.Marshal()
		if err := os.WriteFile(filename, data, 0644); err != nil {
			log.Println("save failed:", err)
		} else {
			log.Println("grid saved to", filename)
		}
	}

	// Load
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		filename := filepath.Join(sourceDir, currentMode.GetName()+".bin")
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
		g.DualGrid.DrawTo(dualGridImage, g.Camera.X, g.Camera.Y)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{24, 20, 37, 255})
	screen.DrawImage(dualGridImage, g.Camera.DrawImageOpts(worldWidth, worldHeight))

	// Debug Draw
	if showGrid || showCorners {
		var xPos, yPos int
		var xStart, yStart = g.DualGrid.TileSize / 2, g.DualGrid.TileSize / 2
		var tl, tr, bl, br dualgrid.TileType
		scaledTile := float32(g.DualGrid.TileSize) * float32(g.Camera.Zoom)
		co := 2 * float32(g.Camera.Zoom)
		for x := range g.DualGrid.WorldGrid.Width + 1 {
			xPos = xStart + x*g.DualGrid.TileSize
			for y := range g.DualGrid.WorldGrid.Height + 1 {
				yPos = yStart + y*g.DualGrid.TileSize
				sx, sy := g.Camera.WorldToScreen(xPos, yPos, worldWidth, worldHeight)
				// Display grid
				if showGrid && x < g.DualGrid.WorldGrid.Width && y < g.DualGrid.WorldGrid.Height {
					vector.StrokeRect(
						screen,
						sx+1, sy+1,
						scaledTile-1, scaledTile-1,
						1,
						materialsColors[g.DualGrid.WorldGrid.Cells[x][y]],
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
						tl = g.DualGrid.WorldGrid.Cells[x-1][y-1]
					}
					if x < g.DualGrid.WorldGrid.Width && y >= 1 {
						tr = g.DualGrid.WorldGrid.Cells[x][y-1]
					}
					if x >= 1 && y < g.DualGrid.WorldGrid.Height {
						bl = g.DualGrid.WorldGrid.Cells[x-1][y]
					}
					if x < g.DualGrid.WorldGrid.Width && y < g.DualGrid.WorldGrid.Height {
						br = g.DualGrid.WorldGrid.Cells[x][y]
					}

					vector.DrawFilledCircle(screen, sx-co, sy-co, 1, materialsColors[tl], false)
					vector.DrawFilledCircle(screen, sx+co, sy-co, 1, materialsColors[tr], false)
					vector.DrawFilledCircle(screen, sx-co, sy+co, 1, materialsColors[bl], false)
					vector.DrawFilledCircle(screen, sx+co, sy+co, 1, materialsColors[br], false)
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
	modeText := "Mode: " + currentMode.GetName() + " [Tab]"
	ebitenutil.DebugPrintAt(screen, modeText, screen.Bounds().Dx()-len(modeText)*6-8, screen.Bounds().Dy()-12-8)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return worldWidth, worldHeight
}

func main() {
	game := &Game{Camera: NewCamera()}
	dualGridImage = ebiten.NewImage(worldWidth, worldHeight)

	game.switchMode(modes[0])

	fmt.Print("EbitenDualGrid Info:\n",
		"  Tab              Switch between Dungeon and Nature mode\n",
		"  1-9              Select material by number\n",
		"  MouseWheel       Scroll through available materials\n",
		"  Left Click       Place selected material\n",
		"  Right Click      Erase (place default material)\n",
		"  Middle Mouse     Pan around\n",
		"  PageUp/PageDown  Zoom in/out\n",
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
