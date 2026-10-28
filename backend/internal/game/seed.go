package game

import (
	"crypto/rand"
	"encoding/binary"
)

// RandomWorldSeed returns a seed in the generator's uint32 range.
func RandomWorldSeed() (int64, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint32(buf[:])), nil
}
