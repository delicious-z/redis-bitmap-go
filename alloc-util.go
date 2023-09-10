package redis_bitmap_go

var zeroBytes = make([]byte, partitionSizeInBytes, partitionSizeInBytes)

func allocBytes() []byte {
	o, err := byteBufPool.BorrowObject(ctx)
	if err != nil {
		panic(err)
	}
	res := o.([]byte)
	copy(res, zeroBytes)
	return res
}
func allocList() []uint32 {
	res, err := listBufPool.BorrowObject(ctx)
	if err != nil {
		panic(err)
	}
	return res.([]uint32)[:0]
}
