package scache

import (
	"errors"
	. "github.com/karlseguin/expect"
	"strconv"
	"testing"
	"time"
)

type ScacheTests struct {
}

func Test_Scache(t *testing.T) {
	Expectify(new(ScacheTests), t)
}

func (_ ScacheTests) GetsNothing() {
	cache := New(10, time.Minute)
	Expect(cache.Get("nothing")).To.Equal(nil)
}

func (_ ScacheTests) GetsAValue() {
	cache := New(10, time.Minute)
	cache.Set("it's over", 9000)
	Expect(cache.Get("it's over")).To.Equal(9000)
}

func (_ ScacheTests) FetchesAMiss() {
	cache := New(10, time.Minute)
	value, err := cache.Fetch("leto", func(key string) (interface{}, error) {
		return key + " atreides", nil
	})
	Expect(err).To.Equal(nil)
	Expect(value).To.Equal("leto atreides")
	Expect(cache.Get("leto")).To.Equal("leto atreides")
}

func (_ ScacheTests) FetchesAnError() {
	cache := New(10, time.Minute)
	value, err := cache.Fetch("leto", func(key string) (interface{}, error) {
		return "", errors.New("X")
	})
	Expect(err.Error()).To.Equal("X")
	Expect(value).To.Equal(nil)
	Expect(cache.Get("leto")).To.Equal(nil)
}

func (_ ScacheTests) FetchesAHit() {
	cache := New(10, time.Minute)
	cache.Set("leto", "worm")
	value, err := cache.Fetch("leto", func(key string) (interface{}, error) {
		return nil, nil
	})
	Expect(err).To.Equal(nil)
	Expect(value).To.Equal("worm")
}

func (_ ScacheTests) FreesSpace() {
	cache := New(100, time.Minute)
	for i := 0; i < 100; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	cache.Set("overflow", "wow")
	time.Sleep(time.Millisecond) //let the GC run

	Expect(cache.Get("overflow")).To.Equal("wow")
	Expect(cache.Get("1")).To.Equal(nil)
}

func (_ ScacheTests) GCWillRunTwice() {
	cache := New(100, time.Minute)
	for i := 0; i < 100; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	cache.Set("overflow", "wow")
	time.Sleep(time.Millisecond)

	for i := 0; i < 100; i++ {
		cache.Set(strconv.Itoa(i), i)
	}
	cache.Set("overflow", "wow2")
	time.Sleep(time.Millisecond)
	Expect(cache.Get("overflow")).To.Equal("wow2")
	Expect(cache.Get("1")).To.Equal(nil)
}
