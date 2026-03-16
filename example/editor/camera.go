package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Camera struct {
	X, Y         int
	Zoom         float64
	dragX, dragY int
	Draging      bool
}

func NewCamera() Camera {
	return Camera{Zoom: 1.0}
}

// CenterOn sets the camera offset to center a world of size (gridW, gridH) on screen.
func (c *Camera) CenterOn(screenW, screenH, gridW, gridH int) {
	c.X = -(screenW - gridW) / 2
	c.Y = -(screenH - gridH) / 2
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
			c.X -= int(float64(newX-c.dragX) / c.Zoom)
			c.Y -= int(float64(newY-c.dragY) / c.Zoom)
			c.dragX, c.dragY = newX, newY
			c.Draging = true
		}
	}

	// Zoom
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		c.ZoomBy(1)
		updateDualGridImage = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		c.ZoomBy(-1)
		updateDualGridImage = true
	}
}

// ZoomBy adjusts the zoom level by delta, rounded to the nearest integer and clamped to [1, 4].
func (c *Camera) ZoomBy(delta float64) {
	c.Zoom = math.Round(c.Zoom + delta)
	if c.Zoom < 1 {
		c.Zoom = 1
	}
	if c.Zoom > 4 {
		c.Zoom = 4
	}
}

// ScreenToWorld converts a screen pixel to a world pixel coordinate.
func (c *Camera) ScreenToWorld(sx, sy, screenW, screenH int) (int, int) {
	wx := int((float64(sx)-float64(screenW)/2)/c.Zoom+float64(screenW)/2) + c.X
	wy := int((float64(sy)-float64(screenH)/2)/c.Zoom+float64(screenH)/2) + c.Y
	return wx, wy
}

// WorldToScreen converts a world pixel to a screen pixel coordinate.
func (c *Camera) WorldToScreen(wx, wy, screenW, screenH int) (float32, float32) {
	sx := (float64(wx-c.X)-float64(screenW)/2)*c.Zoom + float64(screenW)/2
	sy := (float64(wy-c.Y)-float64(screenH)/2)*c.Zoom + float64(screenH)/2
	return float32(sx), float32(sy)
}

// DrawImageOpts returns DrawImageOptions to render the world image to screen zoomed about the screen center.
func (c *Camera) DrawImageOpts(screenW, screenH int) *ebiten.DrawImageOptions {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(-float64(screenW)/2, -float64(screenH)/2)
	opts.GeoM.Scale(c.Zoom, c.Zoom)
	opts.GeoM.Translate(float64(screenW)/2, float64(screenH)/2)
	return opts
}
