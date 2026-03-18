package dualgrid

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
func (g Grid) OutlineRect(x, y, w, h int, value TileType) {
	for dx := range w {
		g.Cells[x+dx][y] = value
		g.Cells[x+dx][y+h-1] = value
	}
	for dy := range h {
		g.Cells[x][y+dy] = value
		g.Cells[x+w-1][y+dy] = value
	}
}
