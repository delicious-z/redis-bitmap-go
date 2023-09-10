package redis_bitmap_go

import (
	"strconv"
)

func (mng *RedisBitmapManager) storePartitions() {
	redisApi := mng.redisClient
	sp := func(msg *storePartitionMsg) {
		name := msg.name
		partition := msg.partition
		bytes := msg.bytes
		k := generateKeyPartition(name, partitionSize)
		c := bitCount(bytes)

		redisApi.Set(ctx, k, bytes, 0)

		done := &storePartitionDoneMsg{
			name:        name,
			partition:   partition,
			cardinality: c,
			spend:       0,
			list:        nil,
			bytes:       bytes,
		}
		msg.ch <- done
	}
	spcp := func(msg *storePartitionMsg) {
		name := msg.name
		partition := msg.partition
		list := msg.list
		k := generateKeyCp(name)
		c := len(list)
		strlist := make([]string, c, c)
		for i, x := range list {
			strlist[i] = strconv.Itoa(int(x))
		}
		redisApi.SAdd(ctx, k, strlist)

		done := &storePartitionDoneMsg{
			name:        name,
			partition:   partition,
			cardinality: uint32(c),
			spend:       0,
			list:        list,
			bytes:       nil,
		}
		msg.ch <- done

	}

	for msg := range storePartitionCh {
		if msg.bytes != nil {
			sp(msg)
		} else if msg.list != nil {
			spcp(msg)
		}
	}
}
