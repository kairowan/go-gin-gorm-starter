package cache

import (
	"context"
	"encoding/json"
	"time"
)

func GetOrLoadJSON[T any](
	c *Cache,
	ctx context.Context,
	key string,
	ttl time.Duration,
	load func(ctx context.Context) (*T, error),
) (*T, error) {
	b, err := c.GetOrLoad(ctx, key, ttl, func(ctx context.Context) ([]byte, error) {
		v, e := load(ctx)
		if e != nil {
			// 负缓存以避免击穿（自行按需开启）
			// if errors.Is(e, gorm.ErrRecordNotFound) { return []byte("null"), nil }
			return nil, e
		}
		return json.Marshal(v)
	})
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return nil, nil
	}
	var out T
	if e := json.Unmarshal(b, &out); e != nil {
		return nil, e
	}
	return &out, nil
}
