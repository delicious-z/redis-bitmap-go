package redis_bitmap_go

import (
	"context"
	"encoding/json"
	"github.com/RoaringBitmap/roaring"
	pool "github.com/jolestar/go-commons-pool/v2"
	"github.com/redis/go-redis/v9"
	"sync"
)
import "runtime"

var ctx = context.Background()
var storeCh = make(chan store, 10)
var storePartitionCh = make(chan *storePartitionMsg, 10)

var byteBufPool *pool.ObjectPool
var listBufPool *pool.ObjectPool

//var redisClient = redis.NewClient(&redis.Options{
//	Addr:     "192.168.137.108:6379",
//	Password: "",
//	DB:       0,
//	Protocol: 3,
//})

type RedisBitmapManager struct {
	redisClient *redis.ClusterClient
}

func Create(redisClusterClient *redis.ClusterClient) *RedisBitmapManager {
	mng := &RedisBitmapManager{redisClient: redisClusterClient}
	mng.init()
	return mng
}

type RedisBitmap struct {
	redisClient    *redis.ClusterClient
	name           string
	mu             sync.RWMutex
	cardinalityMap map[uint32]int
	bloom          []byte
}
type meta struct {
	Name           string         `json:"name"`
	CardinalityMap map[uint32]int `json:"cardinalityMap"`
	Bloom          []byte         `json:"bloom"`
}

func (rb *RedisBitmap) GetBit(idx uint32) bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	partition := getPartition(idx)
	_, ok := rb.cardinalityMap[partition]
	if !ok {
		return false
	}
	if rb.cardinalityMap[partition] < 0 {
		if rb.bloom != nil {
			if !getBit(getOffset(idx), rb.bloom) {
				return false
			}
		}
		return rb.redisClient.SIsMember(ctx, generateKeyCp(rb.name), idx).Val()
	}

	return false
}
func (rb *RedisBitmap) Cardinality() uint32 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var res uint32 = 0
	for _, v := range rb.cardinalityMap {
		if v < 0 {
			res += uint32(-1 * v)
		} else {
			res += uint32(v)
		}
	}

	return res
}
func (rb *RedisBitmap) init() {
	sub := rb.redisClient.Subscribe(ctx, "redis.bitmap.meta")
	for msg := range sub.Channel() {
		payload := msg.Payload
		m := meta{}
		_ = json.Unmarshal([]byte(payload), &m)
		if m.Name != rb.name {
			continue
		}
		rb.mu.Lock()
		rb.cardinalityMap = m.CardinalityMap
		bloom, _ := rb.redisClient.Get(ctx, generateKeyBloom(rb.name)).Bytes()
		rb.bloom = bloom
		rb.mu.Unlock()
	}
}

func (mng *RedisBitmapManager) Load(name string) (*RedisBitmap, error) {
	redisApi := mng.redisClient
	s := redisApi.Get(ctx, generateKey(name))
	bytes, _ := s.Bytes()
	var m meta
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		return nil, err
	}
	s = redisApi.Get(ctx, generateKeyBloom(name))
	bloom, _ := s.Bytes()
	rb := &RedisBitmap{
		redisClient:    redisApi,
		name:           name,
		cardinalityMap: m.CardinalityMap,
		bloom:          bloom,
	}
	go rb.init()
	return rb, nil
}
func (mng *RedisBitmapManager) Create(name string, it roaring.IntIterable) (*RedisBitmap, error) {
	callback := make(chan interface{}, 1)
	msg := store{
		name:     name,
		it:       it,
		callback: callback,
	}
	storeCh <- msg
	<-callback
	return mng.Load(name)
}

type store struct {
	name     string
	it       roaring.IntIterable
	callback chan interface{}
}
type storePartitionMsg struct {
	name      string
	partition uint32
	list      []uint32
	bytes     []byte
	ch        chan *storePartitionDoneMsg
}
type storePartitionDoneMsg struct {
	name        string
	partition   uint32
	cardinality uint32
	spend       int64
	list        []uint32
	bytes       []byte
}
type storeCheckMsg struct {
	name     string
	ps       map[uint32]int
	callback chan interface{}
}

func (mng *RedisBitmapManager) init() {
	initPool()
	for i := 0; i < runtime.NumCPU(); i++ {
		go mng.encode()
		go mng.storePartitions()
	}
}
func initPool() {
	byteBufFactory := pool.NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			return make([]byte, partitionSizeInBytes, partitionSizeInBytes), nil
		})
	listBufFactory := pool.NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			return make([]uint32, 0, 10000), nil
		})

	config := &pool.ObjectPoolConfig{
		MaxTotal:           10,
		MaxIdle:            5,
		MinIdle:            3,
		BlockWhenExhausted: true,
	}
	byteBufPool = pool.NewObjectPool(ctx, byteBufFactory, config)
	listBufPool = pool.NewObjectPool(ctx, listBufFactory, config)

}
