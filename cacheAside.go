package cacheAside

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

var (
	//batchSize is the number of keys to be fetched from db in one batch
	batchSize = 1000
	//defaultCacheExpire is the default expire time of cache
	defaultCacheExpire = time.Hour * 24 * 7
	//defaultMissCacheExpire is the default expire time of miss cache
	defaultMissCacheExpire = time.Minute
	//defaultCleanInterval is the default clean interval of cache
	defaultCleanInterval = time.Hour
)

var (
	//ErrNotFound is the error returned when the key is not found in cache and db
	ErrNotFound = errors.New("not found")
)

var (
	// c is the cache aside instance
	c *cacheAside
	// o is the sync.Once instance
	o sync.Once
)

// Init initializes the cache aside instance
func Init(opt *Option) {
	o.Do(func() {
		c = newCacheAside(opt)
	})
}

// cacheAside is the cache aside instance
type cacheAside struct {
	// cache is the cache instance
	cache *cache.Cache
	// sfKey is the single flight key
	sfKey sync.Map
	// unstable is the unstable instance
	unstable Unstable
	// sfGroup is the single flight group
	sfGroup singleflight.Group
}

// Option is the option of cache aside
type Option struct {
	// BatchSize is the number of keys to be fetched from db in one batch
	BatchSize int
	// DefaultCacheExpire is the default expire time of cache
	DefaultCacheExpire time.Duration
	// MissCacheExpire is the default expire time of miss cache
	MissCacheExpire time.Duration
	// CleanInterval is the default clean interval of cache
	CleanInterval time.Duration
}

// newCacheAside creates a new cache aside instance
func newCacheAside(opt *Option) *cacheAside {
	if opt != nil {
		batchSize = opt.BatchSize
		defaultCacheExpire = opt.DefaultCacheExpire
		defaultMissCacheExpire = opt.MissCacheExpire
		defaultCleanInterval = opt.CleanInterval
	}
	return &cacheAside{
		cache:    cache.New(defaultCacheExpire, defaultCleanInterval),
		sfKey:    sync.Map{},
		unstable: NewUnstable(0.05),
		sfGroup:  singleflight.Group{},
	}
}

// notFoundPlaceHolder is the placeholder of not found key
type notFoundPlaceHolder struct{}

// singleFlightKey returns the single flight key
func singleFlightKey(t any) string {
	k := reflect.TypeOf(t).String()
	v, _ := c.sfKey.LoadOrStore(k, "singleFlight:"+k+":")
	return v.(string)
}

// Get gets the value from cache, if not found, fetch from db then add to cache
func Get[U any](key string, dbFetch func(string) (U, bool, error)) (res U, err error) {
	v, ok := c.cache.Get(key)
	if ok {
		switch v.(type) {
		case notFoundPlaceHolder:
			err = ErrNotFound
		default:
			res = v.(U)
		}
		return
	}
	var rr any
	rr, err, _ = c.sfGroup.Do(singleFlightKey(res)+key, func() (r any, e error) {
		var notFound bool
		r, notFound, e = dbFetch(key)
		if e == nil && notFound {
			e = ErrNotFound
		}
		return
	})
	res = rr.(U)
	var miss any
	if err != nil && err != ErrNotFound {
		return
	}
	if err == ErrNotFound {
		miss = notFoundPlaceHolder{}
	} else {
		miss = res
	}
	addCacheAnyItem(key, miss)
	return
}

// addCacheAnyItem add the item to cache
func addCacheAnyItem(k string, u any) {
	expire := defaultCacheExpire
	switch u.(type) {
	case notFoundPlaceHolder:
		expire = defaultMissCacheExpire
	}
	c.cache.Set(k, u, c.unstable.AroundDuration(expire))
}

// cacheAnyThings caches any things
func cacheAnyThings[T any](keys []string) (res map[string]T) {
	l := len(keys)
	if l == 0 {
		return
	}
	ress := make(map[string]any, len(keys))
	for _, id := range keys {
		v, ok := c.cache.Get(id)
		if ok {
			ress[id] = v
		}
	}
	res = make(map[string]T, len(keys))
	for _, key := range keys {
		v, ok := ress[key]
		if !ok {
			continue
		}
		switch v.(type) {
		case notFoundPlaceHolder:
			delete(ress, key)
		case T:
			res[key] = v.(T)
		default:
			panic("cache aside type error")
		}
	}
	return
}

// MultiGet gets the values from cache, if not found, fetch from db then add to cache
func MultiGet[U any](keys []string, dbFetch func(id []string) (map[string]U, error)) (res map[string]U, err error) {
	if len(keys) == 0 {
		return
	}
	res = cacheAnyThings[U](keys)
	var miss []string
	for _, key := range keys {
		if _, ok := res[key]; !ok {
			miss = append(miss, key)
		}
	}
	missLen := len(miss)
	if missLen == 0 {
		return
	}
	missData := make(map[string]U, missLen)
	var mutex sync.Mutex
	group, _ := errgroup.WithContext(context.Background())
	if missLen > 10 {
		group.SetLimit(10)
	}
	var run = func(ms []string) {
		group.Go(func() (err error) {
			data, err := dbFetch(ms)
			mutex.Lock()
			for k, v := range data {
				missData[k] = v
			}
			mutex.Unlock()
			return
		})
	}
	var (
		i int
		n = missLen / batchSize
	)
	for i = 0; i < n; i++ {
		run(miss[i*n : (i+1)*n])
	}
	if len(miss[i*n:]) > 0 {
		run(miss[i*n:])
	}
	err = group.Wait()
	if res == nil {
		res = make(map[string]U, len(keys))
	}
	for k, v := range missData {
		res[k] = v
	}
	if err != nil {
		return
	}
	cacheData := make(map[string]any, len(missData))
	for k, v := range missData {
		cacheData[k] = v
	}
	for _, key := range miss {
		_, ok := res[key]
		if !ok {
			cacheData[key] = notFoundPlaceHolder{}
		}
	}
	addCacheAnyItems(cacheData)
	return
}

// addCacheAnyItems add the items to cache
func addCacheAnyItems(values map[string]any) {
	if len(values) == 0 {
		return
	}
	for id, val := range values {
		expire := defaultCacheExpire
		switch val.(type) {
		case notFoundPlaceHolder:
			expire = defaultMissCacheExpire
		}
		c.cache.Set(id, val, c.unstable.AroundDuration(expire))
	}
	return
}

// Delete deletes the key from cache
func Delete(k ...string) {
	for _, v := range k {
		c.cache.Delete(v)
	}
}

func debugExist(k string) bool {
	_, ok := c.cache.Get(k)
	return ok
}

func debugInit(opt *Option) {
	c = newCacheAside(opt)
}
