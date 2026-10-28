package gather

import "math"

// tileSize matches the frontend VITE_TILE_SIZE default in frontend/src/game/env.ts.
const tileSize = 32

// inReach allows the player's own tile or an orthogonal neighbor.
func inReach(pixelX, pixelY float64, tileX, tileY int) bool {
	dx := abs(tileX - tileOf(pixelX))
	dy := abs(tileY - tileOf(pixelY))
	if dx == 0 && dy == 0 {
		return true
	}
	return dx+dy == 1
}

func tileOf(pixel float64) int {
	return int(math.Floor(pixel / tileSize))
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
