# EbitenDualGrid
Basic implementation of a dualgrid autotiling system in the Ebiten engine.

Ressources:
- [Basic Explanation](https://youtu.be/buKQjkad2I0?t=220)
- [Bitmask indexing](https://www.lexaloffle.com/bbs/?tid=143710)

## Example
<img src="./example/example.png" alt="drawing" width="300px"/>

```bash
# Editor with pan and zoom
go run .\example\editor\.

# Simple demo
go run .\example\simple\.
```

## Usage

**1. Load your images** (look into [./example/assets](./example/assets) for sprites)
```go
// 4x4 tilemap (64x64 for tileSize 16)
grassTilemapImage, _, err := ebitenutil.NewImageFromFile("grass.png")

// single tile (16x16)
rockTextureImage, _, err := ebitenutil.NewImageFromFile("rock.png")

// 4x4 mask (64x64 for tileSize 16)
rockMaskImage, _, err := ebitenutil.NewImageFromFile("mask.png")
```

**2. Create the DualGrid**
```go
// NewDualGrid(width, height, tileSize int, defaultMaterial TileType)
dg := dualgrid.NewDualGrid(20, 15, 16, 0)
```

**3. Create materials**
```go
// NewMaterialFromTilemap takes a 4x4 tilemap and builds a Material.
grassMat, err := dualgrid.NewMaterialFromTilemap(tileSize, grassTilemapImage, dualgrid.VarientMap{})

// NewMaterialFromMask takes a base texture and a 4x4 mask and builds a Material.
rockMat, err := dualgrid.NewMaterialFromMask(tileSize, rockTextureImage, rockMaskImage, dualgrid.VarientMap{})
```

**4. Add materials**
```go
dg.AddMaterial(grassMat) // index 0
dg.AddMaterial(rockMat)  // index 1
```

**5. Paint cells by setting their material**
```go
// FillRect(x, y, w, h int, value TileType)
dg.WorldGrid.FillRect(2, 2, 16, 11, 1)
```

**6. Render to screen**
```go
// left/top are world-space pixel offsets for scrolling
// DrawTo(img *ebiten.Image, left, top int)
dg.DrawTo(screen, 0, 0) 
```