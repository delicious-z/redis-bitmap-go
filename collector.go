package redis_bitmap_go

import (
	"encoding/json"
	"github.com/redis/go-redis/v9"
)

type collector struct {
	redisApi *redis.ClusterClient

	bitmapName string
	ch1        chan *storePartitionDoneMsg
	ch2        chan *storeCheckMsg
	callback   chan interface{}

	ps    map[uint32]int
	cm    map[uint32]int
	dirty map[uint32]int
	bloom []byte
}

func newCollector(bitmapName string,
	ch1 chan *storePartitionDoneMsg, ch2 chan *storeCheckMsg,
	redisApi *redis.ClusterClient) *collector {
	dirty := map[uint32]int{}
	s := redisApi.Get(ctx, generateKey(bitmapName))
	if s != nil {
		var m meta
		bytes, _ := s.Bytes()
		_ = json.Unmarshal(bytes, &m)
		dirty = m.CardinalityMap
	}

	return &collector{
		ch1:      ch1,
		ch2:      ch2,
		ps:       nil,
		cm:       map[uint32]int{},
		dirty:    dirty,
		bloom:    make([]byte, partitionSizeInBytes, partitionSizeInBytes),
		redisApi: redisApi,
	}
}

func (c *collector) collect() {
	redisApi := c.redisApi

	for flag := false; !flag; {
		select {
		case msg := <-c.ch1:
			delete(c.dirty, msg.partition)
			if msg.list != nil {
				c.cm[msg.partition] = -1 * int(msg.cardinality)
				for _, idx := range msg.list {
					setBit(getOffset(idx), c.bloom)
				}
				err := listBufPool.ReturnObject(ctx, msg.list)
				if err != nil {
					panic(err)
				}
			} else {
				c.cm[msg.partition] = int(msg.cardinality)
				err := byteBufPool.ReturnObject(ctx, msg.bytes)
				if err != nil {
					panic(err)
				}
			}
			if c.judge() {
				flag = true
				break
			}
		case msg := <-c.ch2:
			c.callback = msg.callback
			c.ps = msg.ps
			c.bitmapName = msg.name
			if c.judge() {
				flag = true
				break
			}
		}
	}

	dks := make([]string, 0)
	for partition, _ := range c.dirty {
		dks = append(dks, generateKeyPartition(c.bitmapName, partition))
	}
	redisApi.Del(ctx, dks...)

	m := meta{
		CardinalityMap: c.cm,
		Bloom:          nil,
	}
	metaBytes, _ := json.Marshal(m)
	redisApi.Set(ctx, generateKey(c.bitmapName), metaBytes, 0)
	redisApi.Publish(ctx, "redis.bitmap.meta", metaBytes)
	c.callback <- struct{}{}
}

func (c *collector) judge() bool {
	ps := c.ps
	cm := c.cm
	if ps == nil || cm == nil {
		return false
	}
	return len(ps) == len(cm)
}
