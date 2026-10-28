package mapstate

// Layers stored for every game in one table. A missing row means the world
// seed still owns that layer. A stored NULL means the layer was cleared.
const (
	LayerTerrain    = "terrain"
	LayerTree       = "tree"
	LayerFlora      = "flora"
	LayerFeature    = "feature"
	LayerDurability = "durability"
)

var flowerSlugs = map[string]struct{}{
	"flower-0": {},
	"flower-1": {},
	"flower-2": {},
	"flower-3": {},
}

// IsFlower reports whether slug is one of the four collectible flowers.
func IsFlower(slug string) bool {
	_, ok := flowerSlugs[slug]
	return ok
}

// Cell is one map coordinate with only the layers that have a row.
// A nil layer value is an explicit clear.
type Cell struct {
	X      int
	Y      int
	Layers map[string]*string
}

type cellRow struct {
	X     int
	Y     int
	Layer string
	Value *string
}

type point struct {
	x int
	y int
}

func groupRows(rows []cellRow) []Cell {
	order := make([]point, 0, len(rows))
	grouped := make(map[point]*Cell, len(rows))
	for _, row := range rows {
		key := point{x: row.X, y: row.Y}
		cell := grouped[key]
		if cell == nil {
			cell = &Cell{X: row.X, Y: row.Y, Layers: map[string]*string{}}
			grouped[key] = cell
			order = append(order, key)
		}
		cell.Layers[row.Layer] = row.Value
	}
	cells := make([]Cell, 0, len(order))
	for _, key := range order {
		cells = append(cells, *grouped[key])
	}
	return cells
}
