# redis-bitmap

redis-bitmap is a solution for store large bitmap (e.g. bitmap range[1-1billion]) in redis.
It implements a simple version of [RoaringBitmap](https://github.com/RoaringBitmap/RoaringBitmap) on Redis.

- high performance read & write
- 99% lower memory cost than native approach in condition of  **sparse** data

## Example


**install**
```
go get -u github.com/delicious-z/redis-bitmap-go
```


**create a RedisBitmapManager**

```
	addrs := []string{
		"192.168.137.108:6380",
		"192.168.137.108:6381",
		"192.168.137.108:6382"}
	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{Addrs: addrs})
	mng := redis_bitmap_go.Create(clusterClient)
```

**write**

```
rb := roaring.New()
rb.AddMany([]uint32{1, 2, 3, 100000, 1000000})
mng.Create("my-bitmap", rb.Iterator())
```

**read**

```
redisBitmap, err := mng.Load("my-bitmap")
if err != nil {
    log.Fatalln(err)
}
println(redisBitmap == nil)
println(redisBitmap.GetBit(1))      // true
println(redisBitmap.GetBit(2))      // true
println(redisBitmap.GetBit(3))      // true
println(redisBitmap.GetBit(100000)) // true
println(redisBitmap.GetBit(100001)) // false
println(redisBitmap.GetBit(100000000)) // false
```

## Why redis-bitmap?

### lower memory cost
In redis, Bitmaps are a set of bit-oriented operations defined on the String type which is treated like a bit vector.
In condition of sparse data, native approach wastes memory. For example, if you store a bitmap of [0,1,0,0..(10 million 0)..1]
in redis, it almost requires 1MB memory.
The situation may be more complex in real world. The bitmap may be dense in some intervals, but sparse in others intervals.
redis-bitmap did something to make it cost less memory:
1. divide the bitmap into many partitions(10 million bit per partition)
![img](https://github.com/delicious-z/redis-bitmap-go/assets/49525320/cdafffcc-6e24-441e-9c94-e5cb99deff13)


3. store the sparse partitions at redis in a compact format
![redis-data-structure](https://github.com/delicious-z/redis-bitmap-go/assets/49525320/35e2ee55-d695-4679-b1bb-ef6d06bfde47)


### high performance write & read

- parallel encode & write
- bloom cache for the compact partition
- pooled int[] & byte[]
- backpressure avoid OOM

**parallel encode**
```go
func (mng *RedisBitmapManager) init() {
    initPool()
    for i := 0; i < runtime.NumCPU(); i++ {
        go mng.encode()
        go mng.storePartitions()
    }
}
```
**bloom-cache**

In scenarios where “GetBit” methods almost always return false, the bloom filter can be very useful.
```go
partition := getPartition(idx)
if rb.cardinalityMap[partition] < 0 { // idx might in the compact partition
    if rb.bloom != nil {
    if !getBit(getOffset(idx), rb.bloom) {
        return false
    }
}
return rb.redisClient.SIsMember(ctx, generateKeyCp(rb.name), idx).Val()
```

**bit-op optimize**

```go
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

```
