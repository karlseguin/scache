// a simple cache
package scache

import (
	"hash/fnv"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	BUCKET_COUNT = 16 //must be a power of 2
	bucket_mask  = uint32(BUCKET_COUNT - 1)
)

type Item struct {
	Expires time.Time
	Value   interface{}
}

type Scache struct {
	gcing   uint32
	max     int32
	count   int32
	ttl     time.Duration
	buckets []*Bucket
}

type Bucket struct {
	sync.RWMutex
	count  int
	lookup map[string]*Item
}

func New(max int32, ttl time.Duration) *Scache {
	scache := &Scache{
		count:   0,
		max:     max,
		ttl:     ttl,
		buckets: make([]*Bucket, BUCKET_COUNT),
	}
	for i := 0; i < BUCKET_COUNT; i++ {
		scache.buckets[i] = &Bucket{
			lookup: make(map[string]*Item),
		}
	}
	return scache
}

func (sc *Scache) Get(key string) interface{} {
	bucket := sc.getBucket(key)
	item := bucket.Get(key)
	if item == nil {
		return nil
	}
	if item.Expires.After(time.Now()) {
		return item.Value
	}
	if bucket.Remove(key) == true {
		atomic.AddInt32(&sc.count, -1)
	}
	return nil
}

func (sc *Scache) Fetch(key string, miss func(key string) (interface{}, error)) (interface{}, error) {
	bucket := sc.getBucket(key)
	item := bucket.Get(key)
	if item != nil && item.Expires.After(time.Now()) {
		return item.Value, nil
	}
	value, err := miss(key)
	if err != nil {
		return nil, err
	}
	if value != nil {
		sc.Set(key, value)
	}
	return value, nil
}

func (sc *Scache) Set(key string, value interface{}) {
	item := &Item{
		Expires: time.Now().Add(sc.ttl),
		Value:   value,
	}
	if sc.getBucket(key).Set(key, item) == true {
		if atomic.AddInt32(&sc.count, 1) >= sc.max && atomic.CompareAndSwapUint32(&sc.gcing, 0, 1) {
			go sc.gc()
		}
	}
}

func (sc *Scache) Remove(key string) bool {
	bucket := sc.getBucket(key)
	if bucket.Remove(key) == true {
		atomic.AddInt32(&sc.count, -1)
		return true
	}
	return false
}

func (sc *Scache) Clear() {
	for _, bucket := range sc.buckets {
		lookup := make(map[string]*Item)
		bucket.Lock()
		bucket.count = 0
		bucket.lookup = lookup
		bucket.Unlock()
	}
}

func (sc *Scache) gc() {
	defer atomic.StoreUint32(&sc.gcing, 0)
	freed := int32(0)
	for i := 0; i < BUCKET_COUNT; i++ {
		if sc.buckets[i].gc() {
			freed++
		}
	}
	atomic.AddInt32(&sc.count, -freed)
}

func (b *Bucket) Get(key string) *Item {
	b.RLock()
	value := b.lookup[key]
	b.RUnlock()
	return value
}

func (b *Bucket) Remove(key string) bool {
	b.Lock()
	_, exists := b.lookup[key]
	delete(b.lookup, key)
	b.Unlock()
	return exists
}

func (b *Bucket) Set(key string, item *Item) bool {
	b.Lock()
	_, exists := b.lookup[key]
	b.lookup[key] = item
	b.Unlock()
	return !exists
}

func (b *Bucket) gc() bool {
	visited, now := 0, time.Now()
	ok, oe := "", now.Add(time.Hour*8765)

	b.RLock()
	for key, item := range b.lookup {
		if item.Expires.Before(oe) {
			oe = item.Expires
			ok = key
			if oe.Before(now) {
				break
			}
		}
		if visited++; visited == 10 {
			break
		}
	}
	b.RUnlock()

	if len(ok) == 0 {
		return false
	}

	b.Lock()
	delete(b.lookup, ok)
	b.Unlock()
	return true
}

func (sc *Scache) getBucket(key string) *Bucket {
	h := fnv.New32a()
	h.Write(str2bytes(&key))
	return sc.buckets[h.Sum32()&bucket_mask]
}

func str2bytes(s *string) []byte {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(s))
	sh.Len = len(*s)
	sh.Cap = sh.Len
	return *(*[]byte)(unsafe.Pointer(sh))
}
