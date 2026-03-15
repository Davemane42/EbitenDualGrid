package dualgrid

import (
	"encoding/binary"
	"errors"
)

type Grid [][]TileType

func NewGrid(width, height int) Grid {
	grid := make([][]TileType, width)
	for x := range width {
		grid[x] = make([]TileType, height)
	}
	return grid
}

func NewGridWithValue(width, height int, value TileType) Grid {
	grid := make([][]TileType, width)
	for x := range width {
		grid[x] = make([]TileType, height)
		for y := range height {
			grid[x][y] = value
		}
	}
	return grid
}

// FillRect fills a rectangle on the grid with the given value.
// x, y is the top-left corner; w, h are width and height.
func (g Grid) FillRect(x, y, w, h int, value TileType) {
	for dx := range w {
		for dy := range h {
			g[x+dx][y+dy] = value
		}
	}
}

// OutlineRect draws the border of a rectangle on the grid with the given value.
// x, y is the top-left corner; w, h are width and height.
func (g Grid) OutlineREct(x, y, w, h int, value TileType) {
	for dx := range w {
		g[x+dx][y] = value
		g[x+dx][y+h-1] = value
	}
	for dy := range h {
		g[x][y+dy] = value
		g[x+w-1][y+dy] = value
	}
}

// Marshal encodes the grid to bytes.
// Format: [width uint32][height uint32][tiles...]
func (g Grid) Marshal() []byte {
	width := len(g)
	height := 0
	if width > 0 {
		height = len(g[0])
	}
	buf := make([]byte, 8+width*height)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(width))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(height))
	i := 8
	for x := range width {
		for y := range height {
			buf[i] = byte(g[x][y])
			i++
		}
	}
	return buf
}

// Unmarshal decodes a grid from bytes produced by Marshal.
func Unmarshal(data []byte) (Grid, error) {
	if len(data) < 8 {
		return nil, errors.New("dualgrid: data too short")
	}
	width := int(binary.LittleEndian.Uint32(data[0:4]))
	height := int(binary.LittleEndian.Uint32(data[4:8]))
	if len(data) < 8+width*height {
		return nil, errors.New("dualgrid: data truncated")
	}
	g := NewGrid(width, height)
	i := 8
	for x := range width {
		for y := range height {
			g[x][y] = TileType(data[i])
			i++
		}
	}
	return g, nil
}
