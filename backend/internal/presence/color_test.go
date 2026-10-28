package presence

import "testing"

func TestPickColorAvoidsUsed(t *testing.T) {
	used := map[int]struct{}{}
	for _, color := range markerPalette[:len(markerPalette)-1] {
		used[color] = struct{}{}
	}
	if got := pickColor(used); got != markerPalette[len(markerPalette)-1] {
		t.Fatalf("color %06x", got)
	}
}

func TestPickColorReusesWhenPaletteIsFull(t *testing.T) {
	used := map[int]struct{}{}
	for _, color := range markerPalette {
		used[color] = struct{}{}
	}
	got := pickColor(used)
	if _, ok := used[got]; !ok {
		t.Fatalf("color %06x is outside the palette", got)
	}
}
