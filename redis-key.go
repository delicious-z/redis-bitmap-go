package redis_bitmap_go

import "fmt"

func generateKey(name string) string {
	return fmt.Sprintf("redis.bitmap:%s", name)
}

func generateKeyCp(name string) string {
	return fmt.Sprintf("redis.bitmap.cp:%s", name)
}

func generateKeyBloom(name string) string {
	return fmt.Sprintf("redis.bitmap.bloom:%s", name)
}

func generateKeyPartition(name string, partition uint32) string {
	return fmt.Sprintf("%s:%d", name, partition)
}
