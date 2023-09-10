package redis_bitmap_go

func (mng *RedisBitmapManager) encode() {
	redisApi := mng.redisClient
	for msg := range storeCh {
		it := msg.it
		name := msg.name
		callback := msg.callback
		redisApi.Del(ctx, generateKeyCp(name))

		var a []uint32 = nil
		var bytes []byte = nil
		ps := map[uint32]int{}

		collectCh := make(chan *storePartitionDoneMsg)
		checkCh := make(chan *storeCheckMsg)
		for partition := uint32(1000); it.HasNext(); {
			idx := it.Next()
			if getPartition(idx) != partition {
				msg := &storePartitionMsg{
					name:      name,
					partition: partition,
					list:      a,
					bytes:     bytes,
					ch:        collectCh,
				}
				storePartitionCh <- msg
				a = nil
				bytes = nil
			}
			partition = getPartition(idx)
			ps[partition] = 1
			if a == nil && bytes == nil {
				a = allocList()
			} else if len(a) >= 5000 {
				bytes = allocBytes()
				for _, _idx := range a {
					setBit(getOffset(_idx), bytes)
				}
				err := listBufPool.ReturnObject(ctx, a)
				if err != nil {
					panic(err)
				}
				a = nil
			}

			if a != nil {
				a = append(a, idx)
			} else {
				setBit(getOffset(idx), bytes)
			}

			if !it.HasNext() {
				msg := &storePartitionMsg{
					name:      name,
					partition: partition,
					list:      a,
					bytes:     bytes,
					ch:        collectCh,
				}
				storePartitionCh <- msg
			}

		}

		go newCollector(name, collectCh, checkCh, redisApi).collect()
		checkCh <- &storeCheckMsg{
			name:     name,
			ps:       ps,
			callback: callback,
		}
	}
}
