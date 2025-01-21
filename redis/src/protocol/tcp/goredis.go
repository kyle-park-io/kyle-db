package tcp

import (
	"context"

	"kyle-redis/logger"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func callRedisClient(rdb *redis.Client) error {
	err := rdb.Set(ctx, "key", "value", 0).Err()
	if err != nil {
		return err
	}

	val, err := rdb.Get(ctx, "key").Result()
	if err != nil {
		return err
	}
	logger.Log.Infoln("key", val)

	val2, err := rdb.Get(ctx, "key2").Result()
	if err == redis.Nil {
		logger.Log.Infoln("key2 does not exist")
	} else if err != nil {
		return err
	} else {
		logger.Log.Infoln("key2", val2)
	}

	return nil
}
