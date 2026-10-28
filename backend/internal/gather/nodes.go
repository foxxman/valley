package gather

import "game/backend/internal/mapstate"

// handDamage is one bare-handed blow. A tool later replaces this number.
const handDamage = 1

var grassTile = "grass"

type drop struct {
	Slug     string
	Quantity int
}

type node struct {
	Max   int
	Loot  []drop
	Clear mapstate.Clear
}

func breakable(kind string) (node, bool) {
	nodes := map[string]node{
		"tree": {
			Max:   10,
			Loot:  []drop{{Slug: "wood", Quantity: 4}},
			Clear: mapstate.Clear{Layer: mapstate.LayerTree},
		},
		"stone": {
			Max:   20,
			Loot:  []drop{{Slug: "stone", Quantity: 5}},
			Clear: mapstate.Clear{Layer: mapstate.LayerTerrain, Value: &grassTile},
		},
	}
	found, ok := nodes[kind]
	return found, ok
}
