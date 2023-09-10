package redis_bitmap_go

import (
	"github.com/RoaringBitmap/roaring"
	"math/rand"
)

func generateRandomRb(cm map[uint32]uint32) *roaring.Bitmap {
	rb := roaring.New()
	for p, c := range cm {
		for i := 0; i < int(c); i++ {
			if i == 0 {
				rb.Add(p << 23)
			} else if i == 1 {
				rb.Add((p + 1) * ((1 << 23) - 1))
			} else {
				offset0 := p << 23
				offset1 := rand.Uint32() >> 23
				rb.Add(offset0 + offset1)
			}
		}
	}
	return rb
}
