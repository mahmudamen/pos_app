package redisx

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func Open(ctx context.Context, address string) (*redis.Client, error) {
	opt, err := redis.ParseURL(address)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}
