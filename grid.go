package dualgrid

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
