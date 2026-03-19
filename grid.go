package dualgrid

type Grid struct {
	Width, Height int
	Cells         []TileType
}

func NewGrid(width, height int) Grid {
	return Grid{Width: width, Height: height, Cells: make([]TileType, width*height)}
}

func NewGridWithValue(width, height int, value TileType) Grid {
	cells := make([]TileType, width*height)
	if value != 0 {
		for i := range cells {
			cells[i] = value
		}
	}
	return Grid{Width: width, Height: height, Cells: cells}
}

// // Set sets the TileType at the given cell.
// func (g *Grid) Set(x, y int, value TileType) {
// 	g.Cells[x*g.Height+y] = value
// }

// // Get returns the TileType at the given cell.
// func (g *Grid) Get(x, y int) TileType {
// 	return g.Cells[x*g.Height+y]
// }

// FillRect fills a rectangle on the grid with the given value.
// x, y is the top-left corner; w, h are width and height.
func (g *Grid) FillRect(x, y, w, h int, value TileType) {
	for dx := range w {
		for dy := range h {
			g.Cells[(x+dx)*g.Height+(y+dy)] = value
		}
	}
}

// OutlineRect draws the border of a rectangle on the grid with the given value.
// x, y is the top-left corner; w, h are width and height.
func (g *Grid) OutlineRect(x, y, w, h int, value TileType) {
	for dx := range w {
		g.Cells[(x+dx)*g.Height+y] = value
		g.Cells[(x+dx)*g.Height+(y+h-1)] = value
	}
	for dy := range h {
		g.Cells[x*g.Height+(y+dy)] = value
		g.Cells[(x+w-1)*g.Height+(y+dy)] = value
	}
}
