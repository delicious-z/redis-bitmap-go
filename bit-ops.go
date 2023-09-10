package redis_bitmap_go

const (
	shift                = 23
	partitionSize        = 1 << shift
	partitionSizeInBytes = partitionSize >> 3
)

func getPartition(idx uint32) uint32 {
	return idx >> shift
}

func setBit(offset uint32, bytes []byte) {
	j := offset >> 3
	k := 8 - 1 - (offset & (8 - 1))
	bytes[j] |= 1 << k
}
func getBit(offset uint32, bytes []byte) bool {
	j := offset >> 3
	k := 8 - 1 - (offset & (8 - 1))
	return bytes[j]&(1<<k) != 0
}

func bitCount(bytes []byte) uint32 {
	var res uint32 = 0
	for _, b := range bytes {
		for i := 0; i < 8; i++ {
			if b>>i&1 != 0 {
				res++
			}
		}
	}
	return res
}

func getOffset(idx uint32) uint32 {
	return idx & (partitionSize - 1)
}
