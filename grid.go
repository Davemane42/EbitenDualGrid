package dualgrid

import (
	"encoding/binary"
	"errors"
)

var (
	GridSizeError      = errors.New("Grid data is too short")
	GridTruncatedError = errors.New("Grid data is truncated")
)

type Grid struct {
	Width, Height int
	Cells         [][]TileType
}

func NewGrid(width, height int) Grid {
	cells := make([][]TileType, width)
	for x := range width {
		cells[x] = make([]TileType, height)
	}
	return Grid{Width: width, Height: height, Cells: cells}
}

func NewGridWithValue(width, height int, value TileType) Grid {
	cells := make([][]TileType, width)
	for x := range width {
		cells[x] = make([]TileType, height)
		for y := range height {
			cells[x][y] = value
		}
	}
	return Grid{Width: width, Height: height, Cells: cells}
}

// FillRect fills a rectangle on the grid with the given value.
// x, y is the top-left corner; w, h are width and height.
func (g Grid) FillRect(x, y, w, h int, value TileType) {
	for dx := range w {
		for dy := range h {
			g.Cells[x+dx][y+dy] = value
		}
	}
}

// OutlineRect draws the border of a rectangle on the grid with the given value.
// x, y is the top-left corner; w, h are width and height.
func (g Grid) OutlineREct(x, y, w, h int, value TileType) {
	for dx := range w {
		g.Cells[x+dx][y] = value
		g.Cells[x+dx][y+h-1] = value
	}
	for dy := range h {
		g.Cells[x][y+dy] = value
		g.Cells[x+w-1][y+dy] = value
	}
}

// Marshal encodes the grid to bytes.
//
//	Format: [width uint32][height uint32][tiles...]
func (g Grid) Marshal() []byte {
	buf := make([]byte, 8+g.Width*g.Height)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(g.Width))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(g.Height))
	i := 8
	for x := range g.Width {
		for y := range g.Height {
			buf[i] = byte(g.Cells[x][y])
			i++
		}
	}
	return buf
}

// Unmarshal decodes a grid from bytes produced by Marshal.
func Unmarshal(data []byte) (Grid, error) {
	if len(data) < 8 {
		return Grid{}, GridSizeError
	}
	width := int(binary.LittleEndian.Uint32(data[0:4]))
	height := int(binary.LittleEndian.Uint32(data[4:8]))
	if len(data) < 8+width*height {
		return Grid{}, GridTruncatedError
	}
	g := NewGrid(width, height)
	i := 8
	for x := range width {
		for y := range height {
			g.Cells[x][y] = TileType(data[i])
			i++
		}
	}
	return g, nil
}
