package calendar

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// MemcachedCache wraps gomemcache to implement the Cache interface.
type MemcachedCache struct {
	client *memcache.Client
}

func NewMemcachedCache(addr string) *MemcachedCache {
	return &MemcachedCache{client: memcache.New(addr)}
}

func (m *MemcachedCache) Get(key string) ([]byte, error) {
	item, err := m.client.Get(key)
	if err != nil {
		return nil, err
	}
	return item.Value, nil
}

func (m *MemcachedCache) Set(key string, value []byte, ttl time.Duration) error {
	return m.client.Set(&memcache.Item{
		Key:        key,
		Value:      value,
		Expiration: int32(ttl.Seconds()),
	})
}

// NullCache is a no-op cache for when memcached is not configured.
type NullCache struct{}

func (n *NullCache) Get(key string) ([]byte, error) {
	return nil, memcache.ErrCacheMiss
}

func (n *NullCache) Set(key string, value []byte, ttl time.Duration) error {
	return nil
}
