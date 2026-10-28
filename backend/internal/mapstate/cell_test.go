package mapstate

import "testing"

func TestGroupRowsKeepsNullAndSplitsCells(t *testing.T) {
	grass := "grass"
	flower := "flower-1"
	rows := []cellRow{
		{X: 1, Y: 2, Layer: LayerFlora, Value: nil},
		{X: 1, Y: 2, Layer: LayerTree, Value: nil},
		{X: 4, Y: -3, Layer: LayerTerrain, Value: &grass},
		{X: 4, Y: -3, Layer: LayerFlora, Value: &flower},
	}
	cells := groupRows(rows)
	if len(cells) != 2 {
		t.Fatalf("cells %d", len(cells))
	}
	if cells[0].X != 1 || cells[0].Y != 2 || len(cells[0].Layers) != 2 {
		t.Fatalf("first %+v", cells[0])
	}
	if cells[0].Layers[LayerFlora] != nil || cells[0].Layers[LayerTree] != nil {
		t.Fatalf("clears %+v", cells[0].Layers)
	}
	if cells[1].X != 4 || cells[1].Y != -3 {
		t.Fatalf("second %+v", cells[1])
	}
	if cells[1].Layers[LayerTerrain] == nil || *cells[1].Layers[LayerTerrain] != grass {
		t.Fatalf("terrain %+v", cells[1].Layers)
	}
	if cells[1].Layers[LayerFlora] == nil || *cells[1].Layers[LayerFlora] != flower {
		t.Fatalf("flora %+v", cells[1].Layers)
	}
}

func TestIsFlower(t *testing.T) {
	for _, slug := range []string{"flower-0", "flower-1", "flower-2", "flower-3"} {
		if !IsFlower(slug) {
			t.Fatalf("%s", slug)
		}
	}
	if IsFlower("lily") || IsFlower("flower-4") || IsFlower("") {
		t.Fatal("unexpected flower")
	}
}
