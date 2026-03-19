package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	dualgrid "github.com/davemane42/EbitenDualGrid"
	assets "github.com/davemane42/EbitenDualGrid/example/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	tileSize   = 16
	gridWidth  = 200
	gridHeight = 200
	screenW    = 640
	screenH    = 480

	warmupFrames = 10
	benchFrames  = 200
)

// totalExpectedFrames is the estimated total Update() calls for the full benchmark.
// warmup(10) + baseline(200) + 9 per-frame phases(9*200) + 3 batch phases(3)
// + 3 material phases(3) + 4 scale sizes * (1 setup + 200 drawto + 200 view)
const totalExpectedFrames = warmupFrames + benchFrames + 9*benchFrames + 3 + 3 + 4*(1+2*benchFrames)

const (
	MatRock      dualgrid.TileType = 0
	MatDarkRock  dualgrid.TileType = 1
	MatDarkGrass dualgrid.TileType = 2
	MatGrass     dualgrid.TileType = 3
	MatFlowers   dualgrid.TileType = 4
)

var out io.Writer

// Material texture images, extracted once in main and reused by benchmarks
var matTextures []*ebiten.Image

type BenchGame struct {
	dg     dualgrid.DualGrid
	canvas *ebiten.Image
	frame  int
	phase  string

	// Memory tracking
	memBefore   runtime.MemStats
	baselineB   uint64
	baselineA   uint64
	hasBaseline bool

	// Timing tracking
	elapsed    time.Duration
	baselineNs int64

	// Scaling benchmark state
	scaleSizes  []int
	scaleIdx    int
	scaleDg     dualgrid.DualGrid
	scaleCanvas *ebiten.Image

	// Progress tracking
	totalFrame int
	phaseName  string
}

func setupDualGrid() dualgrid.DualGrid {
	dg := dualgrid.NewDualGrid(gridWidth, gridHeight, tileSize, MatGrass)

	rockMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[0], assets.Images["rockMask"], dualgrid.VarientMap{})
	dirtMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[1], assets.Images["rockMask"], dualgrid.VarientMap{})
	darkGrassMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[2], assets.Images["grassMask"], dualgrid.VarientMap{
		3:  {17},
		5:  {16},
		10: {19},
		12: {18},
	})
	grassMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[3], assets.Images["softMask"], dualgrid.VarientMap{})
	greenGrass, _ := dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["grassTilemap"], dualgrid.VarientMap{})

	dg.AddMaterial(rockMat)
	dg.AddMaterial(dirtMat)
	dg.AddMaterial(darkGrassMat)
	dg.AddMaterial(grassMat)
	dg.AddMaterial(greenGrass)

	fillPattern(&dg, gridWidth, gridHeight)
	return dg
}

// fillPattern populates a grid with diagonal bands that use all materials
func fillPattern(dg *dualgrid.DualGrid, w, h int) {
	for x := range w {
		for y := range h {
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
}

// setupScaleDualGrid creates a DualGrid at the given size with all 5 materials
func setupScaleDualGrid(size int) dualgrid.DualGrid {
	dg := dualgrid.NewDualGrid(size, size, tileSize, MatGrass)

	rockMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[0], assets.Images["rockMask"], dualgrid.VarientMap{})
	dirtMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[1], assets.Images["rockMask"], dualgrid.VarientMap{})
	darkGrassMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[2], assets.Images["grassMask"], dualgrid.VarientMap{
		3:  {17},
		5:  {16},
		10: {19},
		12: {18},
	})
	grassMat, _ := dualgrid.NewMaterialFromMask(tileSize, matTextures[3], assets.Images["softMask"], dualgrid.VarientMap{})
	greenGrass, _ := dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["grassTilemap"], dualgrid.VarientMap{})

	dg.AddMaterial(rockMat)
	dg.AddMaterial(dirtMat)
	dg.AddMaterial(darkGrassMat)
	dg.AddMaterial(grassMat)
	dg.AddMaterial(greenGrass)

	fillPattern(&dg, size, size)
	return dg
}

func memDiff(before, after *runtime.MemStats) (bytes, allocs uint64) {
	return after.TotalAlloc - before.TotalAlloc, after.Mallocs - before.Mallocs
}

func (g *BenchGame) startBench(next string) {
	g.frame = 0
	g.phase = next
	g.elapsed = 0
	runtime.GC()
	runtime.ReadMemStats(&g.memBefore)
}

func (g *BenchGame) endBench(label string, ops int) {
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	totalB, totalA := memDiff(&g.memBefore, &after)
	n := uint64(ops)
	perOpB := totalB / n
	perOpA := totalA / n
	perOpNs := g.elapsed.Nanoseconds() / int64(ops)

	if !g.hasBaseline {
		fmt.Fprintf(out, "%-45s  %10s  per-op: %10s  %4d allocs\n",
			label, fmtDuration(perOpNs), fmtBytes(perOpB), perOpA)
		return
	}

	libB := perOpB - g.baselineB
	libA := perOpA - g.baselineA
	libNs := perOpNs - g.baselineNs
	if perOpB < g.baselineB {
		libB = 0
	}
	if perOpA < g.baselineA {
		libA = 0
	}
	if libNs < 0 {
		libNs = 0
	}
	fmt.Fprintf(out, "%-45s  %10s  per-op: %10s  %4d allocs  |  lib-only: %10s  %10s  %4d allocs\n",
		label, fmtDuration(perOpNs), fmtBytes(perOpB), perOpA, fmtDuration(libNs), fmtBytes(libB), libA)
}

// endBenchNoBaseline reports raw numbers without subtracting baseline (for one-shot benchmarks)
func (g *BenchGame) endBenchRaw(label string, ops int) {
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	totalB, totalA := memDiff(&g.memBefore, &after)
	n := uint64(ops)
	fmt.Fprintf(out, "%-45s  %10s  per-op: %10s  %4d allocs\n",
		label, fmtDuration(g.elapsed.Nanoseconds()/int64(ops)), fmtBytes(totalB/n), totalA/n)
}

func fmtDuration(ns int64) string {
	switch {
	case ns >= 1_000_000:
		return fmt.Sprintf("%.2f ms", float64(ns)/1e6)
	case ns >= 1_000:
		return fmt.Sprintf("%.2f us", float64(ns)/1e3)
	default:
		return fmt.Sprintf("%d ns", ns)
	}
}

func fmtBytes(b uint64) string {
	switch {
	case b >= 1_000_000_000:
		return fmt.Sprintf("%.2f GB", float64(b)/1e9)
	case b >= 1_000_000:
		return fmt.Sprintf("%.2f MB", float64(b)/1e6)
	case b >= 1_000:
		return fmt.Sprintf("%.2f KB", float64(b)/1e3)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func (g *BenchGame) Update() error {
	g.totalFrame++
	switch g.phase {

	case "warmup":
		g.dg.MarkDirty()
		g.dg.DrawTo(g.canvas, 0, 0)
		g.frame++
		if g.frame >= warmupFrames {
			fmt.Fprintf(out, "Grid: %dx%d (%d cells)  |  Materials: %d  |  Frames: %d\n",
				gridWidth, gridHeight, gridWidth*gridHeight, len(g.dg.Materials), benchFrames)
			fmt.Fprintf(out, "Viewport: %dx%d\n\n", screenW, screenH)
			g.phaseName = "Ebiten baseline"
			g.startBench("baseline")
		}

	case "baseline":
		t := time.Now()
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			totalB, totalA := memDiff(&g.memBefore, &after)
			g.baselineB = totalB / uint64(benchFrames)
			g.baselineA = totalA / uint64(benchFrames)
			g.baselineNs = g.elapsed.Nanoseconds() / int64(benchFrames)
			g.hasBaseline = true
			fmt.Fprintf(out, "%-45s  %10s  per-op: %10s  %4d allocs\n\n",
				fmt.Sprintf("Ebiten baseline (idle, %d frames)", benchFrames),
				fmtDuration(g.baselineNs), fmtBytes(g.baselineB), g.baselineA)
			g.phaseName = "DrawTo (full grid)"
			g.startBench("drawto")
		}

	case "drawto":
		t := time.Now()
		g.dg.DrawTo(g.canvas, 0, 0)
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench("DrawTo (full grid redraw)", benchFrames)
			g.phaseName = "Canvas (dirty)"
			g.startBench("canvas_dirty")
		}

	case "canvas_dirty":
		g.dg.MarkDirty()
		t := time.Now()
		g.dg.Canvas()
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench("Canvas (dirty each frame)", benchFrames)
			g.phaseName = "Canvas (cached)"
			g.startBench("canvas_clean")
		}

	case "canvas_clean":
		t := time.Now()
		g.dg.Canvas()
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench("Canvas (clean / cached)", benchFrames)
			g.phaseName = "ViewCanvas"
			g.startBench("viewcanvas")
		}

	case "viewcanvas":
		t := time.Now()
		g.dg.ViewCanvas(screenW, screenH, 100, 100)
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench("ViewCanvas (640x480 viewport)", benchFrames)
			g.phaseName = "RedrawRegion (1x1)"
			g.startBench("redraw_region")
		}

	case "redraw_region":
		g.dg.Canvas()
		t := time.Now()
		g.dg.RedrawCanvasRegion(gridWidth/2, gridHeight/2, 1, 1)
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench("RedrawCanvasRegion (1x1 tile)", benchFrames)
			g.phaseName = "RedrawRegion (8x8)"
			g.startBench("redraw_region_8x8")
		}

	case "redraw_region_8x8":
		t := time.Now()
		g.dg.RedrawCanvasRegion(gridWidth/2-4, gridHeight/2-4, 8, 8)
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench("RedrawCanvasRegion (8x8 tiles)", benchFrames)
			g.phaseName = "SetCell"
			g.startBench("setcell")
		}

	case "setcell":
		t := time.Now()
		for i := range 1000 {
			g.dg.SetCell(i%gridWidth, i/gridWidth%gridHeight, dualgrid.TileType(i%5))
		}
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench(fmt.Sprintf("SetCell (1000/frame x %d frames)", benchFrames), benchFrames)
			g.phaseName = "GetCell"
			g.startBench("getcell")
		}

	case "getcell":
		t := time.Now()
		for i := range 1000 {
			_ = g.dg.GetCell(i%gridWidth, i/gridWidth%gridHeight)
		}
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench(fmt.Sprintf("GetCell (1000/frame x %d frames)", benchFrames), benchFrames)
			g.phaseName = "FillRect"
			g.startBench("fillrect")
		}

	case "fillrect":
		t := time.Now()
		g.dg.WorldGrid.FillRect(10, 10, 50, 50, MatRock)
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			g.endBench("FillRect (50x50)", benchFrames)
			g.phaseName = "Marshal"
			g.startBench("marshal")
		}

	case "marshal":
		t := time.Now()
		for range benchFrames {
			_ = g.dg.Marshal()
		}
		g.elapsed += time.Since(t)
		g.endBench(fmt.Sprintf("Marshal (%d calls)", benchFrames), benchFrames)
		g.phaseName = "Unmarshal"
		g.startBench("unmarshal")

	case "unmarshal":
		data := g.dg.Marshal()
		runtime.GC()
		runtime.ReadMemStats(&g.memBefore)
		t := time.Now()
		for range benchFrames {
			_ = g.dg.Unmarshal(data, false)
		}
		g.elapsed += time.Since(t)
		g.endBench(fmt.Sprintf("Unmarshal (%d calls)", benchFrames), benchFrames)
		g.phaseName = "NewGridWithValue"
		g.startBench("newgrid")

	case "newgrid":
		t := time.Now()
		for range benchFrames {
			_ = dualgrid.NewGridWithValue(gridWidth, gridHeight, 0)
		}
		g.elapsed += time.Since(t)
		g.endBench(fmt.Sprintf("NewGridWithValue %dx%d (%d calls)", gridWidth, gridHeight, benchFrames), benchFrames)

		fmt.Fprintf(out, "\n--- Material Creation ---\n")
		g.phaseName = "NewMaterialFromTilemap"
		g.startBench("mat_tilemap")

	case "mat_tilemap":
		t := time.Now()
		for range benchFrames {
			dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["grassTilemap"], dualgrid.VarientMap{})
		}
		g.elapsed += time.Since(t)
		g.endBenchRaw(fmt.Sprintf("NewMaterialFromTilemap (%d calls)", benchFrames), benchFrames)
		g.phaseName = "NewMaterialFromMask"
		g.startBench("mat_mask")

	case "mat_mask":
		t := time.Now()
		for range benchFrames {
			dualgrid.NewMaterialFromMask(tileSize, matTextures[0], assets.Images["rockMask"], dualgrid.VarientMap{})
		}
		g.elapsed += time.Since(t)
		g.endBenchRaw(fmt.Sprintf("NewMaterialFromMask (%d calls)", benchFrames), benchFrames)
		g.phaseName = "NewMaterialFromMask+variants"
		g.startBench("mat_mask_var")

	case "mat_mask_var":
		vm := dualgrid.VarientMap{
			3:  {17},
			5:  {16},
			10: {19},
			12: {18},
		}
		t := time.Now()
		for range benchFrames {
			dualgrid.NewMaterialFromMask(tileSize, matTextures[2], assets.Images["grassMask"], vm)
		}
		g.elapsed += time.Since(t)
		g.endBenchRaw(fmt.Sprintf("NewMaterialFromMask+variants (%d calls)", benchFrames), benchFrames)

		fmt.Fprintf(out, "\n--- Scaling (DrawTo full grid, 5 materials) ---\n")
		g.scaleSizes = []int{50, 100, 200, 500}
		g.scaleIdx = 0
		g.phaseName = fmt.Sprintf("Scaling: setup (%dx%d)", g.scaleSizes[0], g.scaleSizes[0])
		g.phase = "scale_setup"

	case "scale_setup":
		size := g.scaleSizes[g.scaleIdx]
		g.scaleDg = setupScaleDualGrid(size)
		g.scaleCanvas = ebiten.NewImage((size+1)*tileSize, (size+1)*tileSize)
		g.scaleDg.DrawTo(g.scaleCanvas, 0, 0)
		g.phaseName = fmt.Sprintf("Scaling: DrawTo (%dx%d)", size, size)
		g.startBench("scale_drawto")

	case "scale_drawto":
		t := time.Now()
		g.scaleDg.DrawTo(g.scaleCanvas, 0, 0)
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			size := g.scaleSizes[g.scaleIdx]
			g.endBench(fmt.Sprintf("DrawTo %dx%d (%d cells)", size, size, size*size), benchFrames)
			g.phaseName = fmt.Sprintf("Scaling: ViewCanvas (%dx%d)", size, size)
			g.startBench("scale_view")
		}

	case "scale_view":
		t := time.Now()
		g.scaleDg.ViewCanvas(screenW, screenH, 0, 0)
		g.elapsed += time.Since(t)
		g.frame++
		if g.frame >= benchFrames {
			size := g.scaleSizes[g.scaleIdx]
			g.endBench(fmt.Sprintf("ViewCanvas %dx%d (640x480 vp)", size, size), benchFrames)

			g.scaleIdx++
			if g.scaleIdx < len(g.scaleSizes) {
				nextSize := g.scaleSizes[g.scaleIdx]
				g.phaseName = fmt.Sprintf("Scaling: setup (%dx%d)", nextSize, nextSize)
				g.phase = "scale_setup"
			} else {
				fmt.Fprintln(out)
				var m runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&m)
				fmt.Fprintf(out, "=== Heap Snapshot ===\n")
				fmt.Fprintf(out, "  HeapAlloc:   %s\n", fmtBytes(m.HeapAlloc))
				fmt.Fprintf(out, "  HeapInuse:   %s\n", fmtBytes(m.HeapInuse))
				fmt.Fprintf(out, "  HeapObjects: %d\n", m.HeapObjects)
				fmt.Fprintf(out, "  NumGC:       %d\n", m.NumGC)
				g.phaseName = "Done"
				g.phase = "done"
			}
		}

	case "done":
		return ebiten.Termination
	}

	return nil
}

func (g *BenchGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 40, 255})

	progress := float32(g.totalFrame) / float32(totalExpectedFrames)
	if progress > 1 {
		progress = 1
	}

	// Bar dimensions
	barX := float32(20)
	barY := float32(screenH/2 - 10)
	barW := float32(screenW - 40)
	barH := float32(20)

	// Background
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{60, 60, 80, 255}, false)
	// Fill
	vector.DrawFilledRect(screen, barX, barY, barW*progress, barH, color.RGBA{100, 200, 120, 255}, false)
	// Border
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, color.RGBA{180, 180, 200, 255}, false)

	pct := int(progress * 100)
	tps := ebiten.TPS()
	elapsedSec := g.totalFrame / tps
	estimatedSec := totalExpectedFrames / tps
	label := fmt.Sprintf("%s  [%d%%]  %02d:%02d / %02d:%02d", g.phaseName, pct, elapsedSec/60, elapsedSec%60, estimatedSec/60, estimatedSec%60)
	ebitenutil.DebugPrintAt(screen, label, int(barX), int(barY)-16)
}
func (g *BenchGame) Layout(_, _ int) (int, int) { return screenW, screenH }

func main() {
	runtime.GOMAXPROCS(1)

	_, srcFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(srcFile)
	filename := filepath.Join(dir, time.Now().Format("2006-01-02_15-04-05")+".txt")
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	out = io.MultiWriter(os.Stdout, f)

	// Extract material texture swatches once
	matTextures = make([]*ebiten.Image, assets.Images["materialTypes"].Bounds().Dx()/tileSize)
	for i := range matTextures {
		matTextures[i] = assets.Images["materialTypes"].SubImage(
			image.Rect(i*tileSize, 0, i*tileSize+tileSize, tileSize),
		).(*ebiten.Image)
	}

	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("DualGrid Benchmark")

	// Measure setup
	var mBefore, mAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mBefore)
	tSetup := time.Now()
	dg := setupDualGrid()
	setupDur := time.Since(tSetup)
	runtime.ReadMemStats(&mAfter)
	bytes, allocs := memDiff(&mBefore, &mAfter)
	fmt.Fprintf(out, "%-45s  %10s  %10s  %6d allocs\n\n",
		fmt.Sprintf("Setup (%dx%d, 5 materials)", gridWidth, gridHeight),
		fmtDuration(setupDur.Nanoseconds()), fmtBytes(bytes), allocs)

	canvas := ebiten.NewImage((gridWidth+1)*tileSize, (gridHeight+1)*tileSize)

	game := &BenchGame{
		dg:        dg,
		canvas:    canvas,
		phase:     "warmup",
		phaseName: "Warmup",
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
