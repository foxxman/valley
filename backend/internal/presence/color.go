package presence

import "crypto/rand"

// markerPalette stays clear of the minimap terrain: grass, stone, water.
var markerPalette = []int{
	0xFFEB3B,
	0xFF9800,
	0xF44336,
	0xEC407A,
	0xAB47BC,
	0x7E57C2,
	0x5C6BC0,
	0x42A5F5,
	0xFF7043,
	0xFFFFFF,
	0xFFEE58,
	0x8D6E63,
}

// pickColor chooses a palette entry that nobody else in the room is using.
// When the palette is exhausted it reuses a color.
func pickColor(used map[int]struct{}) int {
	start := randomIndex(len(markerPalette))
	for i := 0; i < len(markerPalette); i++ {
		color := markerPalette[(start+i)%len(markerPalette)]
		if _, taken := used[color]; !taken {
			return color
		}
	}
	return markerPalette[start]
}

func randomIndex(n int) int {
	if n <= 1 {
		return 0
	}
	var buf [1]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	return int(buf[0]) % n
}
