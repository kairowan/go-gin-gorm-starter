package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"time"
)

type Cache struct {
	RDB *redis.Client
	sf  singleflight.Group
}

func New(addr, pass string, db int) *Cache {
	return &Cache{
		RDB: redis.NewClient(&redis.Options{Addr: addr, Password: pass, DB: db}),
	}
}

func (c *Cache) GetOrLoad(ctx context.Context, key string, ttl time.Duration, load func(context.Context) ([]byte, error)) ([]byte, error) {
	// 先读缓存
	if b, err := c.RDB.Get(ctx, key).Bytes(); err == nil {
		return b, nil
	}
	// single flight 合并回源
	v, err, _ := c.sf.Do(key, func() (any, error) {
		b, e := load(ctx)
		if e != nil {
			return nil, e
		}
		_ = c.RDB.Set(ctx, key, b, ttl).Err()
		return b, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}
