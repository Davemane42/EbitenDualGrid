package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Camera struct {
	X, Y          float64
	Scale         float64
	Width, Height int
	dragX, dragY  int
	Draging       bool
}

func NewCamera(Scale float64, Width, Height int) Camera {
	return Camera{
		Scale:  Scale,
		Width:  Width,
		Height: Height,
	}
}

func (c *Camera) Update() {
	// Pan
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		c.dragX, c.dragY = ebiten.CursorPosition()
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		newX, newY := ebiten.CursorPosition()
		if newX == c.dragX && newY == c.dragY {
			c.Draging = false
		} else {
			c.X -= float64(newX - c.dragX)
			c.Y -= float64(newY - c.dragY)
			c.dragX, c.dragY = newX, newY
			c.Draging = true
		}
	}

	// Scale
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		c.SetScale(math.Min(c.Scale+1, 4))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		c.SetScale(math.Max(c.Scale-1, 1))
	}
}

func (c *Camera) LookAt(worldX, worldY float64) {
	c.X = worldX * c.Scale
	c.Y = worldY * c.Scale
}

func (c *Camera) SetScale(newScale float64) {
	worldX, worldY := c.X/c.Scale, c.Y/c.Scale
	c.Scale = newScale
	c.X = worldX * c.Scale
	c.Y = worldY * c.Scale
}

// ScreenToWorld converts a screen pixel to a world pixel coordinate.
func (c Camera) ScreenToWorld(screenX, screenY int) (worldX, worldY float64) {
	worldX = (float64(screenX-c.Width/2) + c.X) / c.Scale
	worldY = (float64(screenY-c.Height/2) + c.Y) / c.Scale

	return worldX, worldY
}

// WorldToScreen converts a world pixel to a screen pixel coordinate.
func (c Camera) WorldToScreen(worldX, worldY float64) (screenX, screenY int) {
	screenX = int((worldX-c.X/c.Scale)*c.Scale + float64(c.Width/2))
	screenY = int((worldY-c.Y/c.Scale)*c.Scale + float64(c.Height/2))

	return screenX, screenY
}

func (c Camera) ApplyTransforms(opts *ebiten.DrawImageOptions, worldX, worldY float64) {
	opts.GeoM.Translate(worldX-c.X/c.Scale, worldY-c.Y/c.Scale)
	opts.GeoM.Scale(c.Scale, c.Scale)
	opts.GeoM.Translate(float64(c.Width)/2, float64(c.Height)/2)
}

func (c Camera) GetCursorWorldCoords() (float64, float64) {
	return c.ScreenToWorld(ebiten.CursorPosition())
}
