package main

import (
	"image"
	"image/color"
	"log"

	dualgrid "github.com/davemane42/EbitenDualGrid"
	assets "github.com/davemane42/EbitenDualGrid/example/assets"
	"github.com/hajimehoshi/ebiten/v2"
)

type Mode interface {
	GetName() string
	Setup() dualgrid.DualGrid
}

var modes = []Mode{DungeonMode{}, NatureMode{}}

// DungeonMode

type DungeonMode struct{}

func (DungeonMode) GetName() string { return "Dungeon" }

func (DungeonMode) Setup() dualgrid.DualGrid {
	dg := dualgrid.NewDualGrid(16, 16, tileSize, 2)
	dg.WorldGrid.FillRect(4, 4, 8, 8, 0)  // main room
	dg.WorldGrid.FillRect(4, 4, 8, 1, 1)  // main room wall
	dg.WorldGrid.FillRect(6, 2, 4, 12, 0) // vertical corridor
	dg.WorldGrid.FillRect(6, 2, 4, 1, 1)  // vertical corridor wall
	dg.WorldGrid.FillRect(2, 6, 12, 4, 0) // horizontal corridor
	dg.WorldGrid.FillRect(2, 6, 2, 1, 1)  // horizontal corridor left wall
	dg.WorldGrid.FillRect(12, 6, 2, 1, 1) // horizontal corridor right wall

	floorMat, err := dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["floor"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	wallMat, err := dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["wall"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	topWallMat, err := dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["topWall"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}

	dg.AddMaterial(floorMat)
	dg.AddMaterial(wallMat)
	dg.AddMaterial(topWallMat)

	materialsColors = []color.Color{
		color.RGBA{139, 155, 180, 255}, // Gray
		color.RGBA{115, 62, 57, 255},   // Brown
		color.RGBA{254, 174, 52, 255},  // Orange
	}
	return dg
}

// NatureMode

type NatureMode struct{}

func (NatureMode) GetName() string { return "Nature" }

func (NatureMode) Setup() dualgrid.DualGrid {
	mats := make([]*ebiten.Image, assets.Images["materialTypes"].Bounds().Dx()/tileSize)
	for i := range mats {
		mats[i] = assets.Images["materialTypes"].SubImage(image.Rect(i*tileSize, 0, i*tileSize+tileSize, tileSize)).(*ebiten.Image)
	}

	dg := dualgrid.NewDualGrid(16, 16, tileSize, 3)

	rockMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[0], assets.Images["rockMask"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	dirtMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[1], assets.Images["rockMask"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	darkGrassMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[2], assets.Images["grassMask"], dualgrid.VarientMap{
		3:  {17},
		5:  {16},
		10: {19},
		12: {18},
	})
	if err != nil {
		log.Fatal(err)
	}
	grassMat, err := dualgrid.NewMaterialFromMask(tileSize, mats[3], assets.Images["softMask"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}
	greenGrass, err := dualgrid.NewMaterialFromTilemap(tileSize, assets.Images["grassTilemap"], dualgrid.VarientMap{})
	if err != nil {
		log.Fatal(err)
	}

	dg.AddMaterial(rockMat)
	dg.AddMaterial(dirtMat)
	dg.AddMaterial(darkGrassMat)
	dg.AddMaterial(grassMat)
	dg.AddMaterial(greenGrass)

	materialsColors = []color.Color{
		color.RGBA{139, 155, 180, 255}, // Gray
		color.RGBA{115, 62, 57, 255},   // Brown
		color.RGBA{38, 92, 66, 255},    // Dark Green
		color.RGBA{62, 137, 72, 255},   // Green
		color.RGBA{254, 174, 52, 255},  // Orange
	}
	return dg
}
