package friendlink

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// RandomColor returns a random hex color string that avoids very light or very
// dark colors. The lightness is kept between 35% and 75% so the color remains
// visible on both light and dark backgrounds.
func RandomColor() string {
	for {
		r, _ := rand.Int(rand.Reader, big.NewInt(256))
		g, _ := rand.Int(rand.Reader, big.NewInt(256))
		b, _ := rand.Int(rand.Reader, big.NewInt(256))

		rf := float64(r.Int64())
		gf := float64(g.Int64())
		bf := float64(b.Int64())

		// Approximate relative luminance.
		luminance := 0.2126*rf + 0.7152*gf + 0.0722*bf
		minLight := 0.35 * 255
		maxLight := 0.75 * 255
		if luminance >= minLight && luminance <= maxLight {
			return fmt.Sprintf("#%02x%02x%02x", r.Int64(), g.Int64(), b.Int64())
		}
	}
}
